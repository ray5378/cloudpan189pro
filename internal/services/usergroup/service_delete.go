package usergroup

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"

	"go.uber.org/zap"
)

type DeleteRequest struct {
	ID int64 `json:"id" binding:"required,min=1" example:"1001"` // 用户组ID，必须大于1
}

func (s *service) Delete(ctx context.Context, req *DeleteRequest) error {
	result := s.getDB(ctx).Where("id = ?", req.ID).Delete(&models.UserGroup{})
	if result.Error != nil {
		ctx.Error("用户组删除失败", zap.Int64("id", req.ID), zap.Error(result.Error))

		return result.Error
	}

	return nil
}
