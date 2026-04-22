package casrestore

import (
	"fmt"
	"strings"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
	"go.uber.org/zap"
)

func (s *service) EnsureRestored(ctx appctx.Context, req RestoreRequest) (*RestoreResult, error) {
	key := inflightKey(req)
	return s.withInflight(ctx, key, func() (*RestoreResult, error) {
		return s.ensureRestoredOnce(ctx, req)
	})
}

func (s *service) EnsureRestoredFromLocalCAS(ctx appctx.Context, req RestoreRequest) (*RestoreResult, error) {
	if strings.TrimSpace(req.LocalCasPath) == "" {
		return nil, fmt.Errorf("localCasPath不能为空")
	}
	key := inflightKey(req)
	return s.withInflight(ctx, key, func() (*RestoreResult, error) {
		return s.ensureRestoredOnce(ctx, req)
	})
}

func buildFamilyTransferTempName(info *casparser.CasInfo) string {
	if info == nil {
		return "0cas.transfer"
	}
	md5 := strings.ToLower(strings.TrimSpace(info.MD5))
	if md5 == "" {
		md5 = "cas"
	}
	return "0" + md5 + ".transfer"
}

func (s *service) ensureRestoredOnce(ctx appctx.Context, req RestoreRequest) (result *RestoreResult, err error) {
	isLocal := strings.TrimSpace(req.LocalCasPath) != ""

	if !isLocal {
		if req.CasFileID == "" {
			return nil, fmt.Errorf("casFileID不能为空")
		}
		if req.CasVirtualID <= 0 {
			return nil, fmt.Errorf("casVirtualID不能为空")
		}
		if req.MountPointID <= 0 {
			return nil, fmt.Errorf("mountPointID不能为空")
		}
	} else {
		if req.TargetTokenID <= 0 {
			return nil, fmt.Errorf("targetTokenID不能为空")
		}
		if req.CasFileID == "" {
			req.CasFileID = req.LocalCasPath
		}
		if req.CasFileName == "" {
			_, name, localErr := resolveLocalCAS(req.LocalCasPath)
			if localErr == nil {
				req.CasFileName = name
			}
		}
	}

	if req.UploadRoute == "" {
		req.UploadRoute = UploadRouteFamily
	}
	if req.DestinationType == "" {
		return nil, fmt.Errorf("destinationType不能为空")
	}
	if req.TargetFolderID == "" {
		return nil, fmt.Errorf("targetFolderID不能为空")
	}

	ctx.Logger.Info("CAS恢复开始(route + destination separated)",
		zap.Int64("storage_id", req.StorageID),
		zap.Int64("mount_point_id", req.MountPointID),
		zap.Int64("target_token_id", req.TargetTokenID),
		zap.Int64("cas_virtual_id", req.CasVirtualID),
		zap.String("cas_file_id", req.CasFileID),
		zap.String("cas_file_name", req.CasFileName),
		zap.String("local_cas_path", req.LocalCasPath),
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

	var (
		casInfo *casparser.CasInfo
		vf      *models.VirtualFile
	)
	if !isLocal {
		resolver := s.newCASMetadataResolver()
		casInfo, vf, err = resolver.Resolve(ctx, req.MountPointID, req.CasVirtualID)
		if err != nil {
			return nil, err
		}
		if !isRecoverableFallbackTarget(vf) {
			return nil, fmt.Errorf("当前文件类型不支持恢复: %s", vf.OsType)
		}
	} else {
		casInfo, _, err = resolveLocalCAS(req.LocalCasPath)
		if err != nil {
			return nil, err
		}
	}

	restoreName := casparser.GetOriginalFileName(req.CasFileName, casInfo)
	if !isLocal && vf != nil {
		restoreName = casparser.GetOriginalFileName(vf.Name, casInfo)
	}
	if req.UploadRoute == UploadRouteFamily && req.DestinationType == DestinationTypePerson {
		restoreName = buildFamilyTransferTempName(casInfo)
	}

	if err = s.markRestoring(ctx, record.ID, restoreName, casInfo.Size, casInfo.MD5, casInfo.SliceMD5); err != nil {
		return nil, err
	}

	var session *appsession.Session
	if !isLocal {
		session, err = s.appSessionService.GetByMountPointID(ctx, req.MountPointID)
	} else {
		session, err = s.appSessionService.GetByTokenID(ctx, req.TargetTokenID)
	}
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
		FamilyID:        req.FamilyID,
		CasInfo:         casInfo,
	}
	if req.FamilyID > 0 {
		familyIDSessionHint[session] = req.FamilyID
		defer delete(familyIDSessionHint, session)
	}

	switch req.UploadRoute {
	case UploadRoutePerson:
		personResult, personErr := (&personRestoreAdapter{}).TryRestore(session, panClient, req.DestinationType, req.TargetFolderID, restoreName, casInfo)
		if personErr != nil {
			return nil, fmt.Errorf("个人路线恢复失败: %w", personErr)
		}
		result.RestoredFileID = personResult.RestoredFileID
		result.RestoredFileName = personResult.RestoredFileName
		if req.FamilyID > 0 {
			result.FamilyID = req.FamilyID
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

	verifyMountPointID := req.MountPointID
	if isLocal {
		verifyMountPointID = 0
	}

	if req.DestinationType == DestinationTypeFamily {
		fileID, fileName, verifyErr := s.verifyRestoredInFamilyFolderByToken(ctx, verifyMountPointID, req.TargetTokenID, result.FamilyID, req.TargetFolderID, restoreName)
		if verifyErr != nil {
			return nil, verifyErr
		}
		result = normalizeRestoreResult(result, fileID, fileName, req.TargetFolderID)
	} else {
		fileID, fileName, verifyErr := s.verifyRestoredInPersonFolderByToken(ctx, verifyMountPointID, req.TargetTokenID, req.TargetFolderID, restoreName)
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
