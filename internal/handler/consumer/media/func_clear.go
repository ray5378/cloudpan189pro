package media

import (
	"strings"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/taskcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"go.uber.org/zap"
)

func (h *handler) Clear() taskcontext.HandlerFunc {
	return func(ctx *taskcontext.Context) error {
		logger := ctx.GetContext().Logger
		storagePath := shared.MediaConfig.StoragePath

		// 基本验证
		if strings.TrimSpace(storagePath) == "" {
			logger.Error("媒体存储路径为空，无法执行清理操作")

			return h.mediaFileService.Clear(ctx.GetContext(), storagePath)
		}

		logger.Info("开始清理媒体文件", zap.String("storage_path", storagePath))

		err := h.mediaFileService.Clear(ctx.GetContext(), storagePath)
		if err != nil {
			logger.Error("清理媒体文件失败", zap.Error(err))
		} else {
			logger.Info("清理媒体文件成功")
		}

		return err
	}
}
