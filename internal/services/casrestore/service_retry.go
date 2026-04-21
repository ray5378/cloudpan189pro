package casrestore

import (
	"fmt"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
)

func (s *service) RetryRecord(ctx appctx.Context, req RetryRequest) (*RestoreResult, error) {
	if req.RecordID <= 0 {
		return nil, fmt.Errorf("recordID不能为空")
	}
	if req.UploadRoute == "" {
		req.UploadRoute = UploadRouteFamily
	}
	if req.DestinationType == "" {
		return nil, fmt.Errorf("destinationType不能为空")
	}

	record, err := s.recordSvc.Query(ctx, req.RecordID)
	if err != nil {
		return nil, err
	}
	if record.CasFilePath == "" {
		return nil, fmt.Errorf("恢复记录缺少casFilePath，暂时无法按record重试")
	}

	vf, err := s.virtualFileService.QueryByPath(ctx, record.CasFilePath)
	if err != nil {
		return nil, err
	}
	if vf == nil {
		return nil, fmt.Errorf("根据记录中的casFilePath无法定位虚拟文件")
	}

	targetFolderID := req.TargetFolderID
	if targetFolderID == "" {
		targetFolderID = record.RestoredParentID
	}
	if targetFolderID == "" {
		return nil, fmt.Errorf("恢复记录缺少目标目录，且本次未显式传targetFolderID")
	}

	return s.EnsureRestored(ctx, RestoreRequest{
		StorageID:       record.StorageID,
		MountPointID:    record.MountPointID,
		CasFileID:       record.CasFileID,
		CasFileName:     record.CasFileName,
		CasVirtualID:    vf.ID,
		UploadRoute:     req.UploadRoute,
		DestinationType: req.DestinationType,
		TargetFolderID:  targetFolderID,
	})
}
