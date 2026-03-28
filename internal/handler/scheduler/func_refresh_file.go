package scheduler

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/taskengine"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"github.com/xxcheng123/cloudpan189-share/internal/types/topic"
	"go.uber.org/zap"
)

type RefreshFileScheduler struct {
	running           bool
	mu                sync.Mutex
	ctx               context.Context
	cancel            context.CancelFunc
	mountPointService mountpoint.Service
	taskEngine        taskengine.TaskEngine

	// 进程内去重：记录每个挂载点最近已触发的 refresh slot，
	// 避免 scheduler 抖动或 doJob 耗时导致同一槽位重复触发。
	lastTriggeredSlot      map[int64]int64
	lastPersistentCheckKey string
	lastAutoDeleteKey      string
}

func NewRefreshFileScheduler(mountPointService mountpoint.Service, taskEngine taskengine.TaskEngine) Scheduler {
	return &RefreshFileScheduler{
		mountPointService: mountPointService,
		taskEngine:        taskEngine,
		running:           false,
		lastTriggeredSlot: make(map[int64]int64),
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
	if now.Hour() != 12 || now.Minute() != 0 {
		return
	}

	checkKey := now.Format("2006-01-02 12:00")
	s.mu.Lock()
	if s.lastAutoDeleteKey == checkKey {
		s.mu.Unlock()
		return
	}
	s.lastAutoDeleteKey = checkKey
	s.mu.Unlock()

	list, err := s.mountPointService.List(ctx, &mountpoint.ListRequest{
		NoPaginate:    true,
		TaskLogStatus: models.StatusFailed,
		FailureKind:   "permanent",
	})
	if err != nil {
		ctx.Error("查询永久失效存储节点失败", zap.Error(err))
		return
	}
	if len(list) == 0 {
		ctx.Info("自动删除永久失效存储执行完成", zap.Int("count", 0), zap.String("schedule_key", checkKey))
		return
	}

	ids := make([]int64, 0, len(list))
	for _, mp := range list {
		if mp == nil {
			continue
		}
		ids = append(ids, mp.FileId)
	}
	if len(ids) == 0 {
		return
	}

	taskReq := &topic.FileBatchDeleteRequest{IDs: ids}
	body, _ := json.Marshal(taskReq)
	taskCtx := ctx.WithValue(consts.CtxKeyInvokeHandlerName, "自动删除永久失效存储")
	if err := s.taskEngine.PushMessage(taskCtx, taskReq.Topic(), body); err != nil {
		ctx.Error("推送自动删除永久失效存储任务失败", zap.Error(err), zap.Int("count", len(ids)))
		return
	}
	ctx.Info("自动删除永久失效存储执行完成", zap.Int("count", len(ids)), zap.String("schedule_key", checkKey))
}

func (s *RefreshFileScheduler) enqueueNormalRefresh(ctx context.Context, mp *models.MountPoint, invokeName string, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.Int64("mount_point_id", mp.ID),
		zap.Int64("file_id", mp.FileId),
		zap.String("full_path", mp.FullPath),
		zap.String("invoke_name", invokeName),
	}
	baseFields = append(baseFields, fields...)
	ctx.Info("文件扫描执行器触发", baseFields...)

	taskReq := &topic.FileScanFileRequest{FileId: mp.FileId, Deep: false}
	body, _ := json.Marshal(taskReq)
	taskCtx := ctx.WithValue(consts.CtxKeyFullPath, mp.FullPath).WithValue(consts.CtxKeyInvokeHandlerName, invokeName)
	if err := s.taskEngine.PushMessage(taskCtx, taskReq.Topic(), body); err != nil {
		ctx.Error("推送文件扫描任务失败", append(baseFields, zap.Error(err))...)
		return
	}
	ctx.Info("下发文件扫描任务成功", baseFields...)
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
