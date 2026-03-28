package setting

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"go.uber.org/zap"
)

func (s *service) Update(ctx context.Context, fields ...utils.Field) error {
	setting, err := s.Query(ctx)
	if err != nil {
		return err
	}

	mp := make(map[string]interface{})

	for _, field := range fields {
		mp[field.Key] = field.Value
	}

	result := s.getDB(ctx).Where("id = ?", setting.ID).Updates(mp)
	if result.Error != nil {
		ctx.Error("更新信息失败", zap.Error(result.Error), zap.Int64("id", setting.ID))

		return result.Error
	}

	return nil
}
