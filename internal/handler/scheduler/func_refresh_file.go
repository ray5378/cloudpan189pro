package scheduler

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/taskengine"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	filetasklogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/filetasklog"
	"github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	"github.com/xxcheng123/cloudpan189-share/internal/services/virtualfile"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"github.com/xxcheng123/cloudpan189-share/internal/types/topic"
	"go.uber.org/zap"
)

type autoDeleteConfirmationState struct {
	RoundsTriggered   int
	RoundsConfirmed   int
	LastObservedLogID int64
	NextRunAt         time.Time
}

type RefreshFileScheduler struct {
	running            bool
	mu                 sync.Mutex
	ctx                context.Context
	cancel             context.CancelFunc
	mountPointService  mountpoint.Service
	fileTaskLogService filetasklogSvi.Service
	virtualFileService virtualfile.Service
	taskEngine         taskengine.TaskEngine

	// 进程内去重：记录每个挂载点最近已触发的 refresh slot，
	// 避免 scheduler 抖动或 doJob 耗时导致同一槽位重复触发。
	lastTriggeredSlot       map[int64]int64
	lastPersistentCheckKey  string
	lastAutoDeleteKey       string
	autoDeleteConfirmations map[int64]*autoDeleteConfirmationState
}

func NewRefreshFileScheduler(mountPointService mountpoint.Service, fileTaskLogService filetasklogSvi.Service, virtualFileService virtualfile.Service, taskEngine taskengine.TaskEngine) Scheduler {
	return &RefreshFileScheduler{
		mountPointService:       mountPointService,
		fileTaskLogService:      fileTaskLogService,
		virtualFileService:      virtualFileService,
		taskEngine:              taskEngine,
		running:                 false,
		lastTriggeredSlot:       make(map[int64]int64),
		autoDeleteConfirmations: make(map[int64]*autoDeleteConfirmationState),
	}
}

func (s *RefreshFileScheduler) Start(ctx context.Context) error {
	if !s.mu.TryLock() {
		return ErrSchedulerRunning
	}
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	s.running = true

	gopool.Go(func() {
		for s.doJob() {
		}

		ctx.Info("文件刷新执行器已停止~")
	})

	return nil
}

func (s *RefreshFileScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cancel()
	s.running = false
}

