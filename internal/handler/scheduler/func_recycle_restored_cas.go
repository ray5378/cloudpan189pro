package scheduler

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/tickstep/cloudpan189-api/cloudpan"
	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	appsessionSvi "github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	casrecordSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrecord"
	casrestoreSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrestore"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"go.uber.org/zap"
)

const (
	recycleRetryMarker       = "[retry-once]"
	fallbackRecycleInterval  = 4 * time.Hour
	fallbackRecycleMaxJitter = 30 * time.Minute
)

type RecycleRestoredCASScheduler struct {
	casRecordService  casrecordSvi.Service
	appSessionService appsessionSvi.Service
	mountPointService mountpointSvi.Service
	cloudTokenService cloudtokenSvi.Service
	quit              chan struct{}
	done              chan struct{}
	running           bool
	rng               *rand.Rand
	nextFallbackRunAt time.Time
}

func NewRecycleRestoredCASScheduler(casRecordService casrecordSvi.Service, appSessionService appsessionSvi.Service, mountPointService mountpointSvi.Service, cloudTokenService cloudtokenSvi.Service) Scheduler {
	return &RecycleRestoredCASScheduler{
		casRecordService:  casRecordService,
		appSessionService: appSessionService,
		mountPointService: mountPointService,
		cloudTokenService: cloudTokenService,
		quit:              make(chan struct{}),
		done:              make(chan struct{}),
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *RecycleRestoredCASScheduler) Start(ctx appctx.Context) error {
	if s.running {
		return ErrSchedulerRunning
	}
	s.running = true
	s.nextFallbackRunAt = s.calcNextFallbackRun(time.Now())
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		s.doJob(ctx)
		for {
			select {
			case <-ticker.C:
				s.doJob(ctx)
			case <-s.quit:
				return
			}
		}
	}()
	return nil
}

func (s *RecycleRestoredCASScheduler) Stop() {
	if !s.running {
		return
	}
	close(s.quit)
	<-s.done
	s.running = false
}

func (s *RecycleRestoredCASScheduler) doJob(ctx appctx.Context) {
	now := time.Now()
	s.doRecordedRecycle(ctx, now)
	s.doFallbackRecycle(ctx, now)
}

func (s *RecycleRestoredCASScheduler) doRecordedRecycle(ctx appctx.Context, now time.Time) {
	list, err := s.casRecordService.ListDueRecycle(ctx, now, 50)
	if err != nil {
		ctx.Error("查询到期CAS恢复文件失败")
		return
	}
	for _, record := range list {
		if record == nil || strings.TrimSpace(record.RestoredFileID) == "" {
			continue
		}
		if record.RestoreStatus == models.CasRestoreStatusRecycling {
			settled, settleErr := s.handleStaleRecycling(ctx, record)
			if settleErr != nil {
				_ = s.casRecordService.Update(ctx, record.ID, map[string]any{"restore_status": models.CasRestoreStatusFailed, "last_error": settleErr.Error()})
				ctx.Error("处理卡住的CAS回收记录失败", zap.Int64("record_id", record.ID), zap.Error(settleErr))
				continue
			}
			if settled {
				continue
			}
		}
		_ = s.casRecordService.Update(ctx, record.ID, map[string]any{"restore_status": models.CasRestoreStatusRecycling})
		if err := s.recycleOne(ctx, record); err != nil {
			exists, checkErr := s.restoredFileStillExists(ctx, record)
			if checkErr == nil && !exists {
				s.markRecycled(ctx, record.ID)
				ctx.Info("CAS恢复文件后置步骤失败但文件已删除，按成功收口", zap.Int64("record_id", record.ID), zap.Error(err))
				continue
			}
			if checkErr != nil {
				ctx.Error("回收到期CAS恢复文件失败，且删除结果复核失败", zap.Int64("record_id", record.ID), zap.Error(err), zap.Error(checkErr))
				_ = s.casRecordService.Update(ctx, record.ID, map[string]any{"restore_status": models.CasRestoreStatusFailed, "last_error": err.Error() + " | verify: " + checkErr.Error()})
				continue
			}
			_ = s.casRecordService.Update(ctx, record.ID, map[string]any{"restore_status": models.CasRestoreStatusFailed, "last_error": err.Error()})
			ctx.Error("回收到期CAS恢复文件失败", zap.Int64("record_id", record.ID), zap.Error(err))
			continue
		}
		s.markRecycled(ctx, record.ID)
	}
}

