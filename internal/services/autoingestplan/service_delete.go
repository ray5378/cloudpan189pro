package autoingestplan

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
)

// Delete 删除自动挂载计划
func (s *service) Delete(ctx context.Context, id int64) error {
	if err := s.getDB(ctx).Where("id = ?", id).Delete(new(models.AutoIngestPlan)).Error; err != nil {
		ctx.Error("删除自动挂载计划失败", zap.Error(err), zap.Int64("id", id))

		return err
	}

	return nil
}
