package scheduler

import (
	"os"
	"strconv"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	filetasklogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/filetasklog"
	"go.uber.org/zap"
)

type CleanupTaskLogScheduler struct {
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	svc     filetasklogSvi.Service
}

func NewCleanupTaskLogScheduler(svc filetasklogSvi.Service) Scheduler {
	return &CleanupTaskLogScheduler{svc: svc}
}

func (s *CleanupTaskLogScheduler) Start(ctx context.Context) error {
	if s.running { return ErrSchedulerRunning }
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	gopool.Go(func(){ for s.doJob(){} })
	return nil
}

func (s *CleanupTaskLogScheduler) Stop() {
	if !s.running { return }
	s.cancel()
	s.running = false
}

func (s *CleanupTaskLogScheduler) retentionDays() int {
	v := os.Getenv("TASKLOG_RETENTION_DAYS")
	if v == "" { return 15 }
	if n, err := strconv.Atoi(v); err == nil && n > 0 { return n }
	return 15
}

func (s *CleanupTaskLogScheduler) doJob() bool {
	ctx := s.ctx
	logger := ctx.Logger

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			// 每日清理一次
			ret := s.retentionDays()
			before := time.Now().Add(-time.Duration(ret) * 24 * time.Hour)
			deleted, err := s.svc.CleanupOlderThan(ctx, before)
			if err != nil {
				logger.Error("清理任务日志失败", zap.Error(err))
			} else {
				logger.Info("任务日志清理完成", zap.Int("retention_days", ret), zap.Int64("deleted", deleted))
			}
		}
	}
}
