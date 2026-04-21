package casrestore

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
)

// familyRestoreAdapter 负责“家庭路线”的秒传恢复。
// 注意：这里的“家庭路线”只描述上传/秒传路径，不代表最终目录一定是家庭目录。
type familyRestoreAdapter struct{}

type familyRestoreResult struct {
	FamilyID         int64
	RestoredFileID   string
	RestoredFileName string
}

func (a *familyRestoreAdapter) TryRestore(
	panClient *cloudpan.PanClient,
	destinationType DestinationType,
	targetFolderID string,
	fileName string,
	info *casparser.CasInfo,
) (*familyRestoreResult, error) {
	if panClient == nil {
		return nil, errors.New("PanClient不能为空")
	}
	if info == nil {
		return nil, errors.New("CAS信息不能为空")
	}
	familyID, err := a.pickFamilyID(panClient)
	if err != nil {
		return nil, err
	}
	if fileName == "" {
		fileName = info.Name
	}

	// 家庭路线下，家庭目录可直接作为上传父目录；若最终目标是个人目录，则先落家庭，再转个人。
	familyParentID := ""
	if destinationType == DestinationTypeFamily {
		familyParentID = targetFolderID
	}

	createRes, apiErr := panClient.AppFamilyCreateUploadFile(&cloudpan.AppCreateUploadFileParam{
		FamilyId:       familyID,
		ParentFolderId: familyParentID,
		FileName:       fileName,
		Size:           info.Size,
		Md5:            info.MD5,
		LastWrite:      time.Now().Format("2006-01-02 15:04:05"),
		LocalPath:      "/tmp/" + fileName,
	})
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "创建家庭上传任务失败")
	}
	if createRes == nil || createRes.UploadFileId == "" {
		return nil, fmt.Errorf("创建家庭上传任务失败: 缺少uploadFileId")
	}

	status, apiErr := panClient.AppFamilyGetUploadFileStatus(familyID, createRes.UploadFileId)
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "查询家庭上传状态失败")
	}
	if status == nil || status.FileDataExists != 1 {
		return nil, fmt.Errorf("家庭云端不存在可直接命中的文件数据")
	}

	commitURL := createRes.FileCommitUrl
	if commitURL == "" && status.FileCommitUrl != "" {
		commitURL = status.FileCommitUrl
	}
	if commitURL == "" {
		return nil, fmt.Errorf("缺少家庭commit地址")
	}

	commitRes, apiErr := panClient.AppFamilyUploadFileCommit(familyID, commitURL, createRes.UploadFileId, createRes.XRequestId)
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "提交家庭恢复失败")
	}
	if commitRes == nil || commitRes.Id == "" {
		return nil, fmt.Errorf("家庭恢复提交成功但未返回文件ID")
	}

	result := &familyRestoreResult{
		FamilyID:         familyID,
		RestoredFileID:   commitRes.Id,
		RestoredFileName: commitRes.Name,
	}
	if destinationType == DestinationTypeFamily {
		return result, nil
	}

	ok, apiErr := panClient.AppFamilySaveFileToPersonCloud(familyID, []string{commitRes.Id})
	if apiErr != nil {
		return nil, errors.Wrap(apiErr, "家庭文件回灌个人云失败")
	}
	if !ok {
		return nil, fmt.Errorf("家庭文件回灌个人云失败")
	}

	if targetFolderID != "" {
		moved, apiErr := panClient.AppMoveFile([]string{commitRes.Id}, targetFolderID)
		if apiErr != nil {
			return nil, errors.Wrap(apiErr, "回灌个人云后移动文件失败")
		}
		if moved != nil && len(moved.FileList) > 0 && moved.FileList[0] != nil {
			result.RestoredFileID = moved.FileList[0].FileId
			result.RestoredFileName = moved.FileList[0].FileName
		}
	}

	return result, nil
}

func (a *familyRestoreAdapter) pickFamilyID(panClient *cloudpan.PanClient) (int64, error) {
	resp, apiErr := panClient.AppFamilyGetFamilyList()
	if apiErr != nil {
		return 0, errors.Wrap(apiErr, "获取家庭列表失败")
	}
	if resp == nil || len(resp.FamilyInfoList) == 0 || resp.FamilyInfoList[0] == nil {
		return 0, fmt.Errorf("当前账号没有可用家庭")
	}
	return resp.FamilyInfoList[0].FamilyId, nil
}
