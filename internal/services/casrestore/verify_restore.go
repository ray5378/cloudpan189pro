package casrestore

import (
	"fmt"
	"strings"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
)

func (s *service) verifyRestoredInPersonFolder(ctx appctx.Context, mountPointID int64, targetFolderID string, expectedName string) (string, string, error) {
	token, err := s.loadMountAuthToken(ctx, mountPointID)
	if err != nil {
		return "", "", err
	}
	resp, err := s.cloudBridgeService.PersonFileList(ctx, token, targetFolderID, 1, 200)
	if err != nil {
		return "", "", err
	}
	for _, item := range resp.Data {
		if item == nil || item.IsFolder == 1 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(expectedName)) {
			return item.ID, item.Name, nil
		}
	}
	return "", "", fmt.Errorf("未在目标目录中找到恢复后的文件: %s", expectedName)
}

func (s *service) loadMountAuthToken(ctx appctx.Context, mountPointID int64) (cloudbridgeSvi.AuthToken, error) {
	mountPointSvc := mountpointSvi.NewService(s.svc, cloudtokenSvi.NewService(s.svc), s.cloudBridgeService)
	cloudTokenSvc := cloudtokenSvi.NewService(s.svc)
	mp, err := mountPointSvc.Query(ctx, mountPointID)
	if err != nil {
		return nil, err
	}
	token, err := cloudTokenSvc.Query(ctx, mp.TokenId)
	if err != nil {
		return nil, err
	}
	return cloudbridgeSvi.NewAuthToken(token.AccessToken, token.ExpiresIn), nil
}

func normalizeRestoreResult(result *RestoreResult, fileID, fileName, targetFolderID string) *RestoreResult {
	if result == nil {
		result = &RestoreResult{}
	}
	if fileID != "" {
		result.RestoredFileID = fileID
	}
	if fileName != "" {
		result.RestoredFileName = fileName
	}
	if targetFolderID != "" {
		result.TargetFolderID = targetFolderID
	}
	return result
}

func isRecoverableFallbackTarget(vf *models.VirtualFile) bool {
	return vf != nil && (vf.OsType == models.OsTypePersonFile || vf.OsType == models.OsTypeFamilyFile)
}
