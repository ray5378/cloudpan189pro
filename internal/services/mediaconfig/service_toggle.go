package mediaconfig

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
)

// Toggle 切换启用状态
func (s *service) Toggle(ctx context.Context, enable bool) error {
	return s.Update(ctx, utils.WithField("enable", enable))
}