func (s *RecycleRestoredCASScheduler) doFallbackRecycle(ctx appctx.Context, now time.Time) {
	retentionHours := shared.SettingAddition.CasRestoreRetentionHours
	if retentionHours <= 0 {
		s.nextFallbackRunAt = time.Time{}
		return
	}
	if s.nextFallbackRunAt.IsZero() {
		s.nextFallbackRunAt = s.calcNextFallbackRun(now)
		return
	}
	if now.Before(s.nextFallbackRunAt) {
		return
	}
	cutoff := now.Add(-time.Duration(retentionHours) * time.Hour)
	if err := s.runFallbackRecycleSweep(ctx, cutoff); err != nil {
		ctx.Error("CAS恢复文件兜底扫描清理失败", zap.Error(err), zap.Time("cutoff", cutoff))
	} else {
		ctx.Info("CAS恢复文件兜底扫描清理完成", zap.Time("cutoff", cutoff))
	}
	s.nextFallbackRunAt = s.calcNextFallbackRun(now)
}

func (s *RecycleRestoredCASScheduler) calcNextFallbackRun(now time.Time) time.Time {
	jitter := time.Duration(0)
	if s.rng != nil {
		jitter = time.Duration(s.rng.Int63n(int64(fallbackRecycleMaxJitter)))
	}
	return now.Add(fallbackRecycleInterval + jitter)
}

func (s *RecycleRestoredCASScheduler) runFallbackRecycleSweep(ctx appctx.Context, cutoff time.Time) error {
	var firstErr error

	personTokenID := shared.SettingAddition.CasPersonTargetTokenId
	personRootID := strings.TrimSpace(shared.SettingAddition.CasPersonTargetFolderId)
	if personTokenID > 0 && personRootID != "" {
		session, err := s.appSessionService.GetByTokenID(ctx, personTokenID)
		if err != nil {
			firstErr = pickFirstErr(firstErr, fmt.Errorf("加载个人兜底清理session失败: %w", err))
		} else {
			panClient, err := s.newPanClientBySession(ctx, session)
			if err != nil {
				firstErr = pickFirstErr(firstErr, fmt.Errorf("创建个人兜底清理panClient失败: %w", err))
			} else {
				deletedAny, err := s.sweepExpiredPersonFiles(ctx, panClient, personRootID, personRootID, cutoff)
				if err != nil {
					firstErr = pickFirstErr(firstErr, fmt.Errorf("个人兜底清理失败: %w", err))
				}
				if deletedAny {
					if apiErr := panClient.RecycleClear(0); apiErr != nil {
						firstErr = pickFirstErr(firstErr, fmt.Errorf("清空个人回收站失败: %w", apiErr))
					}
				}
			}
		}
	}

	familyTokenID := shared.SettingAddition.CasFamilyTargetTokenId
	familyID := strings.TrimSpace(shared.SettingAddition.CasFamilyTargetFamilyId)
	familyRootID := strings.TrimSpace(shared.SettingAddition.CasFamilyTargetFolderId)
	if familyTokenID > 0 && familyID != "" && familyRootID != "" {
		session, err := s.appSessionService.GetByTokenID(ctx, familyTokenID)
		if err != nil {
			firstErr = pickFirstErr(firstErr, fmt.Errorf("加载家庭兜底清理session失败: %w", err))
		} else {
			panClient, err := s.newPanClientBySession(ctx, session)
			if err != nil {
				firstErr = pickFirstErr(firstErr, fmt.Errorf("创建家庭兜底清理panClient失败: %w", err))
			} else {
				cloudToken, err := s.cloudTokenService.Query(ctx, familyTokenID)
				if err != nil {
					firstErr = pickFirstErr(firstErr, fmt.Errorf("加载家庭兜底清理cloudToken失败: %w", err))
				} else {
					refAccessToken, err := casrestoreSvi.BuildRefSDKAccessToken(session, cloudToken)
					if err != nil {
						firstErr = pickFirstErr(firstErr, fmt.Errorf("构建家庭兜底清理accessToken失败: %w", err))
					} else {
						deletedAny, err := s.sweepExpiredFamilyFiles(ctx, panClient, refAccessToken, familyID, familyRootID, familyRootID, cutoff)
						if err != nil {
							firstErr = pickFirstErr(firstErr, fmt.Errorf("家庭兜底清理失败: %w", err))
						}
						if deletedAny {
							if err := casrestoreSvi.ClearFamilyRecycleByAccessToken(refAccessToken, familyID); err != nil {
								firstErr = pickFirstErr(firstErr, fmt.Errorf("清空家庭回收站失败: %w", err))
							}
						}
					}
				}
			}
		}
	}

	return firstErr
}