func (s *RefreshFileScheduler) doJob() bool {
	ctx := s.ctx
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	defer func() {
		if r := recover(); r != nil {
			ctx.Error("文件刷新执行器发生异常",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			ctx.Info("文件刷新执行器停止")
			return false
		case <-ticker.C:
			mountPoints, err := s.mountPointService.GetAutoRefreshList(ctx, &mountpoint.GetAutoRefreshListRequest{})
			if err != nil {
				ctx.Error("查询挂载点失败", zap.Error(err))
				continue
			}

			ctx.Debug("文件刷新执行器查询到挂载点数量", zap.Int("count", len(mountPoints)))
			now := time.Now()
			s.runPersistentCheck(ctx, now)
			s.runAutoDeletePermanentInvalid(ctx, now)

			for _, mp := range mountPoints {
				if !mp.EnableAutoRefresh || mp.RefreshInterval <= 0 || mp.AutoRefreshBeginAt == nil {
					continue
				}
				beginAt := mp.AutoRefreshBeginAt.In(now.Location())
				if now.Before(beginAt) {
					continue
				}

				interval := time.Duration(mp.RefreshInterval) * time.Minute
				elapsed := now.Sub(beginAt)
				if elapsed < 0 {
					continue
				}

				slot := int64(elapsed / interval)
				slotStart := beginAt.Add(time.Duration(slot) * interval)
				if now.Sub(slotStart) >= time.Minute {
					continue
				}

				s.mu.Lock()
				lastSlot, exists := s.lastTriggeredSlot[mp.ID]
				if exists && lastSlot == slot {
					s.mu.Unlock()
					continue
				}
				s.lastTriggeredSlot[mp.ID] = slot
				s.mu.Unlock()

				s.enqueueNormalRefresh(ctx, mp, "定时任务", zap.Int64("refresh_slot", slot), zap.Time("slot_start", slotStart), zap.Int("refresh_interval", mp.RefreshInterval))
			}
		}
	}
}

func (s *RefreshFileScheduler) runPersistentCheck(ctx context.Context, now time.Time) {
	cfg := shared.SettingAddition
	if !cfg.PersistentCheckEnabled {
		return
	}

	hour, minute, ok := parseClockHM(cfg.PersistentCheckTime)
	if !ok {
		ctx.Error("持久检测存储时间配置无效", zap.String("persistent_check_time", cfg.PersistentCheckTime))
		return
	}
	if now.Day() != cfg.PersistentCheckDay || now.Hour() != hour || now.Minute() != minute {
		return
	}

	checkKey := fmt.Sprintf("%04d-%02d-%02d %02d:%02d", now.Year(), now.Month(), now.Day(), hour, minute)
	s.mu.Lock()
	if s.lastPersistentCheckKey == checkKey {
		s.mu.Unlock()
		return
	}
	s.lastPersistentCheckKey = checkKey
	s.mu.Unlock()

	allMounts, err := s.mountPointService.List(ctx, &mountpoint.ListRequest{NoPaginate: true})
	if err != nil {
		ctx.Error("查询持久检测存储列表失败", zap.Error(err))
		return
	}

	count := 0
	for _, mp := range allMounts {
		if mp == nil {
			continue
		}
		if mp.EnableAutoRefresh && mp.IsInAutoRefreshPeriod() {
			continue
		}
		s.enqueueNormalRefresh(ctx, mp, "持久检测存储", zap.Bool("persistent_check", true))
		count++
	}

	ctx.Info("持久检测存储执行完成", zap.Int("count", count), zap.String("schedule_key", checkKey))
}

func (s *RefreshFileScheduler) runAutoDeletePermanentInvalid(ctx context.Context, now time.Time) {
	cfg := shared.SettingAddition
	if !cfg.AutoDeleteInvalidStorageEnabled {
		return
	}
	// 每天只在 12:00 执行一次，不做全天候巡检。
	if now.Hour() != 12 || now.Minute() != 0 {
		return
	}

	checkKey := now.Format("2006-01-02 12:00")
	s.mu.Lock()
	alreadyHandled := s.lastAutoDeleteKey == checkKey
	if !alreadyHandled {
		s.lastAutoDeleteKey = checkKey
	}
	s.mu.Unlock()
	if alreadyHandled {
		return
	}

	keywords := splitKeywords(cfg.AutoDeleteInvalidStorageKeywords)
	mounts, err := s.mountPointService.List(ctx, &mountpoint.ListRequest{NoPaginate: true})
	if err != nil {
		ctx.Error("查询存储节点失败", zap.Error(err))
		return
	}
	if len(mounts) == 0 {
		return
	}

	lastLogs, err := s.collectLatestLogs(ctx, mounts)
	if err != nil {
		ctx.Error("查询失效存储最新任务日志失败", zap.Error(err))
		return
	}

	deleteIDs := make([]int64, 0)
	deleteReasons := make(map[int64]string)
	rule2Candidates := make([]*models.MountPoint, 0)
	for _, mp := range mounts {
		if mp == nil {
			continue
		}
		lastLog := lastLogs[mp.FileId]
		matchedKeyword, matchedKeywordText := logMatchesAnyKeyword(lastLog, keywords)
		if matchedKeyword {
			deleteIDs = append(deleteIDs, mp.FileId)
			deleteReasons[mp.FileId] = fmt.Sprintf("命中自动删除关键词: %s", matchedKeywordText)
			continue
		}

		if !mp.EnableAutoRefresh || !mp.IsInAutoRefreshPeriod() {
			rule2Candidates = append(rule2Candidates, mp)
		}
	}

	fileCountMap, ok := s.collectRule2FileCounts(ctx, rule2Candidates)
	if !ok {
		return
	}
	for _, mp := range rule2Candidates {
		if mp == nil {
			continue
		}
		lastLog := lastLogs[mp.FileId]
		latestRefreshSucceeded := lastLog != nil && lastLog.Status == models.StatusCompleted
		if fileCountMap[mp.FileId] == 0 && latestRefreshSucceeded {
			deleteIDs = append(deleteIDs, mp.FileId)
			deleteReasons[mp.FileId] = "未启用自动刷新或已过期，且文件数量为0、最新刷新成功"
		}
	}

	s.deleteMountsWithReasons(ctx, deleteIDs, deleteReasons, checkKey)
}

func (s *RefreshFileScheduler) collectLatestLogs(ctx context.Context, mounts []*models.MountPoint) (map[int64]*models.FileTaskLog, error) {
	fileIDs := make([]int64, 0, len(mounts))
	for _, mp := range mounts {
		if mp == nil {
			continue
		}
		fileIDs = append(fileIDs, mp.FileId)
	}
	if len(fileIDs) == 0 {
		return make(map[int64]*models.FileTaskLog), nil
	}
	return s.fileTaskLogService.LatestByFileIDs(ctx, fileIDs)
}

func (s *RefreshFileScheduler) collectRule2FileCounts(ctx context.Context, mounts []*models.MountPoint) (map[int64]int64, bool) {
	fileIDs := make([]int64, 0, len(mounts))
	for _, mp := range mounts {
		if mp == nil {
			continue
		}
		fileIDs = append(fileIDs, mp.FileId)
	}

	fileCountMap := make(map[int64]int64, len(fileIDs))
	if len(fileIDs) == 0 {
		return fileCountMap, true
	}

	const batchSize = 200
	for start := 0; start < len(fileIDs); start += batchSize {
		end := start + batchSize
		if end > len(fileIDs) {
			end = len(fileIDs)
		}
		batch := fileIDs[start:end]

		counts, countErr := s.virtualFileService.GroupCountByTopId(ctx, &virtualfile.GroupCountByTopIdRequest{TopIdList: batch})
		if countErr != nil {
			ctx.Error("查询失效存储文件数量失败", zap.Error(countErr))
			return nil, false
		}
		for _, item := range counts {
			if item == nil {
				continue
			}
			fileCountMap[item.TopId] = item.Count
		}
	}

	return fileCountMap, true
}

func (s *RefreshFileScheduler) processAutoDeleteConfirmations(ctx context.Context, now time.Time, mountByFileID map[int64]*models.MountPoint, lastLogs map[int64]*models.FileTaskLog, fileCountMap map[int64]int64) {
	s.mu.Lock()
	pending := make(map[int64]*autoDeleteConfirmationState, len(s.autoDeleteConfirmations))
	for fileID, state := range s.autoDeleteConfirmations {
		pending[fileID] = &autoDeleteConfirmationState{
			RoundsTriggered:   state.RoundsTriggered,
			RoundsConfirmed:   state.RoundsConfirmed,
			LastObservedLogID: state.LastObservedLogID,
			NextRunAt:         state.NextRunAt,
		}
	}
	s.mu.Unlock()

	deleteIDs := make([]int64, 0)
	deleteReasons := make(map[int64]string)

	for fileID, state := range pending {
		mp := mountByFileID[fileID]
		if mp == nil {
			s.mu.Lock()
			delete(s.autoDeleteConfirmations, fileID)
			s.mu.Unlock()
			continue
		}

		lastLog := lastLogs[fileID]
		if state.RoundsTriggered == 0 {
			if now.Before(state.NextRunAt) {
				continue
			}
			s.enqueueRefresh(ctx, mp, "自动删除失效存储-深度确认", true, zap.Int("confirm_round", 1), zap.Int("confirm_round_total", 6))
			state.RoundsTriggered = 1
			state.NextRunAt = now.Add(10 * time.Minute)
			if lastLog != nil {
				state.LastObservedLogID = lastLog.ID
			}
			s.mu.Lock()
			s.autoDeleteConfirmations[fileID] = state
			s.mu.Unlock()
			continue
		}

		if now.Before(state.NextRunAt) {
			continue
		}

		if lastLog == nil || lastLog.ID <= state.LastObservedLogID {
			continue
		}

		expiredOrDisabled := !mp.EnableAutoRefresh || !mp.IsInAutoRefreshPeriod()
		zeroFiles := fileCountMap[fileID] == 0
		if lastLog.Status != models.StatusCompleted || !expiredOrDisabled || !zeroFiles {
			ctx.Info("自动删除失效存储取消深度确认流程", zap.Int64("file_id", fileID), zap.String("full_path", mp.FullPath))
			s.mu.Lock()
			delete(s.autoDeleteConfirmations, fileID)
			s.mu.Unlock()
			continue
		}

		state.RoundsConfirmed++
		state.LastObservedLogID = lastLog.ID
		if state.RoundsConfirmed >= 6 {
			deleteIDs = append(deleteIDs, fileID)
			deleteReasons[fileID] = "未启用自动刷新或已过期，且连续6轮深度刷新（每10分钟一轮）均成功且文件数量仍为0"
			s.mu.Lock()
			delete(s.autoDeleteConfirmations, fileID)
			s.mu.Unlock()
			continue
		}

		nextRound := state.RoundsConfirmed + 1
		s.enqueueRefresh(ctx, mp, "自动删除失效存储-深度确认", true, zap.Int("confirm_round", nextRound), zap.Int("confirm_round_total", 6))
		state.RoundsTriggered = nextRound
		state.NextRunAt = now.Add(10 * time.Minute)
		s.mu.Lock()
		s.autoDeleteConfirmations[fileID] = state
		s.mu.Unlock()
	}

	s.deleteMountsWithReasons(ctx, deleteIDs, deleteReasons, now.Format("2006-01-02 15:04"))
}

func (s *RefreshFileScheduler) deleteMountsWithReasons(ctx context.Context, deleteIDs []int64, deleteReasons map[int64]string, scheduleKey string) {
	if len(deleteIDs) == 0 {
		return
	}
	for _, fileID := range deleteIDs {
		reason := deleteReasons[fileID]
		tracker, createErr := s.fileTaskLogService.Create(
			ctx,
			"自动删除失效存储",
			"自动删除失效存储",
			filetasklogSvi.WithFile(fileID),
			filetasklogSvi.WithDesc(reason),
		)
		if createErr == nil && tracker != nil {
			_ = s.fileTaskLogService.Completed(ctx, tracker)
		}
	}

	taskReq := &topic.FileBatchDeleteRequest{IDs: deleteIDs}
	body, _ := json.Marshal(taskReq)
	taskCtx := ctx.WithValue(consts.CtxKeyInvokeHandlerName, "自动删除失效存储")
	if err := s.taskEngine.PushMessage(taskCtx, taskReq.Topic(), body); err != nil {
		ctx.Error("推送自动删除失效存储任务失败", zap.Error(err), zap.Int("count", len(deleteIDs)))
		return
	}
	ctx.Info("自动删除失效存储执行完成", zap.Int("count", len(deleteIDs)), zap.String("schedule_key", scheduleKey))
}

func (s *RefreshFileScheduler) enqueueRefresh(ctx context.Context, mp *models.MountPoint, invokeName string, deep bool, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.Int64("mount_point_id", mp.ID),
		zap.Int64("file_id", mp.FileId),
		zap.String("full_path", mp.FullPath),
		zap.String("invoke_name", invokeName),
		zap.Bool("deep", deep),
	}
	baseFields = append(baseFields, fields...)
	ctx.Info("文件扫描执行器触发", baseFields...)

	taskReq := &topic.FileScanFileRequest{FileId: mp.FileId, Deep: deep}
	body, _ := json.Marshal(taskReq)
	taskCtx := ctx.WithValue(consts.CtxKeyFullPath, mp.FullPath).WithValue(consts.CtxKeyInvokeHandlerName, invokeName)
	if err := s.taskEngine.PushMessage(taskCtx, taskReq.Topic(), body); err != nil {
		ctx.Error("推送文件扫描任务失败", append(baseFields, zap.Error(err))...)
		return
	}
	ctx.Info("下发文件扫描任务成功", baseFields...)
}

func (s *RefreshFileScheduler) enqueueNormalRefresh(ctx context.Context, mp *models.MountPoint, invokeName string, fields ...zap.Field) {
	s.enqueueRefresh(ctx, mp, invokeName, false, fields...)
}

func splitKeywords(raw string) []string {
	parts := strings.Split(raw, "|")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func logMatchesAnyKeyword(logItem *models.FileTaskLog, keywords []string) (bool, string) {
	if logItem == nil || len(keywords) == 0 {
		return false, ""
	}
	text := strings.ToLower(logItem.ErrorMsg + "\n" + logItem.Result + "\n" + logItem.Desc + "\n" + logItem.Title)
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true, keyword
		}
	}
	return false, ""
}

func parseClockHM(value string) (hour, minute int, ok bool) {
	if _, err := fmt.Sscanf(value, "%02d:%02d", &hour, &minute); err != nil {
		return 0, 0, false
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, false
	}
	return hour, minute, true
}
