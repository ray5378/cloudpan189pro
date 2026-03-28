package setting

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
)

func (s *service) Query(ctx context.Context) (*models.Setting, error) {
	setting := new(models.Setting)

	if err := s.getDB(ctx).First(setting).Error; err != nil {
		ctx.Error("设置查询失败", zap.Error(err))

		return nil, err
	}

	return setting, nil
}
