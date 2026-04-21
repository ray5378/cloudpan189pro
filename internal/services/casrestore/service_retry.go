package casrestore

import (
	"fmt"
	"strings"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	virtualfileSvi "github.com/xxcheng123/cloudpan189-share/internal/services/virtualfile"
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

	vf, err := s.locateVirtualFileForRetry(ctx, record)
	if err != nil {
		return nil, err
	}
	if vf == nil {
		return nil, fmt.Errorf("无法根据恢复记录重新定位CAS虚拟文件")
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
		CasFileName:     chooseNonEmpty(record.CasFileName, vf.Name),
		CasVirtualID:    vf.ID,
		UploadRoute:     req.UploadRoute,
		DestinationType: req.DestinationType,
		TargetFolderID:  targetFolderID,
	})
}

func (s *service) locateVirtualFileForRetry(ctx appctx.Context, record *models.CasMediaRecord) (*models.VirtualFile, error) {
	if record == nil {
		return nil, fmt.Errorf("恢复记录不能为空")
	}

	// 1. 最优先使用持久化的 casFilePath；这是最准的路径定位。
	if record.CasFilePath != "" {
		vf, err := s.virtualFileService.QueryByPath(ctx, record.CasFilePath)
		if err == nil && vf != nil {
			return vf, nil
		}
	}

	// 2. 旧记录兼容：在同一 mount point(top_id) 下，优先按 cloud_id 精确匹配。
	topID := record.StorageID
	if topID == 0 {
		topID = record.MountPointID
	}
	if topID > 0 && record.CasFileID != "" {
		list, err := s.virtualFileService.List(ctx, &virtualfileSvi.ListRequest{
			TopId:    &topID,
			PageSize: 500,
			DescList: []string{"id"},
		})
		if err != nil {
			return nil, err
		}
		for _, item := range list {
			if item == nil || item.IsDir {
				continue
			}
			if item.CloudId == record.CasFileID {
				return item, nil
			}
		}
	}

	// 3. 再退一步：按名字缩窄，但要求是 .cas 文件，避免把同名真实媒体误判进去。
	if topID > 0 && record.CasFileName != "" {
		list, err := s.virtualFileService.List(ctx, &virtualfileSvi.ListRequest{
			TopId:    &topID,
			Name:     record.CasFileName,
			PageSize: 200,
			DescList: []string{"id"},
		})
		if err != nil {
			return nil, err
		}
		for _, item := range list {
			if item == nil || item.IsDir {
				continue
			}
			if strings.EqualFold(item.Name, record.CasFileName) && strings.HasSuffix(strings.ToLower(item.Name), ".cas") {
				return item, nil
			}
		}
	}

	return nil, fmt.Errorf("恢复记录缺少可复用定位信息：casFilePath=%q casFileID=%q casFileName=%q storageID=%d mountPointID=%d",
		record.CasFilePath, record.CasFileID, record.CasFileName, record.StorageID, record.MountPointID)
}

func chooseNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
