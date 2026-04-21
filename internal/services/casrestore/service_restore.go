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

func (s *service) ensureRestoredOnce(ctx appctx.Context, req RestoreRequest) (result *RestoreResult, err error) {
	if req.CasFileID == "" {
		return nil, fmt.Errorf("casFileID不能为空")
	}
	if req.CasVirtualID <= 0 {
		return nil, fmt.Errorf("casVirtualID不能为空")
	}
	if req.MountPointID <= 0 {
		return nil, fmt.Errorf("mountPointID不能为空")
	}
	if req.UploadRoute == "" {
		// 默认路线是家庭优先；这是产品默认值，不等于最终目录一定是家庭目录。
		req.UploadRoute = UploadRouteFamily
	}
	if req.DestinationType == "" {
		// 最终目录类型必须显式给出；它和 UploadRoute 是两个独立维度。
		return nil, fmt.Errorf("destinationType不能为空")
	}
	if req.TargetFolderID == "" {
		return nil, fmt.Errorf("targetFolderID不能为空")
	}

	ctx.Logger.Info("CAS恢复开始(route + destination separated)",
		zap.Int64("storage_id", req.StorageID),
		zap.Int64("mount_point_id", req.MountPointID),
		zap.Int64("cas_virtual_id", req.CasVirtualID),
		zap.String("cas_file_id", req.CasFileID),
		zap.String("upload_route", string(req.UploadRoute)),
		zap.String("destination_type", string(req.DestinationType)),
		zap.String("target_folder_id", req.TargetFolderID),
	)

	record, err := s.getOrCreateRecord(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = s.markRestoreFailed(ctx, record.ID, err)
		}
	}()

	resolver := s.newCASMetadataResolver()
	casInfo, vf, err := resolver.Resolve(ctx, req.MountPointID, req.CasVirtualID)
	if err != nil {
		return nil, err
	}
	if !isRecoverableFallbackTarget(vf) {
		return nil, fmt.Errorf("当前文件类型不支持恢复: %s", vf.OsType)
	}

	restoreName := casparser.GetOriginalFileName(vf.Name, casInfo)
	if err = s.markRestoring(ctx, record.ID, restoreName, casInfo.Size, casInfo.MD5, casInfo.SliceMD5); err != nil {
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

	result = &RestoreResult{
		TargetFolderID:  req.TargetFolderID,
		UploadRoute:     req.UploadRoute,
		DestinationType: req.DestinationType,
		CasInfo:         casInfo,
	}

	switch req.UploadRoute {
	case UploadRoutePerson:
		personResult, personErr := (&personRestoreAdapter{}).TryRestore(panClient, req.DestinationType, req.TargetFolderID, restoreName, casInfo)
		if personErr != nil {
			return nil, fmt.Errorf("个人路线恢复失败: %w", personErr)
		}
		result.RestoredFileID = personResult.RestoredFileID
		result.RestoredFileName = personResult.RestoredFileName
		familyID, pickErr := (&familyRestoreAdapter{}).pickFamilyID(panClient)
		if pickErr == nil {
			result.FamilyID = familyID
		}
	case UploadRouteFamily:
		familyResult, familyErr := (&familyRestoreAdapter{}).TryRestore(session, panClient, req.DestinationType, req.TargetFolderID, restoreName, casInfo)
		if familyErr != nil {
			return nil, fmt.Errorf("家庭路线恢复失败: %w", familyErr)
		}
		result.RestoredFileID = familyResult.RestoredFileID
		result.RestoredFileName = familyResult.RestoredFileName
		result.FamilyID = familyResult.FamilyID
	default:
		return nil, fmt.Errorf("不支持的uploadRoute: %s", req.UploadRoute)
	}

	if req.DestinationType == DestinationTypeFamily {
		fileID, fileName, verifyErr := s.verifyRestoredInFamilyFolder(ctx, req.MountPointID, result.FamilyID, req.TargetFolderID, restoreName)
		if verifyErr != nil {
			return nil, verifyErr
		}
		result = normalizeRestoreResult(result, fileID, fileName, req.TargetFolderID)
	} else {
		fileID, fileName, verifyErr := s.verifyRestoredInPersonFolder(ctx, req.MountPointID, req.TargetFolderID, restoreName)
		if verifyErr != nil {
			return nil, verifyErr
		}
		result = normalizeRestoreResult(result, fileID, fileName, req.TargetFolderID)
	}

	if err = s.markRestored(ctx, record.ID, result); err != nil {
		return nil, err
	}
	return result, nil
}
