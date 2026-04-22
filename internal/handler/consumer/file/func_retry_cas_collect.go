package file

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/taskcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/types/topic"
	"go.uber.org/zap"
)

const (
	// casCollectRetryDelay: 订阅 .cas 自动转存的 SHARE_SAVE 任务失败后，不立即连打重试；
	// 而是等待 5 分钟再重新发起一次新的 SHARE_SAVE 任务，给云盘侧任务态留恢复时间。
	casCollectRetryDelay = 5 * time.Minute
	// casCollectMaxRetryCount: 当前只允许 1 次延迟重试，避免无限重试放大错误。
	casCollectMaxRetryCount = 1
)

func (h *handler) scheduleRetryCASCollect(ctx context.Context, file *models.VirtualFile, retryCount int, reason error) {
	if h.taskEngine == nil || file == nil || file.ID <= 0 {
		return
	}
	if retryCount >= casCollectMaxRetryCount {
		ctx.Warn("CAS自动归集已达到最大延迟重试次数，不再重试",
			zap.Int64("fileId", file.ID),
			zap.String("fileName", file.Name),
			zap.Int("retryCount", retryCount),
			zap.Error(reason),
		)
		return
	}

	req := topic.FileRetryCasCollectRequest{FileId: file.ID, RetryCount: retryCount + 1}
	body, err := json.Marshal(req)
	if err != nil {
		ctx.Warn("CAS自动归集延迟重试任务序列化失败", zap.Error(err), zap.Int64("fileId", file.ID))
		return
	}

	go func(traceID string) {
		time.Sleep(casCollectRetryDelay)
		retryCtx := context.NewContext(ctx, context.WithLogger(h.logger), context.WithTraceId(traceID)).WithValue("cas_retry", true)
		if pushErr := h.taskEngine.PushMessage(retryCtx, req.Topic(), body); pushErr != nil {
			h.logger.Error("CAS自动归集延迟重试任务投递失败",
				zap.Error(pushErr),
				zap.Int64("fileId", file.ID),
				zap.String("fileName", file.Name),
				zap.Int("retryCount", req.RetryCount),
			)
		}
	}(ctx.Trace.ID())

	ctx.Warn("CAS自动归集已安排延迟重试",
		zap.Int64("fileId", file.ID),
		zap.String("fileName", file.Name),
		zap.Duration("delay", casCollectRetryDelay),
		zap.Int("nextRetryCount", req.RetryCount),
		zap.Error(reason),
	)
}

func (h *handler) RetryCasCollect() taskcontext.HandlerFunc {
	return func(taskCtx *taskcontext.Context) error {
		var req topic.FileRetryCasCollectRequest
		if err := taskCtx.Unmarshal(&req); err != nil {
			return fmt.Errorf("解析CAS延迟重试任务失败: %w", err)
		}
		ctx := taskCtx.GetContext()
		file, err := h.virtualFileService.Query(ctx, req.FileId)
		if err != nil {
			return fmt.Errorf("查询CAS延迟重试文件失败: %w", err)
		}
		ctx.Info("开始执行CAS自动归集延迟重试",
			zap.Int64("fileId", file.ID),
			zap.String("fileName", file.Name),
			zap.Int("retryCount", req.RetryCount),
		)
		if err := h.tryCollectCASFromVirtualFileWithRetry(ctx, file, req.RetryCount); err != nil {
			return err
		}
		ctx.Info("CAS自动归集延迟重试成功",
			zap.Int64("fileId", file.ID),
			zap.String("fileName", file.Name),
			zap.Int("retryCount", req.RetryCount),
		)
		return nil
	}
}
