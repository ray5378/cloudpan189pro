package autoingestplan

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
)

// Enable 启用自动挂载计划
func (s *service) Enable(ctx context.Context, id int64) error {
	return s.Update(ctx, id, utils.Field{Key: "enabled", Value: true})
}

// Disable 停用自动挂载计划
func (s *service) Disable(ctx context.Context, id int64) error {
	return s.Update(ctx, id, utils.Field{Key: "enabled", Value: false})
}
