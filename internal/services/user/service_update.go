package user

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
)

func (s *service) Update(ctx context.Context, uid int64, fields ...utils.Field) error {
	mp := make(map[string]interface{})

	for _, field := range fields {
		mp[field.Key] = field.Value
	}

	result := s.getDB(ctx).Model(new(models.User)).Where("id = ?", uid).Updates(mp)
	if result.Error != nil {
		ctx.Error("更新用户信息失败", zap.Error(result.Error), zap.Int64("user_id", uid))

		return result.Error
	}

	return nil
}
