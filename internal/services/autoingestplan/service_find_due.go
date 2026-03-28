package autoingestplan

import (
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
)

// FindDue 基于“当天零点到当前的总分钟 / 计划间隔 取余为 0”的规则，筛选需要执行的计划
func (s *service) FindDue(ctx context.Context, now time.Time) ([]*models.AutoIngestPlan, error) {
	// 计算当天零点
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	passedMinutes := int(now.Sub(midnight).Minutes())

	if passedMinutes < 0 {
		passedMinutes = 0
	}

	// 先查出启用且间隔>0的计划
	var all []*models.AutoIngestPlan
	if err := s.getDB(ctx).
		Where("enabled = ?", true).
		Where("auto_ingest_interval > 0").
		Find(&all).Error; err != nil {
		ctx.Error("查询启用的自动挂载计划失败", zap.Error(err))

		return nil, err
	}

	// 在应用层根据取余规则过滤
	due := make([]*models.AutoIngestPlan, 0, len(all))

	for _, p := range all {
		interval := int(p.AutoIngestInterval)
		if interval <= 0 {
			continue
		}

		if passedMinutes%interval == 0 {
			due = append(due, p)
		}
	}

	ctx.Debug("FindDue 过滤得到到期计划",
		zap.Int("total", len(all)),
		zap.Int("due", len(due)),
		zap.Int("passedMinutes", passedMinutes))

	return due, nil
}
