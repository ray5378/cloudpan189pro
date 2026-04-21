package casrestore

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
)

// personRestoreAdapter 负责“个人路线”的秒传恢复。
// 注意：这里的“个人路线”只描述上传/秒传路径，不代表最终目录一定是个人目录。
type personRestoreAdapter struct{}

// personRestoreResult 表示个人路线恢复后的中间结果。
type personRestoreResult struct {
	RestoredFileID   string
	RestoredFileName string
}

func (a *personRestoreAdapter) TryRestore(
	panClient *cloudpan.PanClient,
	destinationType DestinationType,
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
	if fileName == "" {
		fileName = info.Name
	}

	// 个人路线下，个人目录可直接作为上传父目录；若最终目标是家庭目录，则先落个人，再转存到家庭。
	personParentID := ""
	if destinationType == DestinationTypePerson {
		personParentID = targetFolderID
	}

	createRes, apiErr := panClient.AppCreateUploadFile(&cloudpan.AppCreateUploadFileParam{
		ParentFolderId: personParentID,
		FileName:       fileName,
		Size:           info.Size,
		Md5:            info.MD5,
		LastWrite:      time.Now().Format("2006-01-02 15:04:05"),
		LocalPath:      "/tmp/" + fileName,
	})
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "创建个人上传任务失败")
	}
	if createRes == nil || createRes.UploadFileId == "" {
		return nil, fmt.Errorf("创建个人上传任务失败: 缺少uploadFileId")
	}

	status, apiErr := panClient.AppGetUploadFileStatus(createRes.UploadFileId)
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "查询个人上传状态失败")
	}
	if status == nil || status.FileDataExists != 1 {
		return nil, fmt.Errorf("个人云端不存在可直接命中的文件数据")
	}

	commitURL := createRes.FileCommitUrl
	if commitURL == "" && status.FileCommitUrl != "" {
		commitURL = status.FileCommitUrl
	}
	if commitURL == "" {
		return nil, fmt.Errorf("缺少个人commit地址")
	}

	commitRes, apiErr := panClient.AppUploadFileCommit(commitURL, createRes.UploadFileId, createRes.XRequestId)
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "提交个人恢复失败")
	}
	if commitRes == nil || commitRes.Id == "" {
		return nil, fmt.Errorf("个人恢复提交成功但未返回文件ID")
	}

	result := &personRestoreResult{
		RestoredFileID:   commitRes.Id,
		RestoredFileName: commitRes.Name,
	}
	if destinationType == DestinationTypePerson {
		return result, nil
	}

	// 最终目标是家庭目录时：先从个人转存到家庭，再在家庭内移动到目标目录。
	familyID, err := (&familyRestoreAdapter{}).pickFamilyID(panClient)
	if err != nil {
		return nil, err
	}
	ok, apiErr := panClient.AppSaveFileToFamilyCloud(familyID, []string{commitRes.Id})
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "个人文件转存到家庭云失败")
	}
	if !ok {
		return nil, fmt.Errorf("个人文件转存到家庭云失败")
	}
	if targetFolderID != "" {
		moved, apiErr := panClient.AppFamilyMoveFile(familyID, commitRes.Id, targetFolderID)
		if apiErr != nil {
			return nil, errors.Wrap(apiErr, "转存家庭云后移动文件失败")
		}
		if moved != nil {
			result.RestoredFileID = moved.FileId
			result.RestoredFileName = moved.FileName
		}
	}
	return result, nil
}
