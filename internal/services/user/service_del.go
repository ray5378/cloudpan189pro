package user

import (
	"errors"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"

	"go.uber.org/zap"
)

type DelRequest struct {
	ID int64 `json:"id" binding:"required,min=1" example:"1001"` // 用户ID，必须大于1
}

func (s *service) Del(ctx context.Context, req *DelRequest) error {
	if req.ID == 1 {
		return errors.New("创始人不能删除")
	}

	result := s.getDB(ctx).Where("id = ?", req.ID).Delete(&models.User{})
	if result.Error != nil {
		ctx.Error("用户删除失败", zap.Error(result.Error), zap.Int64("user_id", req.ID))

		return result.Error
	}

	return nil
}