func (s *RecycleRestoredCASScheduler) sweepExpiredPersonFiles(ctx appctx.Context, panClient *cloudpan.PanClient, currentID, rootID string, cutoff time.Time) (bool, error) {
	return s.sweepPersonDir(ctx, panClient, currentID, rootID, cutoff)
}

func (s *RecycleRestoredCASScheduler) sweepPersonDir(ctx appctx.Context, panClient *cloudpan.PanClient, currentID, rootID string, cutoff time.Time) (bool, error) {
	param := cloudpan.NewAppFileListParam()
	param.FileId = currentID
	param.PageSize = 200
	res, apiErr := panClient.AppGetAllFileList(param)
	if apiErr != nil {
		return false, apiErr
	}
	var (
		deletedAny bool
		firstErr   error
	)
	if res != nil {
		for _, item := range res.FileList {
			if item == nil {
				continue
			}
			if item.IsFolder {
				childDeleted, err := s.sweepPersonDir(ctx, panClient, item.FileId, rootID, cutoff)
				if err != nil {
					ctx.Error("个人兜底清理递归扫描目录失败", zap.String("folder_id", item.FileId), zap.String("folder_name", item.FileName), zap.Error(err))
					firstErr = pickFirstErr(firstErr, err)
					continue
				}
				if childDeleted {
					deletedAny = true
				}
				continue
			}
			effectiveTime, sourceLabel, err := s.resolveEffectiveExpireBaseTime(ctx, item.FileId, item.CreateTime)
			if err != nil {
				ctx.Error("个人兜底清理解析时间失败", zap.String("file_id", item.FileId), zap.String("file_name", item.FileName), zap.String("create_time", item.CreateTime), zap.Error(err))
				firstErr = pickFirstErr(firstErr, err)
				continue
			}
			if effectiveTime.After(cutoff) {
				continue
			}
			ctx.Info("个人兜底清理命中过期文件", zap.String("file_id", item.FileId), zap.String("file_name", item.FileName), zap.String("time_source", sourceLabel), zap.Time("effective_time", effectiveTime), zap.Time("cutoff", cutoff))
			ok, delErr := panClient.AppDeleteFile([]string{item.FileId})
			if delErr != nil {
				ctx.Error("个人兜底清理删除过期文件失败", zap.String("file_id", item.FileId), zap.String("file_name", item.FileName), zap.Error(delErr))
				firstErr = pickFirstErr(firstErr, delErr)
				continue
			}
			if !ok {
				err := fmt.Errorf("删除个人过期文件失败: %s", item.FileId)
				ctx.Error("个人兜底清理删除过期文件失败", zap.String("file_id", item.FileId), zap.String("file_name", item.FileName), zap.Error(err))
				firstErr = pickFirstErr(firstErr, err)
				continue
			}
			deletedAny = true
		}
	}
	if currentID != rootID {
		empty, err := s.isPersonFolderEmpty(panClient, currentID)
		if err != nil {
			firstErr = pickFirstErr(firstErr, err)
		} else if empty {
			ok, delErr := panClient.AppDeleteFile([]string{currentID})
			if delErr != nil {
				firstErr = pickFirstErr(firstErr, delErr)
			} else if !ok {
				firstErr = pickFirstErr(firstErr, fmt.Errorf("删除个人空目录失败: %s", currentID))
			} else {
				deletedAny = true
			}
		}
	}
	return deletedAny, firstErr
}

