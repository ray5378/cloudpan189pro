package casrecord

import (
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
)

func (s *service) ListDueRecycle(ctx context.Context, now time.Time, limit int) ([]*models.CasMediaRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	list := make([]*models.CasMediaRecord, 0)
	staleRecyclingBefore := now.Add(-5 * time.Minute)
	err := s.getDB(ctx).
		Where("restored_file_id <> ''").
		Where("recycle_after_at IS NOT NULL").
		Where("recycle_after_at <= ?", now).
		Where("(restore_status = ? OR (restore_status = ? AND updated_at <= ?))", models.CasRestoreStatusRestored, models.CasRestoreStatusRecycling, staleRecyclingBefore).
		Order("recycle_after_at asc").
		Limit(limit).
		Find(&list).Error
	return list, err
}
