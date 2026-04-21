package autoingest

import (
	"fmt"
	"path"
	"strings"

	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	cloudbridge "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"go.uber.org/zap"
)

func buildPanClient(session *appsession.Session) *cloudpan.PanClient {
	if session == nil {
		return nil
	}
	webToken := cloudpan.WebLoginToken{}
	if cookie := cloudpan.RefreshCookieToken(session.Token.SessionKey); cookie != "" {
		webToken.CookieLoginUser = cookie
	}
	return cloudpan.NewPanClient(webToken, session.Token)
}

func (h *handler) tryCollectSubscribeCAS(ctx context.Context, itemPath string, item *cloudbridge.ShareResourceInfo) error {
	cfg := shared.SettingAddition
	if !cfg.CasTargetEnabled || !cfg.CasAutoCollectEnabled {
		return nil
	}
	if cfg.CasTargetTokenId <= 0 {
		return nil
	}
	if item == nil || item.IsFolder {
		return nil
	}
	if !strings.HasSuffix(strings.ToLower(item.Name), ".cas") {
		return nil
	}

	session, err := h.appSessionService.GetByTokenID(ctx, cfg.CasTargetTokenId)
	if err != nil {
		return fmt.Errorf("获取CAS目标App会话失败: %w", err)
	}
	panClient := buildPanClient(session)
	if panClient == nil {
		return fmt.Errorf("创建CAS目标PanClient失败")
	}

	targetFolderID := cfg.CasTargetFolderId
	if targetFolderID == "" {
		targetFolderID = "-11"
	}

	if cfg.CasAutoCollectPreservePath && cfg.CasTargetType == "person" {
		relDir := strings.TrimSpace(path.Dir(strings.TrimPrefix(itemPath, "/")))
		if relDir != "" && relDir != "." {
			folder, apiErr := panClient.AppMkdirRecursive(0, targetFolderID, relDir, 0, strings.Split(relDir, "/"))
			if apiErr != nil {
				return fmt.Errorf("创建CAS归集目录失败: %w", apiErr)
			}
			if folder != nil && folder.FileId != "" {
				targetFolderID = folder.FileId
			}
		}
	}

	if cfg.CasTargetType != "person" {
		return fmt.Errorf("当前自动归集仅先支持保存到个人目录")
	}

	ok, apiErr := panClient.ShareSave(item.AccessCode, "", targetFolderID)
	if apiErr != nil {
		return fmt.Errorf("自动归集CAS失败: %w", apiErr)
	}
	if !ok {
		return fmt.Errorf("自动归集CAS失败: ShareSave未成功")
	}

	ctx.Info("订阅CAS自动归集成功",
		zap.String("name", item.Name),
		zap.String("targetFolderId", targetFolderID),
		zap.String("accessUrl", item.AccessCode),
	)
	return nil
}
