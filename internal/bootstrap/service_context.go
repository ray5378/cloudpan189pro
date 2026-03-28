package bootstrap

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/taskengine"

	"github.com/casbin/casbin/v2"
	"github.com/xxcheng123/cloudpan189-share/internal/configs"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
)

type ServiceContext interface {
	GetDB(ctx context.Context) *gorm.DB
	GetLogger(name string, fields ...zap.Field) *zap.Logger
	Close()
	GetPort() int
	GetTaskEngine() taskengine.TaskEngine
	GetHTTPEngine() *gin.Engine
}

type serviceContext struct {
	config     *configs.RuntimeConfig
	db         *gorm.DB
	logger     *zap.Logger
	taskEngine taskengine.TaskEngine
	httpEngine *gin.Engine
}

func (s *serviceContext) GetDB(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx)
}

func (s *serviceContext) GetLogger(name string, fields ...zap.Field) *zap.Logger {
	return s.logger.Named(name).With(fields...)
}

func (s *serviceContext) Close() {
	_ = s.logger.Sync()
}

func (s *serviceContext) GetPort() int {
	return s.config.Port
}

func (s *serviceContext) GetTaskEngine() taskengine.TaskEngine {
	return s.taskEngine
}

func (s *serviceContext) GetHTTPEngine() *gin.Engine {
	return s.httpEngine
}

func New(c *configs.RuntimeConfig) (ServiceContext, error) {
	return newServiceContext(c)
}

func newServiceContext(c *configs.RuntimeConfig) (ServiceContext, error) {
	var (
		db     *gorm.DB
		err    error
		logger *zap.Logger
	)

	// 连接 db
	if db, err = connectDB(c.Config); err != nil {
		return nil, err
	}

	// 执行数据迁移
	if err = migrateDB(db); err != nil {
		return nil, err
	}

	// 初始化日志
	if logger, err = initLogger(c); err != nil {
		return nil, err
	}

	// 初始化 setting
	if err = initSetting(db); err != nil {
		return nil, err
	}

	// 初始化全局共享变量
	if err = assignShared(db); err != nil {
		return nil, err
	}

	taskEngine := initTaskEngine(logger)

	gLogger := zapgorm2.New(logger)
	gLogger.SetAsDefault()
	gLogger.IgnoreRecordNotFoundError = true
	gLogger.SlowThreshold = time.Second * 3

	db = db.Session(&gorm.Session{
		Logger: gLogger,
	})

	httpEngine := gin.New()
	httpEngine.Use(gin.Recovery())

	return &serviceContext{
		config:     c,
		db:         db,
		logger:     logger,
		taskEngine: taskEngine,
		httpEngine: httpEngine,
	}, nil
}

type mockServiceContext struct {
}

func (m *mockServiceContext) GetDB(ctx context.Context) *gorm.DB {
	// 返回一个假的 gorm.DB，实际测试中可以使用 sqlite 内存数据库
	return nil
}

func (m *mockServiceContext) GetLogger(name string, fields ...zap.Field) *zap.Logger {
	// 返回一个 nop logger，不会输出任何日志
	return zap.NewNop()
}

func (m *mockServiceContext) GetFileEnforcer() *casbin.Enforcer {
	// 返回 nil，测试中不进行权限检查
	return nil
}

func (m *mockServiceContext) Close() {
	// mock 实现中不需要做任何清理工作
}

func (m *mockServiceContext) GetPort() int {
	// 返回假的端口号
	return 9999
}

func (m *mockServiceContext) GetTaskEngine() taskengine.TaskEngine {
	return nil
}

func (m *mockServiceContext) GetHTTPEngine() *gin.Engine {
	return nil
}

// NewMockServiceContext 创建一个用于测试的 mock ServiceContext
func NewMockServiceContext() ServiceContext {
	return &mockServiceContext{}
}
