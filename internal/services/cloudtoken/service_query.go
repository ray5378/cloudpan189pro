package cloudtoken

import (
	"github.com/pkg/errors"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (s *service) Query(ctx context.Context, id int64) (*models.CloudToken, error) {
	var cloudToken models.CloudToken
	if err := s.getDB(ctx).Where("id = ?", id).First(&cloudToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("云盘令牌不存在")
		}

		ctx.Error("查询云盘令牌失败", zap.Error(err), zap.Int64("id", id))

		return nil, errors.Wrap(err, "查询云盘令牌失败")
	}

	return &cloudToken, nil
}