func (s *RecycleRestoredCASScheduler) sweepExpiredFamilyFiles(ctx appctx.Context, panClient *cloudpan.PanClient, refAccessToken, familyID, currentID, rootID string, cutoff time.Time) (bool, error) {
	var fid int64
	fmt.Sscan(familyID, &fid)
	param := cloudpan.NewAppFileListParam()
	param.FamilyId = fid
	param.FileId = currentID
	param.PageSize = 200
	res, apiErr := panClient.AppGetAllFileList(param)
	if apiErr != nil {
		return false, apiErr
	}
	var (
		deletedAny bool
		firstErr   error
	)
	if res != nil {
		for _, item := range res.FileList {
			if item == nil {
				continue
			}
			if item.IsFolder {
				childDeleted, err := s.sweepExpiredFamilyFiles(ctx, panClient, refAccessToken, familyID, item.FileId, rootID, cutoff)
				if err != nil {
					ctx.Error("家庭兜底清理递归扫描目录失败", zap.String("folder_id", item.FileId), zap.String("folder_name", item.FileName), zap.Error(err))
					firstErr = pickFirstErr(firstErr, err)
					continue
				}
				if childDeleted {
					deletedAny = true
				}
				continue
			}
			effectiveTime, sourceLabel, err := s.resolveEffectiveExpireBaseTime(ctx, item.FileId, item.CreateTime)
			if err != nil {
				ctx.Error("家庭兜底清理解析时间失败", zap.String("file_id", item.FileId), zap.String("file_name", item.FileName), zap.String("create_time", item.CreateTime), zap.Error(err))
				firstErr = pickFirstErr(firstErr, err)
				continue
			}
			if effectiveTime.After(cutoff) {
				continue
			}
			ctx.Info("家庭兜底清理命中过期文件", zap.String("file_id", item.FileId), zap.String("file_name", item.FileName), zap.String("time_source", sourceLabel), zap.Time("effective_time", effectiveTime), zap.Time("cutoff", cutoff))
			if err := casrestoreSvi.SafeDeleteFamilyFileByAccessToken(refAccessToken, familyID, item.FileId, item.FileName); err != nil {
				ctx.Error("家庭兜底清理删除过期文件失败", zap.String("file_id", item.FileId), zap.String("file_name", item.FileName), zap.Error(err))
				firstErr = pickFirstErr(firstErr, err)
				continue
			}
			deletedAny = true
		}
	}
	if currentID != rootID {
		empty, err := s.isFamilyFolderEmpty(panClient, fid, currentID)
		if err != nil {
			firstErr = pickFirstErr(firstErr, err)
		} else if empty {
			info, apiErr := panClient.AppGetBasicFileInfo(&cloudpan.AppGetFileInfoParam{FamilyId: fid, FileId: currentID})
			if apiErr != nil {
				firstErr = pickFirstErr(firstErr, apiErr)
			} else if info != nil {
				if err := casrestoreSvi.SafeDeleteFamilyNodeByAccessToken(refAccessToken, familyID, currentID, info.FileName, true); err != nil {
					firstErr = pickFirstErr(firstErr, err)
				} else {
					deletedAny = true
				}
			}
		}
	}
	return deletedAny, firstErr
}

func (s *RecycleRestoredCASScheduler) isPersonFolderEmpty(panClient *cloudpan.PanClient, folderID string) (bool, error) {
	param := cloudpan.NewAppFileListParam()
	param.FileId = folderID
	param.PageSize = 1
	res, apiErr := panClient.AppFileList(param)
	if apiErr != nil {
		return false, apiErr
	}
	return res == nil || res.Count == 0, nil
}

func (s *RecycleRestoredCASScheduler) isFamilyFolderEmpty(panClient *cloudpan.PanClient, familyID int64, folderID string) (bool, error) {
	param := cloudpan.NewAppFileListParam()
	param.FamilyId = familyID
	param.FileId = folderID
	param.PageSize = 1
	res, apiErr := panClient.AppFileList(param)
	if apiErr != nil {
		return false, apiErr
	}
	return res == nil || res.Count == 0, nil
}

