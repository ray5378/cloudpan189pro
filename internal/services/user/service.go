package user

import (
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"gorm.io/gorm"
)

type Service interface {
	Add(ctx context.Context, req *AddRequest, opts ...AddOptionFunc) (resp *AddResponse, err error)
	Del(ctx context.Context, req *DelRequest) error
	Query(ctx context.Context, uid int64) (*models.User, error)
	QueryByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, uid int64, fields ...utils.Field) error
	List(ctx context.Context, req *ListRequest) (list []*models.User, err error)
	Count(ctx context.Context, req *ListRequest) (count int64, err error)
	ModifyPass(ctx context.Context, uid int64, password string) error
	BindGroup(ctx context.Context, req *BindGroupRequest) error

	// ParseAccessToken 解析 token
	ParseAccessToken(tokenString string) (int64, string, int, error)
	// GenerateAccessToken 生成访问Token
	GenerateAccessToken(userId int64, username string, userVersion int) (string, error)
	// GenerateRefreshToken 生成刷新Token
	GenerateRefreshToken(userId int64, username string, userVersion int) (string, error)
	//ParseRefreshToken 解析刷新Token
	ParseRefreshToken(tokenString string) (int64, string, int, error)
	GetExpire() int64
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
	return s.svc.GetDB(ctx).Model(new(models.User))
}

func (s *service) GetExpire() int64 {
	return int64(AccessTokenExpire / time.Second)
}
