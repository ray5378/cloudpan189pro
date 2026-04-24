package scheduler

import (
	stdcontext "context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type VacuumScheduler struct {
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	db      *gorm.DB
}

func NewVacuumScheduler(svc bootstrap.ServiceContext) Scheduler {
	// 用一个无业务负载的 Context 包装（仅用到 DB 对象，不传播生命周期）
	ctx := context.NewContext(stdcontext.Background())
	return &VacuumScheduler{db: svc.GetDB(ctx)}
}

func (s *VacuumScheduler) Start(ctx context.Context) error {
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

func (s *VacuumScheduler) Stop() {
	if !s.running {
		return
	}
	s.cancel()
	s.running = false
}

func (s *VacuumScheduler) retentionWeeks() int {
	v := os.Getenv("SQLITE_VACUUM_WEEKS")
	if v == "" {
		return 1
	}
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return n
	}
	return 1
}

func (s *VacuumScheduler) isSQLite() bool {
	db, err := s.db.DB()
	if err != nil {
		return false
	}
	// 通过驱动名粗略判断（glebarez/sqlite 使用 sqlite）
	return driverName(db) == "sqlite"
}

func driverName(db *sql.DB) string {
	// 标准库没有直接暴露驱动名，这里用类型字符串做一个保守判断
	// 由于项目默认使用 glebarez/sqlite，当 DBType 切到 mysql 时，这里会随之变化
	return fmt.Sprintf("%T", db.Driver())
}

func (s *VacuumScheduler) doJob() bool {
	ctx := s.ctx
	logger := ctx.Logger

	// 每周执行一次（默认每周），启动后先等待一个周期避免刚启动就重写数据库
	interval := time.Duration(s.retentionWeeks()) * 7 * 24 * time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if !s.isSQLite() {
				continue
			}
			if err := s.db.Exec("VACUUM;").Error; err != nil {
				logger.Error("SQLite VACUUM 执行失败", zap.Error(err))
			} else {
				logger.Info("SQLite VACUUM 执行完成")
			}
		}
	}
}
