package casrestore

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const cloudWebBaseURL = "https://cloud.189.cn"

func (c *refSDKClient) copyFamilyFileToPersonal(accessToken string, familyID int64, familyFileID, personalFolderID, fileName string) error {
	params := map[string]string{
		"type":           "COPY",
		"taskInfos":      fmt.Sprintf(`[{"fileId":"%s","fileName":"%s","isFolder":0}]`, familyFileID, refSDKEscapeJSONString(fileName)),
		"targetFolderId": strings.TrimSpace(personalFolderID),
		"familyId":       fmt.Sprintf("%d", familyID),
		"groupId":        "null",
		"copyType":       "2",
		"shareId":        "null",
	}
	resp := new(refSDKBatchCreateResp)
	if err := c.doBatchFormRequest(accessToken, familyBatchAPIBase+"/open/batch/createBatchTask.action", params, 30*time.Second, resp); err != nil {
		return err
	}
	if batchRespError(resp.ResCode, resp.ResMessage) {
		return fmt.Errorf("COPY失败: %s", resp.ResMessage)
	}
	if strings.TrimSpace(resp.TaskID) == "" {
		return fmt.Errorf("COPY失败: 缺少taskId")
	}
	return c.waitForBatchTask(accessToken, "COPY", resp.TaskID, 60*time.Second)
}

func (c *refSDKClient) waitForBatchTask(accessToken, taskType, taskID string, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		resp := new(refSDKBatchCheckResp)
		if err := c.doBatchFormRequest(accessToken, familyBatchAPIBase+"/open/batch/checkBatchTask.action", map[string]string{
			"taskId": taskID,
			"type":   taskType,
		}, 20*time.Second, resp); err != nil {
			return err
		}
		if batchRespError(resp.ResCode, resp.ResMessage) {
			return fmt.Errorf("批量任务查询失败: %s", resp.ResMessage)
		}
		if resp.TaskStatus == 4 {
			if resp.FailedCount > 0 && resp.SuccessedCount == 0 {
				if strings.TrimSpace(resp.ErrorCode) != "" {
					return fmt.Errorf("批量任务失败 taskType=%s failed=%d successed=%d skip=%d errorCode=%s", taskType, resp.FailedCount, resp.SuccessedCount, resp.SkipCount, resp.ErrorCode)
				}
				return fmt.Errorf("批量任务失败 taskType=%s failed=%d successed=%d skip=%d", taskType, resp.FailedCount, resp.SuccessedCount, resp.SkipCount)
			}
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("批量任务超时 type=%s taskId=%s", taskType, taskID)
}

func (c *refSDKClient) safeDeleteFamilyFile(accessToken string, familyID int64, fileID, fileName string) error {
	return c.safeDeleteFamilyNode(accessToken, familyID, fileID, fileName, false)
}

func (c *refSDKClient) safeDeleteFamilyNode(accessToken string, familyID int64, fileID, fileName string, isFolder bool) error {
	folderFlag := 0
	if isFolder {
		folderFlag = 1
	}
	deleteResp := new(refSDKBatchCreateResp)
	if err := c.doBatchFormRequest(accessToken, familyBatchAPIBase+"/open/batch/createBatchTask.action", map[string]string{
		"type":           "DELETE",
		"taskInfos":      fmt.Sprintf(`[{"fileId":"%s","fileName":"%s","isFolder":%d}]`, fileID, refSDKEscapeJSONString(fileName), folderFlag),
		"targetFolderId": "",
		"familyId":       fmt.Sprintf("%d", familyID),
	}, 30*time.Second, deleteResp); err != nil {
		return err
	}
	if batchRespError(deleteResp.ResCode, deleteResp.ResMessage) {
		return fmt.Errorf("DELETE失败: %s", deleteResp.ResMessage)
	}
	if strings.TrimSpace(deleteResp.TaskID) != "" {
		if err := c.waitForBatchTask(accessToken, "DELETE", deleteResp.TaskID, 2*time.Minute); err != nil {
			return err
		}
	}

	clearResp := new(refSDKBatchCreateResp)
	if err := c.doBatchFormRequest(accessToken, familyBatchAPIBase+"/open/batch/createBatchTask.action", map[string]string{
		"type":           "CLEAR_RECYCLE",
		"taskInfos":      "[]",
		"targetFolderId": "",
		"familyId":       fmt.Sprintf("%d", familyID),
	}, 30*time.Second, clearResp); err != nil {
		return err
	}
	if batchRespError(clearResp.ResCode, clearResp.ResMessage) {
		return fmt.Errorf("CLEAR_RECYCLE失败: %s", clearResp.ResMessage)
	}
	if strings.TrimSpace(clearResp.TaskID) != "" {
		if err := c.waitForBatchTask(accessToken, "CLEAR_RECYCLE", clearResp.TaskID, 2*time.Minute); err != nil {
			return err
		}
	}
	return nil
}

func (c *refSDKClient) doBatchFormRequest(accessToken string, targetURL string, params map[string]string, timeout time.Duration, out any) error {
	timestamp, signature := refSDKBuildBatchSignature(strings.TrimSpace(accessToken), params)
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	req, err := http.NewRequest(http.MethodPost, targetURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Sign-Type", "1")
	req.Header.Set("Signature", signature)
	req.Header.Set("Timestamp", timestamp)
	req.Header.Set("Accesstoken", strings.TrimSpace(accessToken))
	req.Header.Set("User-Agent", refSDKUserAgent)
	req.Header.Set("Referer", cloudWebBaseURL+"/web/main/")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := *c.httpClient
	client.Timeout = timeout
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

func refSDKEscapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return s
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

func normalizePersonFolderID(folderID string) string {
	return folderID
}
