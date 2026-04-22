package file

import (
	"fmt"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
)

func (h *handler) downloadCASFileToLocal(ctx context.Context, file *models.VirtualFile) error {
	if h.localCASService == nil {
		return fmt.Errorf("localCASService未初始化")
	}
	_, err := h.localCASService.DownloadToLocal(ctx, file)
	return err
}
