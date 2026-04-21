package casrecord

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
)

func (s *service) Query(ctx context.Context, id int64) (*models.CasMediaRecord, error) {
	record := new(models.CasMediaRecord)
	if err := s.getDB(ctx).Where("id = ?", id).First(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *service) QueryByStorageAndCasFileID(ctx context.Context, storageID int64, casFileID string) (*models.CasMediaRecord, error) {
	record := new(models.CasMediaRecord)
	if err := s.getDB(ctx).
		Where("storage_id = ?", storageID).
		Where("cas_file_id = ?", casFileID).
		First(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}
