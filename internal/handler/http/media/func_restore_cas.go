package media

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	casrestoreSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrestore"
)

// restoreCasRequest 手动触发 CAS 恢复。
// 注意：uploadRoute 表示秒传/上传路线；destinationType 表示最终目录归属。
type restoreCasRequest struct {
	StorageID       int64                         `json:"storageId" binding:"required" example:"1"`
	MountPointID    int64                         `json:"mountPointId" binding:"required" example:"1"`
	CasFileID       string                        `json:"casFileId" binding:"required" example:"123456789"`
	CasFileName     string                        `json:"casFileName" binding:"omitempty" example:"movie.cas"`
	CasVirtualID    int64                         `json:"casVirtualId" binding:"required" example:"1001"`
	UploadRoute     casrestoreSvi.UploadRoute     `json:"uploadRoute" binding:"omitempty,oneof=family person" example:"family"`
	DestinationType casrestoreSvi.DestinationType `json:"destinationType" binding:"required,oneof=family person" example:"family"`
	TargetFolderID  string                        `json:"targetFolderId" binding:"required" example:"-11"`
}

// RestoreCas 手动触发单个 CAS 恢复。
// @Summary 手动恢复CAS文件
// @Description 根据 .cas 元数据立刻执行一次恢复。uploadRoute 表示上传路线，destinationType 表示最终目录归属。
// @Tags 媒体操作
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body restoreCasRequest true "恢复请求"
// @Success 200 {object} httpcontext.Response{data=casrestore.RestoreResult} "恢复成功"
// @Failure 400 {object} httpcontext.Response "参数验证失败，code=99998"
// @Failure 400 {object} httpcontext.Response "恢复失败"
// @Failure 401 {object} httpcontext.Response "未授权访问"
// @Failure 403 {object} httpcontext.Response "权限不足"
// @Router /api/media/restore_cas [post]
func (h *handler) RestoreCas() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		req := new(restoreCasRequest)
		if err := ctx.ShouldBindJSON(req); err != nil {
			ctx.AbortWithInvalidParams(err)
			return
		}

		restoreReq := casrestoreSvi.RestoreRequest{
			StorageID:       req.StorageID,
			MountPointID:    req.MountPointID,
			CasFileID:       req.CasFileID,
			CasFileName:     req.CasFileName,
			CasVirtualID:    req.CasVirtualID,
			UploadRoute:     req.UploadRoute,
			DestinationType: req.DestinationType,
			TargetFolderID:  req.TargetFolderID,
		}
		if restoreReq.UploadRoute == "" {
			restoreReq.UploadRoute = casrestoreSvi.UploadRouteFamily
		}

		resp, err := h.casRestoreService.EnsureRestored(ctx.GetContext(), restoreReq)
		if err != nil {
			ctx.Fail(codeRestoreCasFailed.WithError(err))
			return
		}

		ctx.Success(resp)
	}
}
