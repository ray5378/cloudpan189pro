package media

import (
	"fmt"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
)

// restoreStatusRequest 查询单个 CAS 恢复状态。
// 支持三种定位方式：recordId、casVirtualId、casPath。
type restoreStatusRequest struct {
	RecordID     int64  `form:"recordId" binding:"omitempty" example:"1"`
	CasVirtualID int64  `form:"casVirtualId" binding:"omitempty" example:"1001"`
	CasPath      string `form:"casPath" binding:"omitempty" example:"/电影库/movie.cas"`
}

// RestoreStatus 查询单个 CAS 恢复状态。
// @Summary 查询CAS恢复状态
// @Description 根据 recordId、casVirtualId 或 casPath 查询单个 CAS 恢复记录
// @Tags 媒体操作
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param recordId query int false "恢复记录ID"
// @Param casVirtualId query int false "CAS虚拟文件ID"
// @Param casPath query string false "CAS虚拟文件路径"
// @Success 200 {object} httpcontext.Response{data=models.CasMediaRecord} "查询成功"
// @Failure 400 {object} httpcontext.Response "参数验证失败，code=99998"
// @Failure 400 {object} httpcontext.Response "查询失败"
// @Failure 401 {object} httpcontext.Response "未授权访问"
// @Failure 403 {object} httpcontext.Response "权限不足"
// @Router /api/media/restore_status [get]
func (h *handler) RestoreStatus() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		req := new(restoreStatusRequest)
		if err := ctx.ShouldBindQuery(req); err != nil {
			ctx.AbortWithInvalidParams(err)
			return
		}
		if req.RecordID == 0 && req.CasVirtualID == 0 && req.CasPath == "" {
			ctx.AbortWithInvalidParams(fmt.Errorf("recordId、casVirtualId、casPath 至少传一个"))
			return
		}

		record, err := h.findRestoreRecord(ctx, req)
		if err != nil {
			ctx.Fail(codeRestoreStatusFailed.WithError(err))
			return
		}

		ctx.Success(record)
	}
}

func (h *handler) findRestoreRecord(ctx *httpcontext.Context, req *restoreStatusRequest) (*models.CasMediaRecord, error) {
	if req.RecordID != 0 {
		return h.casRecordService.Query(ctx.GetContext(), req.RecordID)
	}

	vf, err := h.queryCASVirtualFile(ctx, &restoreCasRequest{
		CasVirtualID: req.CasVirtualID,
		CasPath:      req.CasPath,
	})
	if err != nil {
		return nil, err
	}
	if vf == nil {
		return nil, fmt.Errorf("无法定位CAS虚拟文件")
	}

	top, err := h.virtualfileService.QueryTop(ctx.GetContext(), vf.ID)
	if err != nil {
		return nil, err
	}
	if top == nil {
		return nil, fmt.Errorf("无法定位CAS文件所属挂载点")
	}

	mp, err := h.mountpointService.Query(ctx.GetContext(), top.ID)
	if err != nil {
		return nil, err
	}
	if mp == nil {
		return nil, fmt.Errorf("无法定位CAS文件挂载点记录")
	}

	return h.casRecordService.QueryByStorageAndCasFileID(ctx.GetContext(), mp.FileId, vf.CloudId)
}
