package storage

import (
	stdctx "context"
	"time"

	fctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
)

type batchToggleAutoRefreshRequest struct {
	IDs               []int64 `json:"ids" binding:"required,min=1"`
	EnableAutoRefresh bool    `json:"enableAutoRefresh"`
	AutoRefreshDays   int     `json:"autoRefreshDays,omitempty"`
	RefreshInterval   int     `json:"refreshInterval,omitempty"`
	RefreshBeginAt    string  `json:"refreshBeginAt,omitempty"` // yyyy-MM-dd
	EnableDeepRefresh bool    `json:"enableDeepRefresh,omitempty"`
}

type batchResult struct {
	SuccessCount int `json:"successCount"`
	FailCount    int `json:"failCount"`
}

// BatchToggleAutoRefresh 批量开关自动刷新
// @Summary 批量开关自动刷新
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body batchToggleAutoRefreshRequest true "批量自动刷新参数"
// @Success 200 {object} httpcontext.Response{data=batchResult}
// @Router /api/storage/batch_toggle_auto_refresh [post]
func (h *handler) BatchToggleAutoRefresh() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		req := new(batchToggleAutoRefreshRequest)
		if err := ctx.ShouldBindJSON(req); err != nil {
			ctx.AbortWithInvalidParams(err)
			return
		}

		res := &batchResult{}
		sem := make(chan struct{}, 10)
		done := make(chan struct{})

		for _, id := range req.IDs {
			sem <- struct{}{}
			go func(id int64) {
				defer func() { <-sem; done <- struct{}{} }()

				// 从请求 context 剥离出一个独立可存活的后台上下文
				base := ctx.GetContext()
				bg := fctx.NewContext(stdctx.Background(), fctx.WithLogger(base.Logger), fctx.WithTraceId(base.Trace.ID()))

				// 先更新配置（仅当开启时）
				if req.EnableAutoRefresh {
					cfg := mountpoint.RefreshConfig{}
					if req.AutoRefreshDays > 0 { cfg.AutoRefreshDays = &req.AutoRefreshDays }
					if req.RefreshInterval > 0 { cfg.RefreshInterval = &req.RefreshInterval }
					if req.RefreshBeginAt != "" { if t, err := time.Parse(time.DateOnly, req.RefreshBeginAt); err == nil { cfg.AutoRefreshBeginAt = &t } }
					cfg.EnableDeepRefresh = &req.EnableDeepRefresh
					if err := h.mountPointService.UpdateRefreshConfig(bg, id, cfg); err != nil { return }
				}

				if err := h.mountPointService.EnableAutoRefresh(bg, id, req.EnableAutoRefresh); err != nil { return }

				// 结构性回收：尽快断开临时对象引用
				res.SuccessCount++
			}(id)
		}

		// wait all
		for range req.IDs { <-done }
		ctx.Success(res)
	}
}
