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
		if strings.EqualFold(strings.TrimSpace(setting.CasTargetType), "family") {
			if setting.CasFamilyTargetTokenId <= 0 || strings.TrimSpace(setting.CasFamilyTargetFamilyId) == "" {
				ctx.String(http.StatusConflict, "cas target not configured")
				return
			}
		} else {
			if setting.CasPersonTargetTokenId <= 0 || strings.TrimSpace(setting.CasPersonTargetFolderId) == "" {
				ctx.String(http.StatusConflict, "cas target not configured")
				return
			}
		}

		if directLink, ok := h.tryDirectPlaybackLink(ctx, record, setting); ok {
			ctx.Redirect(http.StatusFound, directLink)
			return
		}

		targetFolderID, familyID, err := h.resolvePlaybackTargetFolder(ctx, record, setting)
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
		targetTokenID := setting.CasPersonTargetTokenId
		if destinationType == casrestore.DestinationTypeFamily && setting.CasFamilyTargetTokenId > 0 {
			targetTokenID = setting.CasFamilyTargetTokenId
		}

		result, err := h.casRestoreService.EnsureRestoredFromLocalCAS(ctx.GetContext(), casrestore.RestoreRequest{
			StorageID:       record.StorageID,
			MountPointID:    record.MountPointID,
			TargetTokenID:   targetTokenID,
			CasFileID:       record.CasFileID,
			CasFileName:     record.CasFileName,
			LocalCasPath:    localCASPath,
			UploadRoute:     casrestore.UploadRouteFamily,
			DestinationType: destinationType,
			TargetFolderID:  targetFolderID,
			FamilyID:        familyID,
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
		setting.CasTargetType = string(destinationType)
		if destinationType == casrestore.DestinationTypeFamily && strings.TrimSpace(setting.CasFamilyTargetFamilyId) == "" {
			if result != nil && result.FamilyID > 0 {
				setting.CasFamilyTargetFamilyId = strconv.FormatInt(result.FamilyID, 10)
			} else if familyID > 0 {
				setting.CasFamilyTargetFamilyId = strconv.FormatInt(familyID, 10)
			}
		}

		directLink, ok := h.tryDirectPlaybackLink(ctx, record, setting)
		if !ok || strings.TrimSpace(directLink) == "" {
			ctx.String(http.StatusBadGateway, "failed to get restored download link")
			return
		}
		ctx.Redirect(http.StatusFound, directLink)
	}
}

func (h *handler) resolvePlaybackTargetFolder(ctx *httpcontext.Context, record *models.CasMediaRecord, setting models.SettingAddition) (string, int64, error) {
	if strings.TrimSpace(setting.CasTargetType) != "family" {
		baseTargetFolderID := strings.TrimSpace(setting.CasPersonTargetFolderId)
		if baseTargetFolderID == "" {
			baseTargetFolderID = "-11"
		}
		return baseTargetFolderID, 0, nil
	}
	baseTargetFolderID := strings.TrimSpace(setting.CasFamilyTargetFolderId)
	if baseTargetFolderID == "" {
		baseTargetFolderID = "-11"
	}
	relDir := strings.Trim(strings.TrimPrefix(path.Dir(strings.TrimSpace(record.CasFilePath)), "/"), " ")
	if relDir == "" || relDir == "." || h.appSessionService == nil {
		return baseTargetFolderID, 0, nil
	}
	session, err := h.appSessionService.GetByTokenID(ctx.GetContext(), setting.CasFamilyTargetTokenId)
	if err != nil {
		return "", 0, err
	}
	webToken := cloudpan.WebLoginToken{}
	if cookie := cloudpan.RefreshCookieToken(session.Token.SessionKey); cookie != "" {
		webToken.CookieLoginUser = cookie
	}
	panClient := cloudpan.NewPanClient(webToken, session.Token)
	familyID := int64(0)
	if strings.TrimSpace(setting.CasFamilyTargetFamilyId) != "" {
		if parsed, perr := strconv.ParseInt(strings.TrimSpace(setting.CasFamilyTargetFamilyId), 10, 64); perr == nil {
			familyID = parsed
		}
	}
	if familyID <= 0 {
		return "", 0, fmt.Errorf("未配置家庭恢复目标(casFamilyTargetFamilyId)")
	}
	familyParentID := baseTargetFolderID
	if familyParentID == "-11" {
		familyParentID = ""
	}
	folder, apiErr := panClient.AppMkdirRecursive(familyID, familyParentID, relDir, 0, strings.Split(relDir, "/"))
	if apiErr != nil {
		return "", 0, apiErr
	}
	if folder == nil || strings.TrimSpace(folder.FileId) == "" {
		return "", 0, fmt.Errorf("创建播放目标目录失败: 未返回最终目标目录ID relativeDir=%s", relDir)
	}
	return strings.TrimSpace(folder.FileId), familyID, nil
}

func (h *handler) tryDirectPlaybackLink(ctx *httpcontext.Context, record *models.CasMediaRecord, setting models.SettingAddition) (string, bool) {
	if record == nil || strings.TrimSpace(record.RestoredFileID) == "" {
		return "", false
	}
	targetTokenID := setting.CasPersonTargetTokenId
	if strings.EqualFold(strings.TrimSpace(setting.CasTargetType), "family") {
		targetTokenID = setting.CasFamilyTargetTokenId
	}
	if targetTokenID <= 0 {
		return "", false
	}
	token, err := h.cloudTokenService.Query(ctx.GetContext(), targetTokenID)
	if err != nil {
		return "", false
	}
	authToken := cloudbridgeSvi.NewAuthToken(token.AccessToken, token.ExpiresIn)

	if strings.EqualFold(strings.TrimSpace(setting.CasTargetType), "family") {
		familyID := strings.TrimSpace(setting.CasFamilyTargetFamilyId)
		if familyID == "" && h.appSessionService != nil {
			if session, serr := h.appSessionService.GetByTokenID(ctx.GetContext(), targetTokenID); serr == nil && session != nil {
				webToken := cloudpan.WebLoginToken{}
				if cookie := cloudpan.RefreshCookieToken(session.Token.SessionKey); cookie != "" {
					webToken.CookieLoginUser = cookie
				}
				panClient := cloudpan.NewPanClient(webToken, session.Token)
				if families, ferr := panClient.AppFamilyGetFamilyList(); ferr == nil && families != nil {
					for _, item := range families.FamilyInfoList {
						if item != nil && item.UserRole == 1 {
							familyID = strconv.FormatInt(item.FamilyId, 10)
							break
						}
					}
					if familyID == "" {
						for _, item := range families.FamilyInfoList {
							if item != nil {
								familyID = strconv.FormatInt(item.FamilyId, 10)
								break
							}
						}
					}
				}
			}
		}
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
