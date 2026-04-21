package casrestore

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
)

type personRestoreAdapter struct{}

type personRestoreResult struct {
	RestoredFileID   string
	RestoredFileName string
}

func (a *personRestoreAdapter) TryRestore(
	panClient *cloudpan.PanClient,
	targetFolderID string,
	fileName string,
	info *casparser.CasInfo,
) (*personRestoreResult, error) {
	if panClient == nil {
		return nil, errors.New("PanClient不能为空")
	}
	if info == nil {
		return nil, errors.New("CAS信息不能为空")
	}
	if targetFolderID == "" {
		return nil, errors.New("目标目录不能为空")
	}
	if fileName == "" {
		fileName = info.Name
	}

	createRes, apiErr := panClient.AppCreateUploadFile(&cloudpan.AppCreateUploadFileParam{
		ParentFolderId: targetFolderID,
		FileName:       fileName,
		Size:           info.Size,
		Md5:            info.MD5,
		LastWrite:      time.Now().Format("2006-01-02 15:04:05"),
		LocalPath:      "/tmp/" + fileName,
	})
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "创建上传任务失败")
	}
	if createRes == nil || createRes.UploadFileId == "" {
		return nil, fmt.Errorf("创建上传任务失败: 缺少uploadFileId")
	}

	status, apiErr := panClient.AppGetUploadFileStatus(createRes.UploadFileId)
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "查询上传状态失败")
	}
	if status == nil || status.FileDataExists != 1 {
		return nil, fmt.Errorf("云端不存在可直接命中的文件数据")
	}

	commitURL := createRes.FileCommitUrl
	if commitURL == "" && status.FileCommitUrl != "" {
		commitURL = status.FileCommitUrl
	}
	if commitURL == "" {
		return nil, fmt.Errorf("缺少commit地址")
	}

	commitRes, apiErr := panClient.AppUploadFileCommit(commitURL, createRes.UploadFileId, createRes.XRequestId)
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "提交恢复失败")
	}
	if commitRes == nil || commitRes.Id == "" {
		return nil, fmt.Errorf("恢复提交成功但未返回文件ID")
	}

	return &personRestoreResult{
		RestoredFileID:   commitRes.Id,
		RestoredFileName: commitRes.Name,
	}, nil
}
