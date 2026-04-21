package casrestore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
)

// personRestoreAdapter 负责“个人路线”的秒传恢复。
// 注意：这里的“个人路线”只描述上传/秒传路径，不代表最终目录一定是个人目录。
// 严格按参考实现：个人秒传主链走 upload.cloud.189.cn 的 init/check/commit。
type personRestoreAdapter struct{}

// personRestoreResult 表示个人路线恢复后的中间结果。
type personRestoreResult struct {
	RestoredFileID   string
	RestoredFileName string
}

func (a *personRestoreAdapter) TryRestore(
	session *appsession.Session,
	panClient *cloudpan.PanClient,
	destinationType DestinationType,
	targetFolderID string,
	fileName string,
	info *casparser.CasInfo,
) (*personRestoreResult, error) {
	if session == nil {
		return nil, errors.New("AppSession不能为空")
	}
	if panClient == nil {
		return nil, errors.New("PanClient不能为空")
	}
	if info == nil {
		return nil, errors.New("CAS信息不能为空")
	}
	if fileName == "" {
		fileName = info.Name
	}

	personParentID := ""
	if destinationType == DestinationTypePerson {
		personParentID = targetFolderID
	}

	restoredFileID, err := a.personRapidUpload(session, personParentID, info, fileName)
	if err != nil {
		return nil, err
	}
	result := &personRestoreResult{
		RestoredFileID:   restoredFileID,
		RestoredFileName: fileName,
	}
	if destinationType == DestinationTypePerson {
		return result, nil
	}

	// 严格按参考实现收口：当前已确认的参考主链只覆盖个人秒传本身，以及失败后切家庭中转到个人。
	// 尚未找到与当前产品语义等价、且可直接照搬的 person -> family 收尾链路，因此这里不能继续保留猜测型实现。
	return nil, fmt.Errorf("不支持的操作: 参考实现暂无 person -> family 恢复主链")
}

func (a *personRestoreAdapter) personRapidUpload(session *appsession.Session, personParentID string, info *casparser.CasInfo, fileName string) (string, error) {
	if _, err := getSessionKeyForUpload(session); err != nil {
		return "", err
	}
	sliceSize := calcCasSliceSize(info.Size)

	initRes, err := uploadRequest(session, "/person/initMultiUpload", map[string]string{
		"parentFolderId": personParentID,
		"fileName":       url.QueryEscape(fileName),
		"fileSize":       fmt.Sprintf("%d", info.Size),
		"sliceSize":      fmt.Sprintf("%d", sliceSize),
		"lazyCheck":      "1",
	})
	if err != nil {
		return "", err
	}
	uploadFileID := uploadRespDataString(initRes, "data", "uploadFileId")
	if uploadFileID == "" {
		payload, _ := marshalLimitJSON(initRes)
		return "", fmt.Errorf("CAS秒传初始化失败: 缺少uploadFileId (响应: %s)", payload)
	}
	fileDataExists := uploadRespDataBoolInt(initRes, "data", "fileDataExists")

	time.Sleep(500 * time.Millisecond)

	if !fileDataExists {
		checkRes, err := uploadRequest(session, "/person/checkTransSecond", map[string]string{
			"fileMd5":      info.MD5,
			"sliceMd5":     info.SliceMD5,
			"uploadFileId": uploadFileID,
		})
		if err != nil {
			return "", err
		}
		fileDataExists = uploadRespDataBoolInt(checkRes, "data", "fileDataExists")
	}
	if !fileDataExists {
		return "", fmt.Errorf("CAS秒传失败: 云端不存在该文件数据 (%s)", fileName)
	}

	time.Sleep(500 * time.Millisecond)

	var (
		lastErr   error
		commitRes *uploadResponse
	)
	for retry := 0; retry < maxCommitRetry; retry++ {
		commitRes, lastErr = uploadRequest(session, "/person/commitMultiUploadFile", map[string]string{
			"uploadFileId": uploadFileID,
			"fileMd5":      info.MD5,
			"sliceMd5":     info.SliceMD5,
			"lazyCheck":    "1",
			"opertype":     "3",
		})
		if lastErr == nil {
			restoredFileID := firstNonEmpty(
				uploadRespDataString(commitRes, "file", "userFileId"),
				uploadRespDataString(commitRes, "file", "id"),
				uploadRespDataString(commitRes, "data", "fileId"),
			)
			if restoredFileID == "" {
				return "", fmt.Errorf("CAS秒传commit响应缺少文件ID")
			}
			return restoredFileID, nil
		}
		if _, ok := lastErr.(blacklistedError); ok {
			return "", lastErr
		}
		if httpErr, ok := lastErr.(httpError); ok && httpErr.StatusCode == http.StatusForbidden && retry < maxCommitRetry-1 {
			clearUploadRSAKeyCache(session)
			time.Sleep(time.Duration(retry+1) * 2 * time.Second)
			continue
		}
		return "", lastErr
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("CAS秒传commit失败")
}

func marshalLimitJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	if len(b) > 300 {
		return string(b[:300]), nil
	}
	return string(b), nil
}
