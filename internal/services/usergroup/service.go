package usergroup

import (
	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"gorm.io/gorm"
)

type Service interface {
	Add(ctx context.Context, req *AddRequest) (resp *AddResponse, err error)
	Delete(ctx context.Context, req *DeleteRequest) (err error)
	ModifyName(ctx context.Context, req *ModifyNameRequest) (err error)
	List(ctx context.Context, req *ListRequest) (list []*models.UserGroup, err error)
	Count(ctx context.Context, req *ListRequest) (count int64, err error)
	Query(ctx context.Context, gid int64) (*models.UserGroup, error)
	BatchQuery(ctx context.Context, idList []int64) ([]*models.UserGroup, error)
}

type service struct {
	svc bootstrap.ServiceContext
}

// NewService 创建用户组服务
func NewService(svc bootstrap.ServiceContext) Service {
	return &service{
		svc: svc,
	}
}

func (s *service) getDB(ctx context.Context) *gorm.DB {
	return s.svc.GetDB(ctx).Model(new(models.UserGroup))
}
