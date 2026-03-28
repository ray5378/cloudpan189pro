package mountpoint

import (
	"errors"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (s *service) Query(ctx context.Context, fileId int64) (*models.MountPoint, error) {
	var mountPoint models.MountPoint

	if err := s.getDB(ctx).Where("file_id = ?", fileId).First(&mountPoint).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		ctx.Error("查询挂载点失败", zap.Error(err), zap.Int64("fileId", fileId))

		return nil, err
	}

	return &mountPoint, nil
}

func (s *service) QueryByPath(ctx context.Context, fullPath string) (*models.MountPoint, error) {
	var mountPoint models.MountPoint

	if err := s.getDB(ctx).Where("full_path = ?", fullPath).First(&mountPoint).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		ctx.Error("根据路径查询挂载点失败", zap.Error(err), zap.String("fullPath", fullPath))

		return nil, err
	}

	return &mountPoint, nil
}
