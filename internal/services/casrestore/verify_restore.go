package casrestore

import (
	"fmt"
	"strings"
	"time"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	"go.uber.org/zap"
)

func (s *service) verifyRestoredInPersonFolder(ctx appctx.Context, mountPointID int64, targetFolderID string, expectedName string) (string, string, error) {
	return s.verifyRestoredInPersonFolderByToken(ctx, mountPointID, 0, targetFolderID, expectedName)
}

func (s *service) verifyRestoredInPersonFolderByToken(ctx appctx.Context, mountPointID int64, targetTokenID int64, targetFolderID string, expectedName string) (string, string, error) {
	token, err := s.loadAuthToken(ctx, mountPointID, targetTokenID)
	if err != nil {
		return "", "", err
	}
	return s.waitForRestoredFile(ctx, expectedName, 30, time.Second, func() (string, string, error) {
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
		return "", "", nil
	})
}

func (s *service) verifyRestoredInFamilyFolder(ctx appctx.Context, mountPointID int64, familyID int64, targetFolderID string, expectedName string) (string, string, error) {
	return s.verifyRestoredInFamilyFolderByToken(ctx, mountPointID, 0, familyID, targetFolderID, expectedName)
}

func (s *service) verifyRestoredInFamilyFolderByToken(ctx appctx.Context, mountPointID int64, targetTokenID int64, familyID int64, targetFolderID string, expectedName string) (string, string, error) {
	token, err := s.loadAuthToken(ctx, mountPointID, targetTokenID)
	if err != nil {
		return "", "", err
	}
	return s.waitForRestoredFile(ctx, expectedName, 30, time.Second, func() (string, string, error) {
		resp, err := s.cloudBridgeService.FamilyFileList(ctx, token, fmt.Sprintf("%d", familyID), targetFolderID, 1, 200)
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
		return "", "", nil
	})
}

func (s *service) waitForRestoredFile(ctx appctx.Context, expectedName string, maxAttempts int, interval time.Duration, finder func() (string, string, error)) (string, string, error) {
	for i := 0; i < maxAttempts; i++ {
		fileID, fileName, err := finder()
		if err != nil {
			return "", "", err
		}
		if strings.TrimSpace(fileID) != "" {
			ctx.Info("CAS恢复校验命中目标文件",
				zap.String("expected_name", expectedName),
				zap.String("file_id", fileID),
				zap.String("file_name", fileName),
				zap.Int("attempt", i+1),
			)
			return fileID, fileName, nil
		}
		if i < maxAttempts-1 {
			time.Sleep(interval)
		}
	}
	return "", "", fmt.Errorf("未在目标目录中找到恢复后的文件: %s", expectedName)
}

func (s *service) loadMountAuthToken(ctx appctx.Context, mountPointID int64) (cloudbridgeSvi.AuthToken, error) {
	return s.loadAuthToken(ctx, mountPointID, 0)
}

func (s *service) loadAuthToken(ctx appctx.Context, mountPointID int64, targetTokenID int64) (cloudbridgeSvi.AuthToken, error) {
	cloudTokenSvc := cloudtokenSvi.NewService(s.svc)
	if targetTokenID > 0 {
		token, err := cloudTokenSvc.Query(ctx, targetTokenID)
		if err != nil {
			return nil, err
		}
		return cloudbridgeSvi.NewAuthToken(token.AccessToken, token.ExpiresIn), nil
	}
	mountPointSvc := mountpointSvi.NewService(s.svc, cloudTokenSvc, s.cloudBridgeService)
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
