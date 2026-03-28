package scheduler

import (
	"encoding/json"
	"runtime/debug"
	"sync"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/taskengine"
	"github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
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
	lastTriggeredSlot map[int64]int64
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

			for _, mp := range mountPoints {
				if !mp.EnableAutoRefresh {
					continue
				}
				if mp.RefreshInterval <= 0 || mp.AutoRefreshBeginAt == nil {
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

				ctx.Info("文件扫描执行器触发",
					zap.Int64("mount_point_id", mp.ID),
					zap.Int64("file_id", mp.FileId),
					zap.String("full_path", mp.FullPath),
					zap.Int("refresh_interval", mp.RefreshInterval),
					zap.Int64("refresh_slot", slot),
					zap.Time("slot_start", slotStart))

				taskReq := &topic.FileScanFileRequest{FileId: mp.FileId, Deep: mp.EnableDeepRefresh}
				body, _ := json.Marshal(taskReq)
				taskCtx := ctx.WithValue(consts.CtxKeyFullPath, mp.FullPath).WithValue(consts.CtxKeyInvokeHandlerName, "定时任务")

				if err = s.taskEngine.PushMessage(taskCtx, taskReq.Topic(), body); err != nil {
					ctx.Error("推送文件扫描任务失败",
						zap.Int64("mount_point_id", mp.ID),
						zap.Int64("file_id", mp.FileId),
						zap.String("full_path", mp.FullPath),
						zap.Error(err))
				} else {
					ctx.Info("下发文件扫描任务成功",
						zap.Int64("mount_point_id", mp.ID),
						zap.Int64("file_id", mp.FileId),
						zap.String("full_path", mp.FullPath))
				}
			}
		}
	}
}
