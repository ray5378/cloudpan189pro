package media

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	casrecordSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrecord"
)

type restoreListRequest = casrecordSvi.ListRequest

type restoreListResponse struct {
	Total       int64                    `json:"total"`
	CurrentPage int                      `json:"currentPage"`
	PageSize    int                      `json:"pageSize"`
	Data        []*models.CasMediaRecord `json:"data"`
}

// RestoreList 查询 CAS 恢复记录列表。
// @Summary 查询CAS恢复记录列表
// @Description 按恢复状态、存储、挂载点、文件名、时间范围分页查询恢复记录
// @Tags 媒体操作
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param storageId query int false "存储ID"
// @Param mountPointId query int false "挂载点ID"
// @Param restoreStatus query string false "恢复状态(pending/restoring/restored/failed/recycling/recycled)"
// @Param casFileName query string false "CAS文件名模糊搜索"
// @Param beginAt query string false "开始时间(ISO8601)"
// @Param endAt query string false "结束时间(ISO8601)"
// @Param currentPage query int false "当前页码，默认为1" default(1)
// @Param pageSize query int false "每页大小，默认为10" default(10)
// @Success 200 {object} httpcontext.Response{data=restoreListResponse} "查询成功"
// @Failure 400 {object} httpcontext.Response "参数验证失败，code=99998"
// @Failure 400 {object} httpcontext.Response "查询失败"
// @Failure 401 {object} httpcontext.Response "未授权访问"
// @Failure 403 {object} httpcontext.Response "权限不足"
// @Router /api/media/restore_list [get]
func (h *handler) RestoreList() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		req := new(restoreListRequest)
		if err := ctx.ShouldBindQuery(req); err != nil {
			ctx.AbortWithInvalidParams(err)
			return
		}

		list, err := h.casRecordService.List(ctx.GetContext(), req)
		if err != nil {
			ctx.Fail(codeRestoreListFailed.WithError(err))
			return
		}
		total, err := h.casRecordService.Count(ctx.GetContext(), req)
		if err != nil {
			ctx.Fail(codeRestoreListFailed.WithError(err))
			return
		}

		ctx.Success(&restoreListResponse{
			Total:       total,
			CurrentPage: req.CurrentPage,
			PageSize:    req.PageSize,
			Data:        list,
		})
	}
}
