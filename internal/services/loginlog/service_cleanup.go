package loginlog

import (
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
)

// CleanupOlderThan 删除早于 before 的登录日志，返回删除条数
func (s *service) CleanupOlderThan(ctx context.Context, before time.Time) (int64, error) {
	res := s.getDB(ctx).Where("created_at < ?", before).Delete(nil)
	return res.RowsAffected, res.Error
}
