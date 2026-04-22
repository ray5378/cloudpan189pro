package media

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casrestore"
	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"go.uber.org/zap"
)

func (h *handler) PlayCas() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		recordID, err := strconv.ParseInt(ctx.Param("recordId"), 10, 64)
		if err != nil || recordID <= 0 {
			ctx.String(http.StatusBadRequest, "invalid record id")
			return
		}

		record, err := h.casRecordService.Query(ctx.GetContext(), recordID)
		if err != nil {
			ctx.String(http.StatusNotFound, "cas record not found")
			return
		}

		setting := shared.SettingAddition
		if h.settingService != nil {
			if latest, qerr := h.settingService.Query(ctx.GetContext()); qerr == nil && latest != nil {
				setting = latest.Addition
			}
		}
		if setting.CasTargetTokenId <= 0 || strings.TrimSpace(setting.CasTargetFolderId) == "" {
			ctx.String(http.StatusConflict, "cas target not configured")
			return
		}

		if directLink, ok := h.tryDirectPlaybackLink(ctx, record, setting); ok {
			ctx.Redirect(http.StatusFound, directLink)
			return
		}

		targetFolderID, err := h.resolvePlaybackTargetFolder(ctx, record, setting)
		if err != nil {
			ctx.GetContext().Logger.Error("CAS播放入口解析目标目录失败", zap.Int64("record_id", recordID), zap.Error(err))
			ctx.String(http.StatusBadGateway, fmt.Sprintf("resolve playback target folder failed: %v", err))
			return
		}

		localCASPath := filepath.Join("/local_cas", filepath.FromSlash(strings.TrimPrefix(strings.TrimSpace(record.CasFilePath), "/")))
		destinationType := casrestore.DestinationTypePerson
		if strings.EqualFold(strings.TrimSpace(setting.CasTargetType), "family") {
			destinationType = casrestore.DestinationTypeFamily
		}

		result, err := h.casRestoreService.EnsureRestoredFromLocalCAS(ctx.GetContext(), casrestore.RestoreRequest{
			StorageID:       record.StorageID,
			MountPointID:    record.MountPointID,
			TargetTokenID:   setting.CasTargetTokenId,
			CasFileID:       record.CasFileID,
			CasFileName:     record.CasFileName,
			LocalCasPath:    localCASPath,
			UploadRoute:     casrestore.UploadRouteFamily,
			DestinationType: destinationType,
			TargetFolderID:  targetFolderID,
		})
		if err != nil {
			ctx.GetContext().Logger.Error("CAS播放入口恢复失败", zap.Int64("record_id", recordID), zap.Error(err))
			ctx.String(http.StatusBadGateway, fmt.Sprintf("cas restore failed: %v", err))
			return
		}

		if result == nil || strings.TrimSpace(result.RestoredFileID) == "" {
			ctx.String(http.StatusBadGateway, "cas restore returned empty file id")
			return
		}

		freshRecord, qerr := h.casRecordService.Query(ctx.GetContext(), recordID)
		if qerr == nil && freshRecord != nil {
			record = freshRecord
			record.RestoredFileID = result.RestoredFileID
		}

		directLink, ok := h.tryDirectPlaybackLink(ctx, record, setting)
		if !ok || strings.TrimSpace(directLink) == "" {
			ctx.String(http.StatusBadGateway, "failed to get restored download link")
			return
		}
		ctx.Redirect(http.StatusFound, directLink)
	}
}

func (h *handler) resolvePlaybackTargetFolder(ctx *httpcontext.Context, record *models.CasMediaRecord, setting models.SettingAddition) (string, error) {
	baseTargetFolderID := strings.TrimSpace(setting.CasTargetFolderId)
	if baseTargetFolderID == "" {
		baseTargetFolderID = "-11"
	}
	relDir := strings.Trim(strings.TrimPrefix(path.Dir(strings.TrimSpace(record.CasFilePath)), "/"), " ")
	if relDir == "" || relDir == "." || h.appSessionService == nil {
		return baseTargetFolderID, nil
	}
	session, err := h.appSessionService.GetByTokenID(ctx.GetContext(), setting.CasTargetTokenId)
	if err != nil {
		return "", err
	}
	webToken := cloudpan.WebLoginToken{}
	if cookie := cloudpan.RefreshCookieToken(session.Token.SessionKey); cookie != "" {
		webToken.CookieLoginUser = cookie
	}
	panClient := cloudpan.NewPanClient(webToken, session.Token)
	folder, apiErr := panClient.AppMkdirRecursive(0, baseTargetFolderID, relDir, 0, strings.Split(relDir, "/"))
	if apiErr != nil {
		return "", apiErr
	}
	if folder == nil || strings.TrimSpace(folder.FileId) == "" {
		return "", fmt.Errorf("创建播放目标目录失败: 未返回最终目标目录ID relativeDir=%s", relDir)
	}
	return strings.TrimSpace(folder.FileId), nil
}

func (h *handler) tryDirectPlaybackLink(ctx *httpcontext.Context, record *models.CasMediaRecord, setting models.SettingAddition) (string, bool) {
	if record == nil || strings.TrimSpace(record.RestoredFileID) == "" || setting.CasTargetTokenId <= 0 {
		return "", false
	}
	token, err := h.cloudTokenService.Query(ctx.GetContext(), setting.CasTargetTokenId)
	if err != nil {
		return "", false
	}
	authToken := cloudbridgeSvi.NewAuthToken(token.AccessToken, token.ExpiresIn)

	if strings.EqualFold(strings.TrimSpace(setting.CasTargetType), "family") {
		familyID := strings.TrimSpace(setting.CasTargetFamilyId)
		if familyID == "" {
			return "", false
		}
		link, err := h.cloudBridgeService.FamilyDownloadLink(ctx.GetContext(), authToken, familyID, record.RestoredFileID)
		if err != nil {
			return "", false
		}
		return link, true
	}

	link, err := h.cloudBridgeService.PersonDownloadLink(ctx.GetContext(), authToken, record.RestoredFileID)
	if err != nil {
		return "", false
	}
	return link, true
}
