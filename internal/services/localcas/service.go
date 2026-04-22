package localcas

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	localstrmSvi "github.com/xxcheng123/cloudpan189-share/internal/services/localstrm"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	"go.uber.org/zap"
)

type Service interface {
	DownloadToLocal(ctx appctx.Context, file *models.VirtualFile) (string, error)
}

type service struct {
	svc                bootstrap.ServiceContext
	cloudBridgeService cloudbridgeSvi.Service
	cloudTokenService  cloudtokenSvi.Service
	mountPointService  mountpointSvi.Service
	localSTRMService   localstrmSvi.Service
}

func NewService(svc bootstrap.ServiceContext) Service {
	cloudTokenSvc := cloudtokenSvi.NewService(svc)
	cloudBridgeSvc := cloudbridgeSvi.NewService(svc)
	return &service{
		svc:                svc,
		cloudBridgeService: cloudBridgeSvc,
		cloudTokenService:  cloudTokenSvc,
		mountPointService:  mountpointSvi.NewService(svc, cloudTokenSvc, cloudBridgeSvc),
		localSTRMService:   localstrmSvi.NewService(svc),
	}
}

func (s *service) DownloadToLocal(ctx appctx.Context, file *models.VirtualFile) (string, error) {
	if file == nil {
		return "", fmt.Errorf("file不能为空")
	}
	if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(file.Name)), ".cas") {
		return "", fmt.Errorf("目标文件不是.cas: %s", file.Name)
	}
	mp, err := s.mountPointService.Query(ctx, file.TopId)
	if err != nil {
		return "", err
	}
	token, err := s.cloudTokenService.Query(ctx, mp.TokenId)
	if err != nil {
		return "", err
	}
	authToken := cloudbridgeSvi.NewAuthToken(token.AccessToken, token.ExpiresIn)

	link, err := s.getDownloadLink(ctx, authToken, file)
	if err != nil {
		return "", err
	}

	sourceDirPath, _ := file.Addition.String(consts.FileAdditionKeySourceDirPath)
	sourceDirPath = strings.TrimSpace(sourceDirPath)
	relDir := strings.TrimPrefix(sourceDirPath, "/")
	localRoot := "/local_cas"
	localDir := filepath.Join(localRoot, filepath.FromSlash(relDir))
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return "", err
	}
	localPath := filepath.Join(localDir, file.Name)

	resp, err := ctx.HTTPClient().R().SetContext(ctx).Get(link)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return "", fmt.Errorf("下载订阅CAS失败: status=%d", resp.StatusCode())
	}
	f, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err = io.Copy(f, strings.NewReader(string(resp.Bytes()))); err != nil {
		return "", err
	}
	ctx.Info("订阅CAS已下载到本地",
		zap.String("file_name", file.Name),
		zap.String("local_path", localPath),
	)
	if s.localSTRMService != nil {
		if strmPath, recordID, err := s.localSTRMService.EnsureForLocalCAS(ctx, file, localPath); err != nil {
			ctx.Error("本地CAS生成STRM失败",
				zap.String("file_name", file.Name),
				zap.String("local_path", localPath),
				zap.Error(err),
			)
		} else {
			ctx.Info("本地CAS已自动生成STRM",
				zap.String("file_name", file.Name),
				zap.String("local_path", localPath),
				zap.String("strm_path", strmPath),
				zap.Int64("cas_record_id", recordID),
			)
		}
	}
	return localPath, nil
}

func (s *service) getDownloadLink(ctx appctx.Context, token cloudbridgeSvi.AuthToken, vf *models.VirtualFile) (string, error) {
	switch vf.OsType {
	case models.OsTypeSubscribeShareFile:
		shareId, ok := vf.Addition.Int64(consts.FileAdditionKeyShareId)
		if !ok || shareId <= 0 {
			return "", fmt.Errorf("订阅分享文件缺少shareId")
		}
		return s.cloudBridgeService.ShareDownloadLink(ctx, token, shareId, vf.CloudId)
	case models.OsTypePersonFile:
		return s.cloudBridgeService.PersonDownloadLink(ctx, token, vf.CloudId)
	case models.OsTypeFamilyFile:
		familyID, ok := vf.Addition.String(consts.FileAdditionKeyFamilyId)
		if !ok || strings.TrimSpace(familyID) == "" {
			return "", fmt.Errorf("家庭文件缺少familyId")
		}
		return s.cloudBridgeService.FamilyDownloadLink(ctx, token, familyID, vf.CloudId)
	default:
		return "", fmt.Errorf("不支持该来源类型的CAS本地下载: %s", vf.OsType)
	}
}
