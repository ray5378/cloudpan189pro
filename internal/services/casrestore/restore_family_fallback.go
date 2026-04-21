package casrestore

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
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

type batchTaskCreateResp struct {
	ResCode    any    `json:"res_code"`
	ResMessage string `json:"res_message"`
	TaskID     string `json:"taskId"`
}

type batchTaskCheckResp struct {
	ResCode        any    `json:"res_code"`
	ResMessage     string `json:"res_message"`
	TaskStatus     int    `json:"taskStatus"`
	TaskID         string `json:"taskId"`
	FailedCount    int    `json:"failedCount"`
	SuccessedCount int    `json:"successedCount"`
	SkipCount      int    `json:"skipCount"`
}

// familyRestoreAdapter 负责“家庭路线”的秒传恢复。
// 严格按参照代码：
// 1. upload.cloud.189.cn 家庭秒传（init/check/commit）
// 2. 如目标是个人目录，则走 AccessToken 签名的 batch COPY
// 3. 成功后清理家庭中转文件
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
	familyID, err := a.pickFamilyID(panClient)
	if err != nil {
		return nil, err
	}
	if fileName == "" {
		fileName = info.Name
	}

	familyFolderID := ""
	if destinationType == DestinationTypeFamily {
		familyFolderID = normalizeFamilyFolderID(targetFolderID)
	}

	familyFileID, err := a.familyRapidUpload(session, familyID, familyFolderID, info, fileName)
	if err != nil {
		return nil, err
	}
	result := &familyRestoreResult{
		FamilyID:         familyID,
		RestoredFileID:   familyFileID,
		RestoredFileName: fileName,
	}
	if destinationType == DestinationTypeFamily {
		return result, nil
	}

	if err := a.copyFamilyFileToPersonal(session, familyID, familyFileID, targetFolderID, fileName); err != nil {
		_ = a.safeDeleteFamilyFile(panClient, familyID, familyFileID, fileName)
		return nil, err
	}
	_ = a.safeDeleteFamilyFile(panClient, familyID, familyFileID, fileName)
	return result, nil
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
		msg := strings.ToLower(lastErr.Error())
		if strings.Contains(msg, "http 403") && retry < maxCommitRetry-1 {
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

	familyFileID := firstNonEmpty(
		uploadRespDataString(commitRes, "file", "userFileId"),
		uploadRespDataString(commitRes, "file", "id"),
		uploadRespDataString(commitRes, "data", "fileId"),
	)
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

// copyFamilyFileToPersonal 严格参照 cloud189-auto-save 的 _copyFamilyFileToPersonal：
// 使用 AccessToken + Timestamp + 参数字典序拼接后的 MD5 小写签名，请求 /open/batch/createBatchTask.action(type=COPY, copyType=2)。
func (a *familyRestoreAdapter) copyFamilyFileToPersonal(session *appsession.Session, familyID int64, familyFileID, personalFolderID, fileName string) error {
	accessToken := strings.TrimSpace(session.Token.AccessToken)
	if accessToken == "" {
		return fmt.Errorf("家庭中转COPY失败: 无法获取AccessToken")
	}
	params := map[string]string{
		"type":           "COPY",
		"taskInfos":      fmt.Sprintf(`[{"fileId":"%s","fileName":"%s","isFolder":0}]`, familyFileID, escapeJSONString(fileName)),
		"targetFolderId": normalizePersonFolderID(targetFolderIDOrEmpty(personalFolderID)),
		"familyId":       strconv.FormatInt(familyID, 10),
		"groupId":        "null",
		"copyType":       "2",
		"shareId":        "null",
	}
	resp := new(batchTaskCreateResp)
	if err := doAccessTokenFormJSONRequest(accessToken, familyBatchAPIBase+"/open/batch/createBatchTask.action", params, 30*time.Second, resp); err != nil {
		return errors.Wrap(err, "家庭中转COPY失败")
	}
	if batchRespError(resp.ResCode, resp.ResMessage) {
		return fmt.Errorf("家庭中转COPY失败: %s", resp.ResMessage)
	}
	if resp.TaskID == "" {
		return fmt.Errorf("家庭中转COPY失败: 缺少taskId")
	}
	return a.waitForBatchTask(accessToken, "COPY", resp.TaskID, 30*time.Second)
}

// waitForBatchTask 严格参照 _waitForBatchTask：1s 轮询 /open/batch/checkBatchTask.action，taskStatus=4 视为成功。
func (a *familyRestoreAdapter) waitForBatchTask(accessToken, taskType, taskID string, maxWait time.Duration) error {
	if strings.TrimSpace(accessToken) == "" {
		return fmt.Errorf("批量任务查询失败: 无法获取AccessToken")
	}
	deadline := time.Now().Add(maxWait)
	lastStatus := 0
	for time.Now().Before(deadline) {
		time.Sleep(1 * time.Second)
		resp := new(batchTaskCheckResp)
		if err := doAccessTokenFormJSONRequest(accessToken, familyBatchAPIBase+"/open/batch/checkBatchTask.action", map[string]string{
			"type":   taskType,
			"taskId": taskID,
		}, 15*time.Second, resp); err != nil {
			return errors.Wrap(err, "批量任务查询失败")
		}
		if batchRespError(resp.ResCode, resp.ResMessage) {
			return fmt.Errorf("批量任务查询失败: %s", resp.ResMessage)
		}
		lastStatus = resp.TaskStatus
		if lastStatus == 4 {
			return nil
		}
	}
	return fmt.Errorf("家庭中转批量任务超时 taskStatus=%d", lastStatus)
}

func (a *familyRestoreAdapter) safeDeleteFamilyFile(panClient *cloudpan.PanClient, familyID int64, fileID, fileName string) error {
	_, apiErr := panClient.AppCreateBatchTask(familyID, &cloudpan.BatchTaskParam{
		TypeFlag: cloudpan.BatchTaskTypeDelete,
		TaskInfos: cloudpan.BatchTaskInfoList{&cloudpan.BatchTaskInfo{
			FileId:   fileID,
			FileName: fileName,
			IsFolder: 0,
		}},
	})
	if apiErr != nil {
		return apiErr
	}
	return nil
}

func doAccessTokenFormJSONRequest(accessToken string, targetURL string, params map[string]string, timeout time.Duration, out any) error {
	timestamp, signature := buildAccessTokenSignature(strings.TrimSpace(accessToken), params)
	req, err := http.NewRequest(http.MethodPost, targetURL, strings.NewReader(formURLEncode(params)))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Sign-Type", "1")
	req.Header.Set("Signature", signature)
	req.Header.Set("Timestamp", timestamp)
	req.Header.Set("AccessToken", strings.TrimSpace(accessToken))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}

// buildAccessTokenSignature 严格参照 JS：
// AccessToken=xxx&Timestamp=xxx&key1=val1&key2=val2...（按 key 字典序） -> md5 hex lower。
func buildAccessTokenSignature(accessToken string, params map[string]string) (string, string) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{"AccessToken=" + accessToken, "Timestamp=" + timestamp}
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	sum := md5.Sum([]byte(strings.Join(parts, "&")))
	return timestamp, hex.EncodeToString(sum[:])
}

func formURLEncode(params map[string]string) string {
	vals := url.Values{}
	for k, v := range params {
		vals.Set(k, v)
	}
	return vals.Encode()
}

func batchRespError(code any, _ string) bool {
	switch v := code.(type) {
	case nil:
		return false
	case float64:
		return int(v) != 0
	case int:
		return v != 0
	case string:
		return v != "" && v != "0"
	default:
		return false
	}
}

func normalizeFamilyFolderID(folderID string) string {
	if folderID == "-11" {
		return ""
	}
	return folderID
}

func normalizePersonFolderID(folderID string) string {
	return folderID
}

func targetFolderIDOrEmpty(folderID string) string {
	return folderID
}

func escapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return s
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