func (s *RecycleRestoredCASScheduler) resolveEffectiveExpireBaseTime(ctx appctx.Context, restoredFileID, createTimeRaw string) (time.Time, string, error) {
	if s.casRecordService != nil {
		record, err := s.casRecordService.QueryByRestoredFileID(ctx, restoredFileID)
		if err == nil && record != nil && record.RestoredAt != nil && !record.RestoredAt.IsZero() {
			return *record.RestoredAt, "restored_at", nil
		}
	}
	createdAt, err := parseCloudFileTime(createTimeRaw)
	if err != nil {
		return time.Time{}, "", err
	}
	return createdAt, "create_time", nil
}

func parseCloudFileTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("空创建时间")
	}
	layouts := []string{time.DateTime, "2006-01-02 15:04", time.RFC3339, "2006-01-02T15:04:05"}
	for _, layout := range layouts {
		if ts, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析创建时间: %s", raw)
}

func pickFirstErr(current error, next error) error {
	if current != nil {
		return current
	}
	return next
}

func (s *RecycleRestoredCASScheduler) handleStaleRecycling(ctx appctx.Context, record *models.CasMediaRecord) (bool, error) {
	exists, err := s.restoredFileStillExists(ctx, record)
	if err != nil {
		return false, err
	}
	if !exists {
		s.markRecycled(ctx, record.ID)
		return true, nil
	}
	if strings.Contains(record.LastError, recycleRetryMarker) {
		_ = s.casRecordService.Update(ctx, record.ID, map[string]any{
			"restore_status": models.CasRestoreStatusFailed,
			"last_error":     strings.TrimSpace(record.LastError + " 文件仍存在，已达重试上限"),
		})
		return true, nil
	}
	_ = s.casRecordService.Update(ctx, record.ID, map[string]any{
		"restore_status": models.CasRestoreStatusRestored,
		"last_error":     strings.TrimSpace(record.LastError + " " + recycleRetryMarker),
	})
	return true, nil
}

func (s *RecycleRestoredCASScheduler) markRecycled(ctx appctx.Context, recordID int64) {
	now := time.Now()
	_ = s.casRecordService.Update(ctx, recordID, map[string]any{
		"restore_status":   models.CasRestoreStatusRecycled,
		"restored_file_id": "",
		"recycle_after_at": nil,
		"last_access_at":   &now,
		"last_error":       "",
	})
}

func (s *RecycleRestoredCASScheduler) restoredFileStillExists(ctx appctx.Context, record *models.CasMediaRecord) (bool, error) {
	targetTokenID := record.TargetTokenID
	if targetTokenID <= 0 {
		return true, fmt.Errorf("缺少回收目标token")
	}
	session, err := s.appSessionService.GetByTokenID(ctx, targetTokenID)
	if err != nil {
		return true, err
	}
	panClient, err := s.newPanClientBySession(ctx, session)
	if err != nil {
		return true, err
	}
	if strings.EqualFold(strings.TrimSpace(record.DestinationType), "family") {
		familyID := strings.TrimSpace(record.TargetFamilyID)
		if familyID == "" {
			return true, fmt.Errorf("缺少家庭回收目标familyId")
		}
		var fid int64
		fmt.Sscan(familyID, &fid)
		info, apiErr := panClient.AppFileInfoById(fid, record.RestoredFileID)
		if apiErr != nil {
			msg := strings.ToLower(apiErr.Error())
			if strings.Contains(msg, "not found") || strings.Contains(apiErr.Error(), "文件不存在") {
				return false, nil
			}
			return true, apiErr
		}
		return info != nil, nil
	}
	info, apiErr := panClient.AppFileInfoById(0, record.RestoredFileID)
	if apiErr != nil {
		msg := strings.ToLower(apiErr.Error())
		if strings.Contains(msg, "not found") || strings.Contains(apiErr.Error(), "文件不存在") {
			return false, nil
		}
		return true, apiErr
	}
	return info != nil, nil
}

