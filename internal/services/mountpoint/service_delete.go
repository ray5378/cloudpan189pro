package mountpoint

import (
	"github.com/samber/lo"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
)

func (s *service) Delete(ctx context.Context, fileId int64) error {
	if err := s.getDB(ctx).Unscoped().Where("file_id = ?", fileId).Delete(&models.MountPoint{}).Error; err != nil {
		ctx.Error("删除挂载点失败", zap.Error(err), zap.Int64("fileId", fileId))
		return err
	}
	return nil
}

func (s *service) BatchDelete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	sampleIDs := lo.Slice(ids, 0, lo.Ternary(len(ids) > 10, 10, len(ids)))
	db := s.getDB(ctx).Unscoped().Where("file_id IN ?", ids).Delete(&models.MountPoint{})

	if db.Error != nil {
		ctx.Error("批量删除挂载点失败",
			zap.Error(db.Error),
			zap.Int("req_count", len(ids)),
			zap.Int64s("req_sample_ids", sampleIDs))
		return db.Error
	}

	ctx.Info("批量删除挂载点执行完成",
		zap.Int("req_count", len(ids)),
		zap.Int64s("req_sample_ids", sampleIDs),
		zap.Int64("deleted_rows", db.RowsAffected))

	return nil
}
