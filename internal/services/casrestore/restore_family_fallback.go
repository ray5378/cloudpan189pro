package casrestore

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/tickstep/cloudpan189-api/cloudpan/apiutil"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
)

const familyBatchAPIBase = "https://api.cloud.189.cn"

type familyRestoreAdapter struct{}

type familyRestoreResult struct {
	FamilyID         int64
	RestoredFileID   string
	RestoredFileName string
}

// familyRestoreAdapter 负责“家庭路线”的秒传恢复。
// 当前它只承担 family -> family 这条链：
// 1. upload.cloud.189.cn 家庭秒传（init/check/commit）
// 2. 结果直接落到家庭目录
//
// 下面这些属于已对齐项，不能随意改动：
// - familyId 选择优先 userRole=1
// - family root 通过 listFiles/path 解析，不再硬编码 -11
// - init 不传 md5，且 lazyCheck=1
// - commit 必须保留 403 retry + 清 RSA cache
// - familyFileId 提取顺序必须保持参考顺序
//
// 注意：family -> person 已收口到 refsdk 主链，不再由这里承担。
func (a *familyRestoreAdapter) TryRestore(
	session *appsession.Session,
	panClient *cloudpan.PanClient,
	destinationType DestinationType,
	targetFolderID string,
	fileName string,
	info *casparser.CasInfo,
) (*familyRestoreResult, error) {
	if session == nil {
		return nil, errors.New("AppSession不能为空")
	}
	if panClient == nil {
		return nil, errors.New("PanClient不能为空")
	}
	if info == nil {
		return nil, errors.New("CAS信息不能为空")
	}
	if destinationType != DestinationTypeFamily {
		return nil, fmt.Errorf("familyRestoreAdapter 不再承担 family -> person，当前仅支持 refsdk 主链")
	}

	familyID := reqFamilyIDFromContext(session)
	var err error
	if familyID <= 0 {
		familyID, err = a.pickFamilyID(panClient)
		if err != nil {
			return nil, err
		}
	}
	if fileName == "" {
		fileName = info.Name
	}

	familyFolderID := normalizeFamilyFolderID(targetFolderID)
	familyFileID, err := a.familyRapidUpload(session, familyID, familyFolderID, info, fileName)
	if err != nil {
		return nil, err
	}
	return &familyRestoreResult{
		FamilyID:         familyID,
		RestoredFileID:   familyFileID,
		RestoredFileName: fileName,
	}, nil
}

