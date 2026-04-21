package casrestore

import (
	"fmt"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
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
	if req.CasVirtualID <= 0 {
		return nil, fmt.Errorf("casVirtualID不能为空")
	}
	if req.MountPointID <= 0 {
		return nil, fmt.Errorf("mountPointID不能为空")
	}
	if req.TargetFolderID == "" {
		return nil, fmt.Errorf("targetFolderID不能为空")
	}

	ctx.Logger.Info("CAS恢复开始",
		zap.Int64("storage_id", req.StorageID),
		zap.Int64("mount_point_id", req.MountPointID),
		zap.Int64("cas_virtual_id", req.CasVirtualID),
		zap.String("cas_file_id", req.CasFileID),
		zap.String("target_folder_id", req.TargetFolderID),
	)

	resolver := s.newCASMetadataResolver()
	casInfo, vf, err := resolver.Resolve(ctx, req.MountPointID, req.CasVirtualID)
	if err != nil {
		return nil, err
	}

	session, err := s.appSessionService.GetByMountPointID(ctx, req.MountPointID)
	if err != nil {
		return nil, err
	}
	panClient := buildPanClient(session)
	if panClient == nil {
		return nil, fmt.Errorf("创建PanClient失败")
	}

	restoreName := casparser.GetOriginalFileName(vf.Name, casInfo)
	personResult, err := (&personRestoreAdapter{}).TryRestore(panClient, req.TargetFolderID, restoreName, casInfo)
	if err != nil {
		return nil, err
	}

	return &RestoreResult{
		RestoredFileID:   personResult.RestoredFileID,
		RestoredFileName: personResult.RestoredFileName,
		TargetFolderID:   req.TargetFolderID,
		CasInfo:          casInfo,
	}, nil
}
