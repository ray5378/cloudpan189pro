package cloudtoken

import (
	"github.com/pkg/errors"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"go.uber.org/zap"
)

// ModifyNameRequest 修改云盘令牌名称请求
type ModifyNameRequest struct {
	ID   int64  `json:"id" binding:"required" example:"1"`     // 云盘令牌ID
	Name string `json:"name" binding:"required" example:"新名称"` // 新名称
}

func (s *service) ModifyName(ctx context.Context, req *ModifyNameRequest) (err error) {
	if err = s.getDB(ctx).Where("id = ?", req.ID).Update("name", req.Name).Error; err != nil {
		ctx.Error("修改云盘令牌名称失败", zap.Error(err), zap.Int64("id", req.ID), zap.String("name", req.Name))

		return errors.Wrap(err, "修改云盘令牌名称失败")
	}

	return nil
}
