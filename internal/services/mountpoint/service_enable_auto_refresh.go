package mountpoint

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"go.uber.org/zap"
)

func (s *service) EnableAutoRefresh(ctx context.Context, fileId int64, enable bool) error {
	if err := s.getDB(ctx).Where("file_id = ?", fileId).Update("enable_auto_refresh", enable).Error; err != nil {
		ctx.Error("更新挂载点自动刷新状态失败", zap.Error(err), zap.Int64("fileId", fileId), zap.Bool("enable", enable))

		return err
	}

	return nil
}
