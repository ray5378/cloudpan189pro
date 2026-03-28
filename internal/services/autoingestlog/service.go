package autoingestlog

import (
	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/types/autoingest"
	"gorm.io/gorm"
)

// Service 面向 AutoIngestLog 的日志服务接口（与计划服务分离）
type Service interface {
	// Create 写日志
	Create(ctx context.Context, planID int64, level autoingest.LogLevel, content string) (int64, error)

	// List 查日志
	List(ctx context.Context, req *ListRequest) ([]*models.AutoIngestLog, error)
	Count(ctx context.Context, req *ListRequest) (int64, error)
}

// service 与构造函数，保持与 autoingestplan/mountpoint/filetasklog 风格一致
type service struct {
	svc bootstrap.ServiceContext
}

func NewService(svc bootstrap.ServiceContext) Service {
	return &service{
		svc: svc,
	}
}

func (s *service) getDB(ctx context.Context) *gorm.DB {
	return s.svc.GetDB(ctx).Model(new(models.AutoIngestLog))
}
