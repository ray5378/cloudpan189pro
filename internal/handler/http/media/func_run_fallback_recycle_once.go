package media

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	schedulerHandler "github.com/xxcheng123/cloudpan189-share/internal/handler/scheduler"
)

// RunFallbackRecycleOnce 手动立即触发一次 CAS 恢复文件兜底扫描清理。
// @Summary 手动触发一次 CAS 兜底清理
// @Description 按当前 CAS 最终目录与恢复后文件留存时间，立即执行一次兜底扫描清理，并在结束后清空回收站
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
