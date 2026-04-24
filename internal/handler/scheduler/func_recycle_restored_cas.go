package scheduler

import (
	"fmt"
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

const recycleRetryMarker = "[retry-once]"

type RecycleRestoredCASScheduler struct {
	casRecordService  casrecordSvi.Service
	appSessionService appsessionSvi.Service
	mountPointService mountpointSvi.Service
	cloudTokenService cloudtokenSvi.Service
	quit              chan struct{}
	done              chan struct{}
	running           bool
}

func NewRecycleRestoredCASScheduler(casRecordService casrecordSvi.Service, appSessionService appsessionSvi.Service, mountPointService mountpointSvi.Service, cloudTokenService cloudtokenSvi.Service) Scheduler {
	return &RecycleRestoredCASScheduler{casRecordService: casRecordService, appSessionService: appSessionService, mountPointService: mountPointService, cloudTokenService: cloudTokenService, quit: make(chan struct{}), done: make(chan struct{})}
}

func (s *RecycleRestoredCASScheduler) Start(ctx appctx.Context) error {
	if s.running {
		return ErrSchedulerRunning
	}
	s.running = true
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
	list, err := s.casRecordService.ListDueRecycle(ctx, time.Now(), 50)
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
			_ = s.casRecordService.Update(ctx, record.ID, map[string]any{"restore_status": models.CasRestoreStatusFailed, "last_error": err.Error()})
			ctx.Error("回收到期CAS恢复文件失败", zap.Int64("record_id", record.ID), zap.Error(err))
			continue
		}
		s.markRecycled(ctx, record.ID)
	}
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
