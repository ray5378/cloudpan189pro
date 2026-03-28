package storage

import (
	stdctx "context"
	"encoding/json"
	"sync/atomic"

	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	fctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/types/topic"
)

type batchRefreshRequest struct {
	IDs  []int64 `json:"ids" binding:"required,min=1"`
	Deep bool    `json:"deep"`
}

// BatchRefresh 批量刷新（普通/深度）
// @Summary 批量刷新（普通/深度）
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body batchRefreshRequest true "批量刷新参数"
// @Success 200 {object} httpcontext.Response{data=batchResult}
// @Router /api/storage/batch_refresh [post]
func (h *handler) BatchRefresh() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		req := new(batchRefreshRequest)
		if err := ctx.ShouldBindJSON(req); err != nil {
			ctx.AbortWithInvalidParams(err)
			return
		}

		res := &batchResult{}
		sem := make(chan struct{}, 10)
		done := make(chan struct{})
		var (
			successCount atomic.Int64
			failCount    atomic.Int64
		)

		for _, id := range req.IDs {
			sem <- struct{}{}
			go func(id int64) {
				defer func() { <-sem; done <- struct{}{} }()

				// 派生一个与 HTTP 生命周期解耦的后台 ctx（避免前端关闭导致任务中断）
				reqCtx := ctx.GetContext()
				bg := fctx.NewContext(stdctx.Background(), fctx.WithLogger(reqCtx.Logger), fctx.WithTraceId(reqCtx.Trace.ID()))

				// 查询挂载点，确保刷新的是挂载点对应的 file_id，而不是 mount_point.id
				mp, err := h.mountPointService.Query(bg, id)
				if err != nil || mp == nil {
					failCount.Add(1)
					return
				}

				payload := &topic.FileScanFileRequest{FileId: mp.FileId, Deep: req.Deep}
				body, _ := json.Marshal(payload)
				base := bg.WithValue(consts.CtxKeyInvokeHandlerName, "批量刷新").WithValue(consts.CtxKeyFullPath, mp.FullPath)
				if err := h.taskEngine.PushMessage(base, payload.Topic(), body); err != nil {
					failCount.Add(1)
					return
				}

				successCount.Add(1)
			}(id)
		}

		for range req.IDs {
			<-done
		}
		res.SuccessCount = int(successCount.Load())
		res.FailCount = int(failCount.Load())
		ctx.Success(res)
	}
}
