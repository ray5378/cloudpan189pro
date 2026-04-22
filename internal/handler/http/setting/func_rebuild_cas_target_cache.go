package setting

import (
	"strings"
	"time"

	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"go.uber.org/zap"
)

// RebuildCasTargetCache 重建 CAS 目标目录缓存（仅管理员）
// @Summary 重建 CAS 目标目录缓存
// @Description 只读扫描当前 CAS 目标目录及缓存表里已存在的目录，重建本地缓存；不会创建云盘目录
// @Tags 系统设置
// @Accept json
// @Produce json
// @Success 200 {object} httpcontext.Response{data=map[string]interface{}} "重建成功"
// @Failure 400 {object} httpcontext.Response "重建CAS缓存失败"
// @Failure 401 {object} httpcontext.Response "未授权访问"
// @Failure 403 {object} httpcontext.Response "权限不足"
// @Router /api/setting/rebuild_cas_target_cache [post]
func (h *handler) RebuildCasTargetCache() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		if h.casTargetCacheService == nil || h.appSessionService == nil {
			ctx.Success(map[string]any{"dirCount": 0, "itemCount": 0})
			return
		}

		tokenID := shared.SettingAddition.CasTargetTokenId
		baseTargetFolderID := strings.TrimSpace(shared.SettingAddition.CasTargetFolderId)
		if tokenID <= 0 || baseTargetFolderID == "" || baseTargetFolderID == "0" {
			ctx.Success(map[string]any{"dirCount": 0, "itemCount": 0})
			return
		}

		session, err := h.appSessionService.GetByTokenID(ctx.GetContext(), tokenID)
		if err != nil {
			ctx.Fail(codeModifyAdditionFailed.WithError(err))
			return
		}
		panClient := buildSettingPanClient(session)
		if panClient == nil {
			ctx.Fail(codeModifyAdditionFailed.WithError(err))
			return
		}

		dirMap := map[string]struct{}{baseTargetFolderID: {}}
		list, err := h.casTargetCacheService.ListDistinctDirs(ctx.GetContext())
		if err != nil {
			ctx.Fail(codeModifyAdditionFailed.WithError(err))
			return
		}
		for _, item := range list {
			if item == nil || item.TargetTokenID != tokenID {
				continue
			}
			folderID := strings.TrimSpace(item.TargetFolderID)
			if folderID == "" {
				continue
			}
			dirMap[folderID] = struct{}{}
		}

		dirCount := 0
		itemCount := 0
		for folderID := range dirMap {
			param := cloudpan.NewAppFileListParam()
			param.FileId = folderID
			param.PageSize = 200
			result, apiErr := panClient.AppGetAllFileList(param)
			if apiErr != nil {
				ctx.GetContext().Warn("重建CAS缓存时读取目标目录失败", zap.Error(apiErr), zap.String("targetFolderId", folderID))
				continue
			}
			now := time.Now()
			items := make([]*models.CasTargetDirCache, 0)
			if result != nil {
				for _, fi := range result.FileList {
					if fi == nil {
						continue
					}
					items = append(items, &models.CasTargetDirCache{
						TargetTokenID:  tokenID,
						TargetFolderID: folderID,
						FileName:       strings.TrimSpace(fi.FileName),
						IsDir:          fi.IsFolder,
						RefreshedAt:    now,
					})
				}
			}
			if err := h.casTargetCacheService.RefreshDir(ctx.GetContext(), tokenID, folderID, items); err != nil {
				ctx.Fail(codeModifyAdditionFailed.WithError(err))
				return
			}
			dirCount++
			itemCount += len(items)
		}

		ctx.Success(map[string]any{"dirCount": dirCount, "itemCount": itemCount})
	}
}
