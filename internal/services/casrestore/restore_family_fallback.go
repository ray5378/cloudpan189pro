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
const cloudWebBaseURL = "https://cloud.189.cn"
const cloudWebOpenAppKey = "600100422"

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
	ErrorCode      string `json:"errorCode"`
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

	familyFolderID := ""
	if destinationType == DestinationTypeFamily {
		familyFolderID = normalizeFamilyFolderID(targetFolderID)
	} else {
		familyFolderID, err = a.getFamilyRootFolderID(session, familyID)
		if err != nil {
			return nil, err
		}
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
	if destinationType != DestinationTypeFamily {
		return nil, fmt.Errorf("familyRestoreAdapter 不再承担 family -> person，当前仅支持 refsdk 主链")
	}
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
			if resp.FailedCount > 0 && resp.SuccessedCount == 0 {
				if strings.TrimSpace(resp.ErrorCode) != "" {
					return fmt.Errorf("家庭中转批量任务失败 taskStatus=%d failed=%d successed=%d skip=%d errorCode=%s", resp.TaskStatus, resp.FailedCount, resp.SuccessedCount, resp.SkipCount, resp.ErrorCode)
				}
				return fmt.Errorf("家庭中转批量任务失败 taskStatus=%d failed=%d successed=%d skip=%d", resp.TaskStatus, resp.FailedCount, resp.SuccessedCount, resp.SkipCount)
			}
			return nil
		}
	}
	return fmt.Errorf("家庭中转批量任务超时 taskStatus=%d", lastStatus)
}

func (a *familyRestoreAdapter) safeDeleteFamilyFile(session *appsession.Session, familyID int64, fileID, fileName string) error {
	accessToken := strings.TrimSpace(session.Token.AccessToken)
	if accessToken == "" {
		return fmt.Errorf("家庭中转清理失败: 无法获取AccessToken")
	}

	deleteResp := new(batchTaskCreateResp)
	if err := doAccessTokenFormJSONRequest(accessToken, familyBatchAPIBase+"/open/batch/createBatchTask.action", map[string]string{
		"type":           "DELETE",
		"taskInfos":      fmt.Sprintf(`[{"fileId":"%s","fileName":"%s","isFolder":0}]`, fileID, escapeJSONString(fileName)),
		"targetFolderId": "",
		"familyId":       strconv.FormatInt(familyID, 10),
	}, 30*time.Second, deleteResp); err != nil {
		return errors.Wrap(err, "提交DELETE任务失败")
	}
	if batchRespError(deleteResp.ResCode, deleteResp.ResMessage) {
		return fmt.Errorf("提交DELETE任务失败: %s", deleteResp.ResMessage)
	}
	if deleteResp.TaskID == "" {
		return fmt.Errorf("提交DELETE任务失败: 缺少taskId")
	}
	if err := a.waitForBatchTask(accessToken, "DELETE", deleteResp.TaskID, 2*time.Minute); err != nil {
		return err
	}

	clearResp := new(batchTaskCreateResp)
	if err := doAccessTokenFormJSONRequest(accessToken, familyBatchAPIBase+"/open/batch/createBatchTask.action", map[string]string{
		"type":           "CLEAR_RECYCLE",
		"taskInfos":      "[]",
		"targetFolderId": "",
		"familyId":       strconv.FormatInt(familyID, 10),
	}, 30*time.Second, clearResp); err != nil {
		return errors.Wrap(err, "提交CLEAR_RECYCLE任务失败")
	}
	if batchRespError(clearResp.ResCode, clearResp.ResMessage) {
		return fmt.Errorf("提交CLEAR_RECYCLE任务失败: %s", clearResp.ResMessage)
	}
	if clearResp.TaskID == "" {
		return fmt.Errorf("提交CLEAR_RECYCLE任务失败: 缺少taskId")
	}
	if err := a.waitForBatchTask(accessToken, "CLEAR_RECYCLE", clearResp.TaskID, 2*time.Minute); err != nil {
		return err
	}
	return nil
}

type ssKeyAccessTokenResp struct {
	AccessToken string `json:"accessToken"`
}

func getAccessTokenBySsKey(session *appsession.Session) (string, error) {
	if session == nil {
		return "", fmt.Errorf("AppSession不能为空")
	}
	sessionKey := strings.TrimSpace(session.Token.SessionKey)
	if sessionKey == "" {
		return "", fmt.Errorf("sessionKey为空")
	}
	targetURL := familyBatchAPIBase + "/open/oauth2/getAccessTokenBySsKey.action?sessionKey=" + url.QueryEscape(sessionKey)
	timestamp, signature := buildWebOpenSignature(targetURL, nil)
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Sign-Type", "1")
	req.Header.Set("Signature", signature)
	req.Header.Set("Timestamp", timestamp)
	req.Header.Set("AppKey", cloudWebOpenAppKey)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	req.Header.Set("Referer", cloudWebBaseURL+"/web/main/")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	out := new(ssKeyAccessTokenResp)
	if err := json.Unmarshal(body, out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return "", fmt.Errorf("响应缺少accessToken")
	}
	return strings.TrimSpace(out.AccessToken), nil
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
	req.Header.Set("Accesstoken", strings.TrimSpace(accessToken))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	req.Header.Set("Referer", cloudWebBaseURL+"/web/main/")
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

// buildAccessTokenSignature 严格参照 cloud189-auto-save/cloud189-sdk：
// 把 AccessToken / Timestamp / 全部业务参数放进同一个 map，整体按 key 字典序排序，再做 md5 hex lower。
func buildAccessTokenSignature(accessToken string, params map[string]string) (string, string) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	payload := make(map[string]string, len(params)+2)
	for k, v := range params {
		payload[k] = v
	}
	payload["AccessToken"] = accessToken
	payload["Timestamp"] = timestamp
	return timestamp, buildSortedMD5Signature(payload)
}

func buildWebOpenSignature(targetURL string, params map[string]string) (string, string) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	payload := make(map[string]string, 4)
	if parsed, err := url.Parse(targetURL); err == nil {
		for key, values := range parsed.Query() {
			if len(values) > 0 {
				payload[key] = values[0]
			}
		}
	}
	for k, v := range params {
		payload[k] = v
	}
	payload["Timestamp"] = timestamp
	payload["AppKey"] = cloudWebOpenAppKey
	return timestamp, buildSortedMD5Signature(payload)
}

func buildSortedMD5Signature(payload map[string]string) string {
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+payload[k])
	}
	sum := md5.Sum([]byte(strings.Join(parts, "&")))
	return hex.EncodeToString(sum[:])
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
