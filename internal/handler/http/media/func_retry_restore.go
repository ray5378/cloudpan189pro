package media

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	casrestoreSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrestore"
)

// retryRestoreRequest 基于已有恢复记录重试恢复。
type retryRestoreRequest struct {
	RecordID        int64                         `json:"recordId" binding:"required" example:"1"`
	UploadRoute     casrestoreSvi.UploadRoute     `json:"uploadRoute" binding:"omitempty,oneof=family person" example:"family"`
	DestinationType casrestoreSvi.DestinationType `json:"destinationType" binding:"required,oneof=family person" example:"family"`
	TargetFolderID  string                        `json:"targetFolderId" binding:"omitempty" example:"-11"`
}

// RetryRestore 基于已有恢复记录重试恢复。
// @Summary 重试CAS恢复
// @Description 根据已有恢复记录重新触发恢复。若未传 targetFolderId，则默认沿用记录中的 restoredParentId。
// @Tags 媒体操作
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body retryRestoreRequest true "重试恢复请求"
// @Success 200 {object} httpcontext.Response{data=casrestore.RestoreResult} "重试成功"
// @Failure 400 {object} httpcontext.Response "参数验证失败，code=99998"
// @Failure 400 {object} httpcontext.Response "重试失败"
// @Failure 401 {object} httpcontext.Response "未授权访问"
// @Failure 403 {object} httpcontext.Response "权限不足"
// @Router /api/media/retry_restore [post]
func (h *handler) RetryRestore() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		req := new(retryRestoreRequest)
		if err := ctx.ShouldBindJSON(req); err != nil {
			ctx.AbortWithInvalidParams(err)
			return
		}

		resp, err := h.casRestoreService.RetryRecord(ctx.GetContext(), casrestoreSvi.RetryRequest{
			RecordID:        req.RecordID,
			UploadRoute:     req.UploadRoute,
			DestinationType: req.DestinationType,
			TargetFolderID:  req.TargetFolderID,
		})
		if err != nil {
			ctx.Fail(codeRetryRestoreFailed.WithError(err))
			return
		}

		ctx.Success(resp)
	}
}
