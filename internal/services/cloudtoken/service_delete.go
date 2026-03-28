package cloudtoken

import (
	"github.com/pkg/errors"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"go.uber.org/zap"
)

// DeleteRequest 删除云盘令牌请求
type DeleteRequest struct {
	ID int64 `json:"id" binding:"required" example:"1"` // 云盘令牌ID
}

func (s *service) Delete(ctx context.Context, req *DeleteRequest) (err error) {
	result := s.getDB(ctx).Where("id = ?", req.ID).Delete(nil)
	if result.Error != nil {
		ctx.Error("删除云盘令牌失败", zap.Error(result.Error), zap.Int64("id", req.ID))

		return errors.Wrap(result.Error, "删除云盘令牌失败")
	}

	return nil
}
