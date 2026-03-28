package storage

import (
	stdctx "context"
	"encoding/json"
	"sync"

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

		var payloadPool = sync.Pool{New: func() any { b := make([]byte, 0, 512); return &b }}

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
					return
				}

				// 结构性回收：payload 缓冲复用，使用完立刻置空
				bufp := payloadPool.Get().(*[]byte)
				buf := (*bufp)[:0]
				payload := &topic.FileScanFileRequest{FileId: mp.FileId, Deep: req.Deep}
				b, _ := json.Marshal(payload)
				buf = append(buf, b...)
				base := bg.WithValue(consts.CtxKeyInvokeHandlerName, "批量刷新").WithValue(consts.CtxKeyFullPath, mp.FullPath)
				if err := h.taskEngine.PushMessage(base, payload.Topic(), buf); err != nil {
					// 归还前置空
					buf = buf[:0]
					*bufp = buf
					payloadPool.Put(bufp)
					return
				}
				// 成功后同样归还缓冲并断开对象引用
				buf = buf[:0]
				*bufp = buf
				payloadPool.Put(bufp)
				payload = nil

				res.SuccessCount++
			}(id)
		}

		for range req.IDs {
			<-done
		}
		ctx.Success(res)
	}
}
