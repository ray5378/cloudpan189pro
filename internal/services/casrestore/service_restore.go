package casrestore

import (
	"fmt"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"go.uber.org/zap"
)

func (s *service) EnsureRestored(ctx appctx.Context, req RestoreRequest) (*RestoreResult, error) {
	key := inflightKey(req)
	return s.withInflight(ctx, key, func() (*RestoreResult, error) {
		return s.ensureRestoredOnce(ctx, req)
	})
}

func (s *service) ensureRestoredOnce(ctx appctx.Context, req RestoreRequest) (*RestoreResult, error) {
	if req.CasFileID == "" {
		return nil, fmt.Errorf("casFileID不能为空")
	}
	if req.TargetFolderID == "" {
		return nil, fmt.Errorf("targetFolderID不能为空")
	}

	ctx.Logger.Info("CAS恢复骨架入口",
		zap.Int64("storage_id", req.StorageID),
		zap.String("cas_file_id", req.CasFileID),
		zap.String("target_folder_id", req.TargetFolderID),
	)

	return nil, fmt.Errorf("CAS恢复写链尚未实现")
}
