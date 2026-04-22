package appsession

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

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

type appRefreshUserSessionResp struct {
	XMLName             xml.Name `xml:"userSession"`
	LoginName           string   `xml:"loginName"`
	SessionKey          string   `xml:"sessionKey"`
	SessionSecret       string   `xml:"sessionSecret"`
	KeepAlive           int      `xml:"keepAlive"`
	GetFileDiffSpan     int      `xml:"getFileDiffSpan"`
	GetUserInfoSpan     int      `xml:"getUserInfoSpan"`
	FamilySessionKey    string   `xml:"familySessionKey"`
	FamilySessionSecret string   `xml:"familySessionSecret"`
}

func getSessionByAccessToken(accessToken string) (*appRefreshUserSessionResp, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, errors.New("accessToken为空")
	}
	url := fmt.Sprintf("https://api.cloud.189.cn/getSessionForPC.action?appId=%s&accessToken=%s&clientSn=%s&clientType=%s&version=%s&model=%s", "8025431004", accessToken, "cloudpan189pro", "TELEMAC", "1.0.0", "PC")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Request-ID", "cloudpan189pro-appsession")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	item := &appRefreshUserSessionResp{}
	if err := xml.NewDecoder(resp.Body).Decode(item); err != nil {
		return nil, err
	}
	if strings.TrimSpace(item.SessionKey) == "" {
		return nil, errors.New("通过accessToken刷新App会话失败")
	}
	return item, nil
}

func (s *service) getFromCloudToken(ctx context.Context, cloudToken *models.CloudToken) (*Session, error) {
	if cloudToken == nil {
		return nil, errors.New("云盘令牌不存在")
	}

	// 优先使用现有 accessToken 刷出 app session，避免每次都再次用户名密码 AppLogin。
	if refreshed, err := getSessionByAccessToken(cloudToken.AccessToken); err == nil && refreshed != nil {
		return &Session{Token: cloudpan.AppLoginToken{
			SessionKey:          refreshed.SessionKey,
			SessionSecret:       refreshed.SessionSecret,
			FamilySessionKey:    refreshed.FamilySessionKey,
			FamilySessionSecret: refreshed.FamilySessionSecret,
			AccessToken:         strings.TrimSpace(cloudToken.AccessToken),
		}}, nil
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
