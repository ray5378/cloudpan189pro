package casrecord

import (
	"strings"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
)

func (s *service) QueryByRestoredFileID(ctx context.Context, restoredFileID string) (*models.CasMediaRecord, error) {
	restoredFileID = strings.TrimSpace(restoredFileID)
	if restoredFileID == "" {
		return nil, nil
	}
	record := new(models.CasMediaRecord)
	err := s.getDB(ctx).Where("restored_file_id = ?", restoredFileID).Order("id desc").First(record).Error
	if err != nil {
		return nil, err
	}
	return record, nil
}
