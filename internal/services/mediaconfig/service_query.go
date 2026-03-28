package mediaconfig

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
)

func (s *service) Query(ctx context.Context) (*models.MediaConfig, error) {
	cfg := new(models.MediaConfig)

	if err := s.getDB(ctx).First(cfg).Error; err != nil {
		ctx.Error("媒体配置查询失败", zap.Error(err))

		return nil, err
	}

	return cfg, nil
}
