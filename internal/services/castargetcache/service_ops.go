package castargetcache

import (
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *service) Exists(ctx context.Context, targetTokenID int64, targetFolderID, fileName string) (bool, error) {
	var count int64
	err := s.getDB(ctx).
		Where("target_token_id = ?", targetTokenID).
		Where("target_folder_id = ?", targetFolderID).
		Where("file_name = ?", fileName).
		Count(&count).Error
	return count > 0, err
}

func (s *service) Upsert(ctx context.Context, item *models.CasTargetDirCache) error {
	return s.getDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "target_token_id"}, {Name: "target_folder_id"}, {Name: "file_name"}},
		DoUpdates: clause.Assignments(map[string]any{
			"is_dir":       item.IsDir,
			"refreshed_at": item.RefreshedAt,
			"updated_at":   time.Now(),
		}),
	}).Create(item).Error
}

func (s *service) RefreshDir(ctx context.Context, targetTokenID int64, targetFolderID string, items []*models.CasTargetDirCache) error {
	return s.getDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("target_token_id = ?", targetTokenID).Where("target_folder_id = ?", targetFolderID).Delete(new(models.CasTargetDirCache)).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return tx.Create(items).Error
	})
}

func (s *service) NeedsRefresh(ctx context.Context, targetTokenID int64, targetFolderID string, ttl time.Duration) (bool, error) {
	var item models.CasTargetDirCache
	err := s.getDB(ctx).
		Where("target_token_id = ?", targetTokenID).
		Where("target_folder_id = ?", targetFolderID).
		Order("refreshed_at desc").
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, nil
		}
		return false, err
	}
	return time.Since(item.RefreshedAt) >= ttl, nil
}

func (s *service) IsEmpty(ctx context.Context) (bool, error) {
	var count int64
	if err := s.getDB(ctx).Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

func (s *service) ListDistinctDirs(ctx context.Context) ([]*models.CasTargetDirCache, error) {
	list := make([]*models.CasTargetDirCache, 0)
	err := s.getDB(ctx).
		Select("target_token_id, target_folder_id, max(refreshed_at) as refreshed_at").
		Group("target_token_id, target_folder_id").
		Find(&list).Error
	return list, err
}

func (s *service) ClearAll(ctx context.Context) (int64, error) {
	tx := s.getDB(ctx).Where("1 = 1").Delete(new(models.CasTargetDirCache))
	return tx.RowsAffected, tx.Error
}
