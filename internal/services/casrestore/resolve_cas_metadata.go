package casrestore

import (
	"fmt"
	"strings"

	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	virtualfileSvi "github.com/xxcheng123/cloudpan189-share/internal/services/virtualfile"
)

type casMetadataResolver struct {
	cloudBridgeService cloudbridgeSvi.Service
	cloudTokenService  cloudtokenSvi.Service
	mountPointService  mountpointSvi.Service
	virtualFileService virtualfileSvi.Service
}

func (s *service) newCASMetadataResolver() *casMetadataResolver {
	cloudTokenSvc := cloudtokenSvi.NewService(s.svc)
	cloudBridgeSvc := cloudbridgeSvi.NewService(s.svc)
	return &casMetadataResolver{
		cloudBridgeService: cloudBridgeSvc,
		cloudTokenService:  cloudTokenSvc,
		mountPointService:  mountpointSvi.NewService(s.svc, cloudTokenSvc, cloudBridgeSvc),
		virtualFileService: virtualfileSvi.NewService(s.svc),
	}
}

func (r *casMetadataResolver) Resolve(ctx appctx.Context, mountPointID, fileID int64) (*casparser.CasInfo, *models.VirtualFile, error) {
	vf, err := r.virtualFileService.Query(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	if !casparser.IsCasFile(vf.Name) {
		return nil, nil, fmt.Errorf("目标文件不是.cas: %s", vf.Name)
	}

	mp, err := r.mountPointService.Query(ctx, mountPointID)
	if err != nil {
		return nil, nil, err
	}
	token, err := r.cloudTokenService.Query(ctx, mp.TokenId)
	if err != nil {
		return nil, nil, err
	}
	authToken := cloudbridgeSvi.NewAuthToken(token.AccessToken, token.ExpiresIn)

	link, err := r.getDownloadLink(ctx, authToken, vf)
	if err != nil {
		return nil, nil, err
	}
	content, err := r.downloadContent(ctx, link)
	if err != nil {
		return nil, nil, err
	}
	info, err := casparser.ParseCasContent(content)
	if err != nil {
		return nil, nil, err
	}
	return info, vf, nil
}

func (r *casMetadataResolver) getDownloadLink(ctx appctx.Context, token cloudbridgeSvi.AuthToken, vf *models.VirtualFile) (string, error) {
	switch vf.OsType {
	case models.OsTypePersonFile:
		return r.cloudBridgeService.PersonDownloadLink(ctx, token, vf.CloudId)
	case models.OsTypeFamilyFile:
		familyID, _ := vf.GetAddition(consts.FileAdditionKeyFamilyId).(string)
		if strings.TrimSpace(familyID) == "" {
			return "", fmt.Errorf("家庭文件缺少familyId")
		}
		return r.cloudBridgeService.FamilyDownloadLink(ctx, token, familyID, vf.CloudId)
	default:
		return "", fmt.Errorf("暂不支持该来源类型的.cas下载: %s", vf.OsType)
	}
}

func (r *casMetadataResolver) downloadContent(ctx appctx.Context, link string) ([]byte, error) {
	resp, err := ctx.HTTPClient().R().SetContext(ctx).Get(link)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("下载.cas失败: status=%d", resp.StatusCode())
	}
	return resp.Bytes(), nil
}
