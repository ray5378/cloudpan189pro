package casrestore

import (
	"time"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"gorm.io/gorm"
)

func (s *service) getOrCreateRecord(ctx appctx.Context, req RestoreRequest) (*models.CasMediaRecord, error) {
	record, err := s.recordSvc.QueryByStorageAndCasFileID(ctx, req.StorageID, req.CasFileID)
	if err == nil {
		return record, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	now := time.Now()
	record = &models.CasMediaRecord{
		StorageID:        req.StorageID,
		MountPointID:     req.MountPointID,
		CasFileID:        req.CasFileID,
		CasFileName:      req.CasFileName,
		RestoredParentID: req.TargetFolderID,
		RestoreStatus:    models.CasRestoreStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	id, err := s.recordSvc.Create(ctx, record)
	if err != nil {
		return nil, err
	}
	record.ID = id
	return record, nil
}

func (s *service) markRestoring(ctx appctx.Context, recordID int64, originalName string, originalSize int64, fileMD5, sliceMD5 string) error {
	return s.recordSvc.Update(ctx, recordID, map[string]any{
		"restore_status":     models.CasRestoreStatusRestoring,
		"original_file_name": originalName,
		"original_file_size": originalSize,
		"file_md5":           fileMD5,
		"slice_md5":          sliceMD5,
		"last_error":         "",
	})
}

func (s *service) markRestoreFailed(ctx appctx.Context, recordID int64, err error) error {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return s.recordSvc.Update(ctx, recordID, map[string]any{
		"restore_status": models.CasRestoreStatusFailed,
		"last_error":     msg,
	})
}

func (s *service) markRestored(ctx appctx.Context, recordID int64, result *RestoreResult) error {
	now := time.Now()
	updates := map[string]any{
		"restore_status":      models.CasRestoreStatusRestored,
		"restored_file_id":    result.RestoredFileID,
		"restored_file_name":  result.RestoredFileName,
		"restored_parent_id":  result.TargetFolderID,
		"restored_at":         &now,
		"last_access_at":      &now,
		"last_error":          "",
	}
	if result.CasInfo != nil {
		updates["original_file_name"] = result.CasInfo.Name
		updates["original_file_size"] = result.CasInfo.Size
		updates["file_md5"] = result.CasInfo.MD5
		updates["slice_md5"] = result.CasInfo.SliceMD5
	}
	return s.recordSvc.Update(ctx, recordID, updates)
}
