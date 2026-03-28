package storage

import (
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"gorm.io/gorm"
)

type batchModifyTokenRequest struct {
	IDs     []int64 `json:"ids" binding:"required,min=1"`
	TokenID int64   `json:"tokenId"` // 0 表示解绑
}

type batchModifyTokenResult struct {
	SuccessCount int     `json:"successCount"`
	FailCount    int     `json:"failCount"`
	FailIDs      []int64 `json:"failIds"`
}

// BatchModifyToken 批量更换令牌
// @Summary 批量更换令牌
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body batchModifyTokenRequest true "批量更换令牌参数"
// @Success 200 {object} httpcontext.Response{data=batchModifyTokenResult}
// @Router /api/storage/batch_modify_token [post]
func (h *handler) BatchModifyToken() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		req := new(batchModifyTokenRequest)
		if err := ctx.ShouldBindJSON(req); err != nil {
			ctx.AbortWithInvalidParams(err)
			return
		}

		// 令牌存在性校验（非 0 时）
		if req.TokenID != 0 {
			if _, err := h.cloudTokenService.Query(ctx.GetContext(), req.TokenID); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					ctx.Fail(busCodeStorageCloudTokenNotExist.WithError(err))
				} else {
					ctx.Fail(busCodeStorageQueryCloudTokenError.WithError(err))
				}
				return
			}
		}

		res := &batchModifyTokenResult{}
		sem := make(chan struct{}, 10)
		done := make(chan struct{})
		var (
			successCount atomic.Int64
			failCount    atomic.Int64
			failIDsMu    sync.Mutex
		)

		for _, id := range req.IDs {
			id := id
			sem <- struct{}{}
			go func() {
				defer func() { <-sem; done <- struct{}{} }()
				// 校验挂载点是否存在
				if _, err := h.mountPointService.Query(ctx.GetContext(), id); err != nil {
					failCount.Add(1)
					failIDsMu.Lock()
					res.FailIDs = append(res.FailIDs, id)
					failIDsMu.Unlock()
					return
				}
				if err := h.mountPointService.ModifyToken(ctx.GetContext(), id, req.TokenID); err != nil {
					failCount.Add(1)
					failIDsMu.Lock()
					res.FailIDs = append(res.FailIDs, id)
					failIDsMu.Unlock()
					return
				}
				successCount.Add(1)
			}()
		}

		for range req.IDs {
			<-done
		}
		res.SuccessCount = int(successCount.Load())
		res.FailCount = int(failCount.Load())
		ctx.Success(res)
	}
}
