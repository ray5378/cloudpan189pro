package media

import (
	"encoding/json"

	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/types/topic"
)

// RebuildStrmFile 重建strm文件
// @Summary 重建strm文件
// @Description 扫描所有挂载点并重新生成strm文件
// @Tags 媒体操作
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} httpcontext.Response "重建任务已提交"
// @Failure 400 {object} httpcontext.Response "媒体功能未启用，code=xxxx"
// @Failure 400 {object} httpcontext.Response "提交重建任务失败，code=xxxx"
// @Failure 401 {object} httpcontext.Response "未授权访问"
// @Failure 403 {object} httpcontext.Response "权限不足"
// @Router /api/media/rebuild_strm_file [post]
func (h *handler) RebuildStrmFile() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		// 检查媒体功能是否启用
		cfg, err := h.mediaConfigService.Query(ctx.GetContext())
		if err != nil {
			ctx.Fail(codeMediaNotEnabled.WithError(err))

			return
		}

		if !cfg.Enable {
			ctx.Fail(codeMediaNotEnabled)

			return
		}

		// 推送重建strm文件任务到消息队列
		taskReq := &topic.MediaRebuildStrmFileRequest{}
		body, _ := json.Marshal(taskReq)

		if err = h.taskEngine.PushMessage(
			ctx.GetContext().
				WithValue(consts.CtxKeyInvokeHandlerName, "媒体重建strm文件执行器"),
			taskReq.Topic(), body); err != nil {
			ctx.Fail(codeRebuildFailed.WithError(err))

			return
		}

		ctx.Success()
	}
}
