package casrecord

import "github.com/xxcheng123/cloudpan189-share/internal/framework/context"

func (s *service) Update(ctx context.Context, id int64, updates map[string]any) error {
	return s.getDB(ctx).Where("id = ?", id).Updates(updates).Error
}
