package storage

import (
	"github.com/pkg/errors"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"gorm.io/gorm"
)

type modifyTokenRequest struct {
	ID      int64 `json:"id" binding:"required" example:"1001"` // 挂载点ID
	TokenID int64 `json:"tokenId" example:"123"`                // 新的令牌ID
}

// ModifyToken 修改存储挂载点令牌
// @Summary 修改存储挂载点令牌
// @Description 修改指定存储挂载点关联的云盘令牌
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body modifyTokenRequest true "修改令牌请求参数"
// @Success 200 {object} httpcontext.Response "令牌修改成功"
// @Failure 400 {object} httpcontext.Response "参数验证失败，code=99998"
// @Failure 400 {object} httpcontext.Response "挂载点不存在，code=4022"
// @Failure 400 {object} httpcontext.Response "查询挂载点失败，code=4021"
// @Failure 400 {object} httpcontext.Response "云盘令牌不存在，code=4013"
// @Failure 400 {object} httpcontext.Response "查询云盘令牌失败，code=4019"
// @Failure 400 {object} httpcontext.Response "修改令牌失败，code=4030"
// @Failure 401 {object} httpcontext.Response "未授权访问"
// @Failure 403 {object} httpcontext.Response "权限不足"
// @Router /api/storage/modify_token [post]
func (h *handler) ModifyToken() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		req := new(modifyTokenRequest)
		if err := ctx.ShouldBindJSON(req); err != nil {
			ctx.AbortWithInvalidParams(err)

			return
		}

		// 验证挂载点是否存在
		if _, err := h.mountPointService.Query(ctx.GetContext(), req.ID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				ctx.Fail(busCodeStorageMountPointNotFound.WithError(err))
			} else {
				ctx.Fail(busCodeStorageQueryMountPointError.WithError(err))
			}

			return
		}

		// 验证云盘令牌是否存在
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

		// 修改挂载点的令牌
		if err := h.mountPointService.ModifyToken(ctx.GetContext(), req.ID, req.TokenID); err != nil {
			ctx.Fail(busCodeStorageModifyTokenFailed.WithError(err))

			return
		}

		ctx.Success()
	}
}