func (s *RecycleRestoredCASScheduler) recycleOne(ctx appctx.Context, record *models.CasMediaRecord) error {
	targetTokenID := record.TargetTokenID
	if targetTokenID <= 0 {
		return fmt.Errorf("缺少回收目标token")
	}
	session, err := s.appSessionService.GetByTokenID(ctx, targetTokenID)
	if err != nil {
		return err
	}
	fileName := strings.TrimSpace(record.RestoredFileName)
	if fileName == "" {
		fileName = strings.TrimSpace(record.OriginalFileName)
	}
	if fileName == "" {
		fileName = record.RestoredFileID
	}
	if strings.EqualFold(strings.TrimSpace(record.DestinationType), "family") {
		familyID := strings.TrimSpace(record.TargetFamilyID)
		if familyID == "" {
			return fmt.Errorf("缺少家庭回收目标familyId")
		}
		cloudToken, err := s.cloudTokenService.Query(ctx, targetTokenID)
		if err != nil {
			return err
		}
		refAccessToken, err := casrestoreSvi.BuildRefSDKAccessToken(session, cloudToken)
		if err != nil {
			return err
		}
		if err := casrestoreSvi.SafeDeleteFamilyFileByAccessToken(refAccessToken, familyID, record.RestoredFileID, fileName); err != nil {
			return err
		}
		return s.cleanupEmptyAncestorsFamily(ctx, session, refAccessToken, familyID, strings.TrimSpace(record.RestoredParentID), strings.TrimSpace(shared.SettingAddition.CasFamilyTargetFolderId))
	}
	panClient, err := s.newPanClientBySession(ctx, session)
	if err != nil {
		return err
	}
	ok, apiErr := panClient.AppDeleteFile([]string{record.RestoredFileID})
	if apiErr != nil {
		return apiErr
	}
	if !ok {
		return fmt.Errorf("删除个人恢复文件失败")
	}
	if apiErr := panClient.RecycleDelete(0, []string{record.RestoredFileID}); apiErr != nil {
		return apiErr
	}
	if apiErr := panClient.RecycleClear(0); apiErr != nil {
		return apiErr
	}
	return s.cleanupEmptyAncestorsPerson(ctx, panClient, strings.TrimSpace(record.RestoredParentID), strings.TrimSpace(shared.SettingAddition.CasPersonTargetFolderId))
}

func (s *RecycleRestoredCASScheduler) cleanupEmptyAncestorsPerson(ctx appctx.Context, panClient *cloudpan.PanClient, currentID, stopAtID string) error {
	currentID = strings.TrimSpace(currentID)
	stopAtID = strings.TrimSpace(stopAtID)
	for currentID != "" && currentID != stopAtID {
		param := cloudpan.NewAppFileListParam()
		param.FileId = currentID
		param.PageSize = 1
		res, apiErr := panClient.AppFileList(param)
		if apiErr != nil {
			return apiErr
		}
		if res != nil && res.Count > 0 {
			return nil
		}
		info, apiErr := panClient.AppFileInfoById(0, currentID)
		if apiErr != nil {
			return apiErr
		}
		if info == nil {
			return nil
		}
		parentID := strings.TrimSpace(info.ParentId)
		ok, delErr := panClient.AppDeleteFile([]string{currentID})
		if delErr != nil {
			return delErr
		}
		if !ok {
			return fmt.Errorf("删除个人空目录失败: %s", currentID)
		}
		currentID = parentID
	}
	return nil
}

func (s *RecycleRestoredCASScheduler) cleanupEmptyAncestorsFamily(ctx appctx.Context, session *appsession.Session, refAccessToken, familyID, currentID, stopAtID string) error {
	currentID = strings.TrimSpace(currentID)
	stopAtID = strings.TrimSpace(stopAtID)
	panClient, err := s.newPanClientBySession(ctx, session)
	if err != nil {
		return err
	}
	var fid int64
	fmt.Sscan(familyID, &fid)
	for currentID != "" && currentID != stopAtID {
		param := cloudpan.NewAppFileListParam()
		param.FamilyId = fid
		param.FileId = currentID
		param.PageSize = 1
		res, apiErr := panClient.AppFileList(param)
		if apiErr != nil {
			return apiErr
		}
		if res != nil && res.Count > 0 {
			return nil
		}
		info, apiErr := panClient.AppFileInfoById(fid, currentID)
		if apiErr != nil {
			return apiErr
		}
		if info == nil {
			return nil
		}
		parentID := strings.TrimSpace(info.ParentId)
		if err := casrestoreSvi.SafeDeleteFamilyNodeByAccessToken(refAccessToken, familyID, currentID, info.FileName, true); err != nil {
			return err
		}
		currentID = parentID
	}
	return nil
}

