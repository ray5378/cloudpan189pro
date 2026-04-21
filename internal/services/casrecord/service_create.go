package casrecord

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
)

func (s *service) Create(ctx context.Context, record *models.CasMediaRecord) (int64, error) {
	if err := s.getDB(ctx).Create(record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}
