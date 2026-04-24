package scheduler

import (
	"time"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	localstrmSvi "github.com/xxcheng123/cloudpan189-share/internal/services/localstrm"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
)

type LocalCASSTRMScanScheduler struct {
	localSTRMService localstrmSvi.Service
	quit             chan struct{}
	done             chan struct{}
	running          bool
	nextRunAt        time.Time
}

func NewLocalCASSTRMScanScheduler(localSTRMService localstrmSvi.Service) Scheduler {
	return &LocalCASSTRMScanScheduler{localSTRMService: localSTRMService, quit: make(chan struct{}), done: make(chan struct{})}
}

func (s *LocalCASSTRMScanScheduler) Start(ctx appctx.Context) error {
	if s.running {
		return ErrSchedulerRunning
	}
	s.running = true
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		s.doJob(ctx, true)
		for {
			select {
			case <-ticker.C:
				s.doJob(ctx, false)
			case <-s.quit:
				return
			}
		}
	}()
	return nil
}

func (s *LocalCASSTRMScanScheduler) Stop() {
	if !s.running {
		return
	}
	close(s.quit)
	<-s.done
	s.running = false
}

func (s *LocalCASSTRMScanScheduler) doJob(ctx appctx.Context, immediate bool) {
	if s.localSTRMService == nil {
		return
	}
	cfg := shared.SettingAddition
	if !cfg.LocalCASAutoScanEnabled {
		s.nextRunAt = time.Time{}
		return
	}
	interval := cfg.LocalCASAutoScanIntervalMin
	if interval <= 0 {
		interval = 10
	}
	now := time.Now()
	if !immediate && !s.nextRunAt.IsZero() && now.Before(s.nextRunAt) {
		return
	}
	_, _ = s.localSTRMService.ScanAndEnsureAll(ctx)
	s.nextRunAt = now.Add(time.Duration(interval) * time.Minute)
}
