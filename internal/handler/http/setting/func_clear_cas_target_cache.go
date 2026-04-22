package setting

import "github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"

// ClearCasTargetCache 清空 CAS 目标目录缓存（仅管理员）
// @Summary 清空 CAS 目标目录缓存
// @Description 清空表 cas_target_dir_caches，仅管理员可操作
// @Tags 系统设置
// @Accept json
// @Produce json
// @Success 200 {object} httpcontext.Response{data=map[string]interface{}} "清空成功"
// @Failure 400 {object} httpcontext.Response "清空CAS缓存失败"
// @Failure 401 {object} httpcontext.Response "未授权访问"
// @Failure 403 {object} httpcontext.Response "权限不足"
// @Router /api/setting/clear_cas_target_cache [post]
func (h *handler) ClearCasTargetCache() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		if h.casTargetCacheService == nil {
			ctx.Success(map[string]any{"deleted": int64(0)})
			return
		}
		deleted, err := h.casTargetCacheService.ClearAll(ctx.GetContext())
		if err != nil {
			ctx.Fail(codeModifyAdditionFailed.WithError(err))
			return
		}
		ctx.Success(map[string]any{"deleted": deleted})
	}
}
