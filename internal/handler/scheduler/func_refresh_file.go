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

	defer func() {
		if r := recover(); r != nil {
			ctx.Error("文件刷新执行器发生异常",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()

	select {
	case <-ctx.Done():
		ctx.Info("文件刷新执行器停止")

		return false
	case <-time.After(time.Minute):
		mountPoints, err := s.mountPointService.GetAutoRefreshList(ctx, &mountpoint.GetAutoRefreshListRequest{})
		if err != nil {
			ctx.Error("查询挂载点失败", zap.Error(err))

			return true
		}

		ctx.Debug("文件刷新执行器查询到挂载点数量", zap.Int("count", len(mountPoints)))

		now := time.Now()

		for _, mp := range mountPoints {
			// 只处理启用自动刷新的挂载点
			if !mp.EnableAutoRefresh {
				continue
			}

			if mp.RefreshInterval <= 0 || mp.AutoRefreshBeginAt == nil {
				continue
			}

			// 以每个挂载点自己的 auto_refresh_begin_at 作为刷新节奏锚点，
			// 并用“槽位(slot)”判断是否该刷新：
			//   slot = floor((now - beginAt) / interval)
			// 仅当当前时间位于该槽位开始后的 60 秒窗口内时才触发，
			// 这样能容忍 scheduler 每分钟轮询的轻微漂移，不再依赖“恰好整除”。
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

			// 同一进程内对每个 mount point 的同一 slot 只触发一次
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

			// 创建文件扫描任务
			taskReq := &topic.FileScanFileRequest{
				FileId: mp.FileId,
				Deep:   mp.EnableDeepRefresh,
			}

			body, _ := json.Marshal(taskReq)
			taskCtx := ctx.
				WithValue(consts.CtxKeyFullPath, mp.FullPath).
				WithValue(consts.CtxKeyInvokeHandlerName, "定时任务")

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

	return true
}
