package mediaconfig

import (
	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"gorm.io/gorm"
)

type Service interface {
	// Query 查询当前媒体配置（单条记录）
	Query(ctx context.Context) (*models.MediaConfig, error)
	// Update 更新配置指定字段
	Update(ctx context.Context, fields ...utils.Field) error
	// Init 初始化媒体配置
	Init(ctx context.Context, req *InitRequest) error
	// Toggle 切换启用状态
	Toggle(ctx context.Context, enable bool) error
}

type service struct {
	svc bootstrap.ServiceContext
}

func NewService(svc bootstrap.ServiceContext) Service {
	return &service{
		svc: svc,
	}
}

func (s *service) getDB(ctx context.Context) *gorm.DB {
	return s.svc.GetDB(ctx).Model(new(models.MediaConfig))
}
