package scheduler

import (
	"os"
	"strconv"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	loginlogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/loginlog"
	"go.uber.org/zap"
)

type CleanupLoginLogScheduler struct {
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	svc     loginlogSvi.Service
}

func NewCleanupLoginLogScheduler(svc loginlogSvi.Service) Scheduler {
	return &CleanupLoginLogScheduler{svc: svc}
}

func (s *CleanupLoginLogScheduler) Start(ctx context.Context) error {
	if s.running {
		return ErrSchedulerRunning
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	gopool.Go(func() {
		for s.doJob() {
		}
	})
	return nil
}

func (s *CleanupLoginLogScheduler) Stop() {
	if !s.running {
		return
	}
	s.cancel()
	s.running = false
}

func (s *CleanupLoginLogScheduler) retentionDays() int {
	v := os.Getenv("LOGINLOG_RETENTION_DAYS")
	if v == "" {
		return 15
	}
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return n
	}
	return 15
}

func (s *CleanupLoginLogScheduler) doJob() bool {
	ctx := s.ctx
	logger := ctx.Logger

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			ret := s.retentionDays()
			before := time.Now().Add(-time.Duration(ret) * 24 * time.Hour)
			deleted, err := s.svc.CleanupOlderThan(ctx, before)
			if err != nil {
				logger.Error("清理登录日志失败", zap.Error(err))
			} else {
				logger.Info("登录日志清理完成", zap.Int("retention_days", ret), zap.Int64("deleted", deleted))
			}
		}
	}
}
