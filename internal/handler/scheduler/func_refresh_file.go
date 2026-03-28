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
}

func NewRefreshFileScheduler(mountPointService mountpoint.Service, taskEngine taskengine.TaskEngine) Scheduler {
	return &RefreshFileScheduler{
		mountPointService: mountPointService,
		taskEngine:        taskEngine,
		running:           false,
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
			// 避免所有同 interval 节点在全局整点被一起误刷。
			beginAt := mp.AutoRefreshBeginAt.In(now.Location())
			if now.Before(beginAt) {
				continue
			}

			elapsedMin := int(now.Sub(beginAt).Minutes())
			if elapsedMin < 0 || elapsedMin%mp.RefreshInterval != 0 {
				continue
			}

			ctx.Info("文件扫描执行器触发",
				zap.Int64("mount_point_id", mp.ID),
				zap.Int64("file_id", mp.FileId),
				zap.String("full_path", mp.FullPath),
				zap.Int("refresh_interval", mp.RefreshInterval))

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
