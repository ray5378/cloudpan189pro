package scheduler

import (
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"go.uber.org/zap"
)

type MemTrimScheduler struct {
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewMemTrimScheduler() Scheduler { return &MemTrimScheduler{} }

func (s *MemTrimScheduler) enabled() bool {
	v := os.Getenv("MEM_TRIM_ENABLE")
	return v == "1" || v == "true" || v == "TRUE" || v == "True"
}

func (s *MemTrimScheduler) interval() time.Duration {
	mins := 10
	if v := os.Getenv("MEM_TRIM_INTERVAL_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			mins = n
		}
	}
	return time.Duration(mins) * time.Minute
}

func (s *MemTrimScheduler) thresholdBytes() uint64 {
	mb := 128
	if v := os.Getenv("MEM_TRIM_THRESHOLD_MB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			mb = n
		}
	}
	return uint64(mb) * 1024 * 1024
}

func (s *MemTrimScheduler) Start(ctx context.Context) error {
	if !s.enabled() { return nil }
	if s.running { return ErrSchedulerRunning }
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	gopool.Go(func(){ for s.doJob(){} })
	return nil
}

func (s *MemTrimScheduler) Stop() {
	if !s.running { return }
	s.cancel()
	s.running = false
}

func (s *MemTrimScheduler) doJob() bool {
	ctx := s.ctx
	logger := ctx.Logger
	interval := s.interval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			// 读取内存统计
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			reclaimable := m.HeapIdle - m.HeapReleased // 可归还的空闲页
			th := s.thresholdBytes()
			if reclaimable < th { continue }
			beforeRSS := getRSS()
			logger.Info("memory trim: start", zap.Uint64("reclaimable_bytes", reclaimable), zap.Uint64("threshold_bytes", th), zap.Uint64("rss_before", beforeRSS))
			debug.FreeOSMemory()
			runtime.GC()
			var m2 runtime.MemStats
			runtime.ReadMemStats(&m2)
			afterRSS := getRSS()
			logger.Info("memory trim: done", zap.Uint64("rss_after", afterRSS), zap.Uint64("heap_idle", m2.HeapIdle), zap.Uint64("heap_released", m2.HeapReleased))
		}
	}
}

// getRSS 返回近似 RSS（仅用于日志观测；失败则返回 0）
func getRSS() uint64 {
	// 在不同平台取 RSS 实现差异较大；这里保守返回 0，避免引入额外依赖。
	// 留作扩展：可以读取 /proc/self/statm 或者使用 golang.org/x/sys 读取进程内存。
	return 0
}
