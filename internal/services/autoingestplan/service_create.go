package autoingestplan

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
)

// Create 创建自动挂载计划
func (s *service) Create(ctx context.Context, plan *models.AutoIngestPlan) (int64, error) {
	if err := s.getDB(ctx).Create(plan).Error; err != nil {
		ctx.Error("创建自动挂载计划失败", zap.Error(err), zap.String("name", plan.Name))

		return 0, err
	}

	return plan.ID, nil
}
