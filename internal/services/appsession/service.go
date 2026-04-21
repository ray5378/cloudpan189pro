package appsession

import (
	"github.com/pkg/errors"
	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
)

type Service interface {
	GetByTokenID(ctx context.Context, tokenID int64) (*Session, error)
	GetByMountPointID(ctx context.Context, mountPointID int64) (*Session, error)
}

type service struct {
	svc               bootstrap.ServiceContext
	cloudTokenService cloudtokenSvi.Service
	mountPointService mountpointSvi.Service
}

func NewService(
	svc bootstrap.ServiceContext,
	cloudTokenService cloudtokenSvi.Service,
	mountPointService mountpointSvi.Service,
) Service {
	return &service{
		svc:               svc,
		cloudTokenService: cloudTokenService,
		mountPointService: mountPointService,
	}
}

func (s *service) GetByMountPointID(ctx context.Context, mountPointID int64) (*Session, error) {
	mp, err := s.mountPointService.Query(ctx, mountPointID)
	if err != nil {
		return nil, err
	}
	return s.GetByTokenID(ctx, mp.TokenId)
}

func (s *service) GetByTokenID(ctx context.Context, tokenID int64) (*Session, error) {
	cloudToken, err := s.cloudTokenService.Query(ctx, tokenID)
	if err != nil {
		return nil, err
	}
	return s.getFromCloudToken(ctx, cloudToken)
}

func (s *service) getFromCloudToken(ctx context.Context, cloudToken *models.CloudToken) (*Session, error) {
	if cloudToken == nil {
		return nil, errors.New("云盘令牌不存在")
	}
	if cloudToken.LoginType != models.LoginTypePassword {
		return nil, errors.New("当前仅支持密码登录令牌生成App会话")
	}
	if cloudToken.Username == "" || cloudToken.Password == "" {
		return nil, errors.New("云盘令牌缺少用户名或密码")
	}

	appToken, apiErr := cloudpan.AppLogin(cloudToken.Username, cloudToken.Password)
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "获取App会话失败")
	}
	return &Session{Token: *appToken}, nil
}
