package media

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	schedulerHandler "github.com/xxcheng123/cloudpan189-share/internal/handler/scheduler"
)

// RunFallbackRecycleOnce 手动立即触发一次 CAS 恢复文件清理。
// @Summary 手动触发一次 CAS 恢复记录清理
// @Description 立即执行一次原有“按恢复记录驱动”的到期清理链，复用自动清理同一套回收逻辑
// @Tags 媒体管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} httpcontext.Response "触发成功"
// @Failure 400 {object} httpcontext.Response "触发失败"
// @Router /api/media/run_fallback_recycle_once [post]
func (h *handler) RunFallbackRecycleOnce() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		if err := schedulerHandler.RunCASFallbackRecycleOnce(ctx.GetContext(), h.casRecordService, h.appSessionService, h.cloudTokenService); err != nil {
			ctx.Fail(codeRunFallbackRecycleFailed.WithError(err))
			return
		}
		ctx.Success(map[string]any{"ok": true})
	}
}
