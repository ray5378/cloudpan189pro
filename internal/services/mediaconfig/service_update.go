package mediaconfig

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"go.uber.org/zap"
)

func (s *service) Update(ctx context.Context, fields ...utils.Field) error {
	cfg, err := s.Query(ctx)
	if err != nil {
		return err
	}

	mp := make(map[string]interface{})
	for _, field := range fields {
		mp[field.Key] = field.Value
	}

	result := s.getDB(ctx).Where("id = ?", cfg.ID).Updates(mp)
	if result.Error != nil {
		ctx.Error("媒体配置更新失败", zap.Error(result.Error), zap.Int64("id", cfg.ID))

		return result.Error
	}

	defer func() {
		cfg := new(models.MediaConfig)
		// 回查写入
		if err = s.getDB(ctx).First(cfg).Error; err != nil {
			ctx.Error("媒体配置回查写入失败", zap.Error(err), zap.Int64("id", cfg.ID))
		} else {
			shared.MediaConfig = cfg
		}
	}()

	return nil
}