func (a *familyRestoreAdapter) familyRapidUpload(session *appsession.Session, familyID int64, familyFolderID string, info *casparser.CasInfo, fileName string) (string, error) {
	sessionKey := strings.TrimSpace(session.Token.SessionKey)
	if sessionKey == "" {
		return "", fmt.Errorf("家庭秒传失败: 缺少sessionKey")
	}
	sliceSize := calcCasSliceSize(info.Size)

	initRes, err := uploadRequest(session, "/family/initMultiUpload", map[string]string{
		"parentFolderId": familyFolderID,
		"familyId":       strconv.FormatInt(familyID, 10),
		"fileName":       url.QueryEscape(fileName),
		"fileSize":       strconv.FormatInt(info.Size, 10),
		"sliceSize":      strconv.FormatInt(sliceSize, 10),
		"lazyCheck":      "1",
	})
	if err != nil {
		return "", err
	}
	uploadFileID := uploadRespDataString(initRes, "data", "uploadFileId")
	if uploadFileID == "" {
		return "", fmt.Errorf("家庭秒传init失败: 缺少uploadFileId")
	}
	fileDataExists := uploadRespDataBoolInt(initRes, "data", "fileDataExists")

	time.Sleep(500 * time.Millisecond)

	if !fileDataExists {
		checkRes, err := uploadRequest(session, "/family/checkTransSecond", map[string]string{
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
		return "", fmt.Errorf("家庭秒传失败: 云端不存在该文件数据 (%s)", fileName)
	}

	time.Sleep(500 * time.Millisecond)

	var commitRes *uploadResponse
	var lastErr error
	for retry := 0; retry < maxCommitRetry; retry++ {
		commitRes, lastErr = uploadRequest(session, "/family/commitMultiUploadFile", map[string]string{
			"uploadFileId": uploadFileID,
			"fileMd5":      info.MD5,
			"sliceMd5":     info.SliceMD5,
			"lazyCheck":    "1",
			"opertype":     "3",
		})
		if lastErr == nil {
			break
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
	if commitRes == nil {
		if lastErr != nil {
			return "", lastErr
		}
		return "", fmt.Errorf("家庭秒传commit失败")
	}

	userFileID := uploadRespDataString(commitRes, "file", "userFileId")
	fileID := uploadRespDataString(commitRes, "file", "id")
	dataFileID := uploadRespDataString(commitRes, "data", "fileId")
	familyFileID := firstNonEmpty(userFileID, fileID, dataFileID)
	if familyFileID == "" {
		b, _ := json.Marshal(commitRes)
		return "", fmt.Errorf("家庭秒传commit响应缺少文件ID: %s", string(b))
	}
	return familyFileID, nil
}

func (a *familyRestoreAdapter) pickFamilyID(panClient *cloudpan.PanClient) (int64, error) {
	resp, apiErr := panClient.AppFamilyGetFamilyList()
	if apiErr != nil {
		return 0, errors.Wrap(apiErr, "获取家庭列表失败")
	}
	if resp == nil || len(resp.FamilyInfoList) == 0 {
		return 0, fmt.Errorf("当前账号没有可用家庭")
	}
	for _, item := range resp.FamilyInfoList {
		if item != nil && item.UserRole == 1 && item.FamilyId > 0 {
			return item.FamilyId, nil
		}
	}
	for _, item := range resp.FamilyInfoList {
		if item != nil && item.FamilyId > 0 {
			return item.FamilyId, nil
		}
	}
	return 0, fmt.Errorf("当前账号没有可用家庭")
}

func (a *familyRestoreAdapter) getFamilyRootFolderID(session *appsession.Session, familyID int64) (string, error) {
	if session == nil {
		return "", fmt.Errorf("AppSession不能为空")
	}
	if familyID <= 0 {
		return "", fmt.Errorf("家庭中转不可用: 当前账号没有家庭组")
	}
	targetURL := fmt.Sprintf("https://api.cloud.189.cn/family/file/listFiles.action?familyId=%d&folderId=&needPath=true&pageNum=1&pageSize=1", familyID)
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return "", err
	}
	dateOfGmt := apiutil.DateOfGmtStr()
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Date", dateOfGmt)
	req.Header.Set("SessionKey", session.Token.FamilySessionKey)
	req.Header.Set("Signature", apiutil.SignatureOfHmac(session.Token.FamilySessionSecret, session.Token.FamilySessionKey, http.MethodGet, targetURL, dateOfGmt))
	req.Header.Set("X-Request-ID", apiutil.XRequestId())
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("获取家庭根目录ID失败: http %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Path []struct {
			FileID string `json:"fileId"`
		} `json:"path"`
		FileListAO struct {
			Path []struct {
				FileID string `json:"fileId"`
			} `json:"path"`
		} `json:"fileListAO"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	for i := len(parsed.Path) - 1; i >= 0; i-- {
		fileID := strings.TrimSpace(parsed.Path[i].FileID)
		if fileID != "" && fileID != "-11" && fileID != "-16" {
			return fileID, nil
		}
	}
	if len(parsed.FileListAO.Path) > 0 {
		fileID := strings.TrimSpace(parsed.FileListAO.Path[0].FileID)
		if fileID != "" {
			return fileID, nil
		}
	}
	return "", nil
}

func normalizeFamilyFolderID(folderID string) string {
	if folderID == "-11" {
		return ""
	}
	return folderID
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
