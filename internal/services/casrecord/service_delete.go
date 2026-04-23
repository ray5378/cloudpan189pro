package casrecord

import "github.com/xxcheng123/cloudpan189-share/internal/framework/context"

func (s *service) DeleteByCasFilePath(ctx context.Context, casFilePath string) error {
	return s.getDB(ctx).Where("cas_file_path = ?", casFilePath).Delete(nil).Error
}
