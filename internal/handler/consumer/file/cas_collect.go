package file

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"go.uber.org/zap"
)

func buildPanClient(session *appsession.Session) *cloudpan.PanClient {
	if session == nil {
		return nil
	}
	webToken := cloudpan.WebLoginToken{}
	if cookie := cloudpan.RefreshCookieToken(session.Token.SessionKey); cookie != "" {
		webToken.CookieLoginUser = cookie
	}
	return cloudpan.NewPanClient(webToken, session.Token)
}

type batchTaskCreateResp struct {
	ResCode    any    `json:"res_code"`
	ResMessage string `json:"res_message"`
	TaskID     string `json:"taskId"`
}

type batchTaskCheckResp struct {
	ResCode     any    `json:"res_code"`
	ResMessage  string `json:"res_message"`
	TaskStatus  int    `json:"taskStatus"`
	FailedCount int    `json:"failedCount"`
}

func batchRespError(code any) bool {
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

func buildAccessTokenSignature(accessToken string, params map[string]string) (string, string) {
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
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

func doAccessTokenFormJSONRequest(accessToken string, targetURL string, params map[string]string, timeout time.Duration, out any) error {
	timestamp, signature := buildAccessTokenSignature(strings.TrimSpace(accessToken), params)
	vals := url.Values{}
	for k, v := range params {
		vals.Set(k, v)
	}
	req, err := http.NewRequest(http.MethodPost, targetURL, strings.NewReader(vals.Encode()))
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

func waitForShareSaveTask(accessToken, taskID string, maxWait time.Duration) error {
	if strings.TrimSpace(accessToken) == "" {
		return fmt.Errorf("自动归集CAS失败: 无法获取AccessToken")
	}
	deadline := time.Now().Add(maxWait)
	lastStatus := 0
	for time.Now().Before(deadline) {
		time.Sleep(1 * time.Second)
		resp := new(batchTaskCheckResp)
		if err := doAccessTokenFormJSONRequest(accessToken, "https://api.cloud.189.cn/open/batch/checkBatchTask.action", map[string]string{
			"type":   "SHARE_SAVE",
			"taskId": taskID,
		}, 15*time.Second, resp); err != nil {
			return fmt.Errorf("自动归集CAS失败: 查询SHARE_SAVE任务失败: %w", err)
		}
		if batchRespError(resp.ResCode) {
			return fmt.Errorf("自动归集CAS失败: 查询SHARE_SAVE任务失败: %s", resp.ResMessage)
		}
		lastStatus = resp.TaskStatus
		if lastStatus == 4 {
			return nil
		}
		if resp.FailedCount > 0 {
			return fmt.Errorf("自动归集CAS失败: SHARE_SAVE任务失败 taskStatus=%d failedCount=%d", resp.TaskStatus, resp.FailedCount)
		}
	}
	return fmt.Errorf("自动归集CAS失败: SHARE_SAVE任务超时 taskStatus=%d", lastStatus)
}

func (h *handler) collectSubscribeShareCAS(ctx context.Context, panClient *cloudpan.PanClient, targetFolderID string, file *models.VirtualFile) error {
	shareID, ok := file.Addition.Int64(consts.FileAdditionKeyShareId)
	if !ok || shareID <= 0 {
		return fmt.Errorf("自动归集CAS失败: 缺少订阅分享ID")
	}
	fileID := strings.TrimSpace(file.CloudId)
	if fileID == "" {
		return fmt.Errorf("自动归集CAS失败: 缺少订阅文件ID")
	}

	// 优先使用 cloud token 的 accessToken 直连 SHARE_SAVE，避免强依赖 AppLogin(username/password)。
	if shared.SettingAddition.CasTargetTokenId > 0 {
		if token, err := h.cloudTokenService.Query(ctx, shared.SettingAddition.CasTargetTokenId); err == nil && token != nil && strings.TrimSpace(token.AccessToken) != "" {
			resp := new(batchTaskCreateResp)
			if err := doAccessTokenFormJSONRequest(strings.TrimSpace(token.AccessToken), "https://api.cloud.189.cn/open/batch/createBatchTask.action", map[string]string{
				"type":           "SHARE_SAVE",
				"taskInfos":      fmt.Sprintf(`[{"fileId":"%s","fileName":"%s","isFolder":0}]`, fileID, strings.ReplaceAll(file.Name, `"`, `\"`)),
				"targetFolderId": targetFolderID,
				"shareId":        fmt.Sprintf("%d", shareID),
			}, 30*time.Second, resp); err != nil {
				return fmt.Errorf("自动归集CAS失败: 提交SHARE_SAVE任务失败: %w", err)
			}
			if batchRespError(resp.ResCode) {
				return fmt.Errorf("自动归集CAS失败: 提交SHARE_SAVE任务失败: %s", resp.ResMessage)
			}
			if strings.TrimSpace(resp.TaskID) == "" {
				return fmt.Errorf("自动归集CAS失败: SHARE_SAVE未返回任务ID")
			}
			return waitForShareSaveTask(strings.TrimSpace(token.AccessToken), resp.TaskID, 2*time.Minute)
		}
	}

	if panClient == nil {
		return fmt.Errorf("自动归集CAS失败: 无法获取PanClient")
	}
	taskID, apiErr := panClient.CreateBatchTask(&cloudpan.BatchTaskParam{
		TypeFlag: cloudpan.BatchTaskTypeShareSave,
		TaskInfos: cloudpan.BatchTaskInfoList{
			&cloudpan.BatchTaskInfo{FileId: fileID, FileName: file.Name, IsFolder: 0},
		},
		TargetFolderId: targetFolderID,
		ShareId:        shareID,
	})
	if apiErr != nil {
		return fmt.Errorf("自动归集CAS失败: 提交SHARE_SAVE任务失败: %w", apiErr)
	}
	if strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("自动归集CAS失败: SHARE_SAVE未返回任务ID")
	}
	deadline := time.Now().Add(2 * time.Minute)
	lastStatus := cloudpan.BatchTaskStatusNotAction
	for time.Now().Before(deadline) {
		time.Sleep(1 * time.Second)
		result, checkErr := panClient.CheckBatchTask(cloudpan.BatchTaskTypeShareSave, taskID)
		if checkErr != nil {
			return fmt.Errorf("自动归集CAS失败: 查询SHARE_SAVE任务失败: %w", checkErr)
		}
		if result == nil {
			continue
		}
		lastStatus = result.TaskStatus
		if result.TaskStatus == cloudpan.BatchTaskStatusOk {
			return nil
		}
		if result.FailedCount > 0 {
			return fmt.Errorf("自动归集CAS失败: SHARE_SAVE任务失败 taskStatus=%d failedCount=%d", result.TaskStatus, result.FailedCount)
		}
	}
	return fmt.Errorf("自动归集CAS失败: SHARE_SAVE任务超时 taskStatus=%d", lastStatus)
}

func (h *handler) tryCollectCASFromVirtualFile(ctx context.Context, file *models.VirtualFile) error {
	cfg := shared.SettingAddition
	if !cfg.CasTargetEnabled || !cfg.CasAutoCollectEnabled {
		return nil
	}
	if cfg.CasTargetTokenId <= 0 {
		return nil
	}
	if file == nil || file.IsDir {
		return nil
	}
	if file.OsType != models.OsTypeShareFile && file.OsType != models.OsTypeSubscribeShareFile {
		return nil
	}
	if !strings.HasSuffix(strings.ToLower(file.Name), ".cas") {
		return nil
	}
	if cfg.CasTargetType != "person" {
		return fmt.Errorf("当前自动归集仅先支持保存到个人目录")
	}

	session, err := h.appSessionService.GetByTokenID(ctx, cfg.CasTargetTokenId)
	if err != nil {
		return fmt.Errorf("获取CAS目标App会话失败: %w", err)
	}
	panClient := buildPanClient(session)
	if panClient == nil {
		return fmt.Errorf("创建CAS目标PanClient失败")
	}

	targetFolderID := cfg.CasTargetFolderId
	if targetFolderID == "" {
		targetFolderID = "-11"
	}

	if cfg.CasAutoCollectPreservePath {
		fullPath, fullPathErr := h.virtualFileService.CalFullPath(ctx, file.ID)
		if fullPathErr == nil {
			relDir := strings.TrimSpace(path.Dir(strings.TrimPrefix(fullPath, "/")))
			if relDir != "" && relDir != "." {
				folder, apiErr := panClient.AppMkdirRecursive(0, targetFolderID, relDir, 0, strings.Split(relDir, "/"))
				if apiErr != nil {
					return fmt.Errorf("创建CAS归集目录失败: %w", apiErr)
				}
				if folder != nil && folder.FileId != "" {
					targetFolderID = folder.FileId
				}
			}
		}
	}

	switch file.OsType {
	case models.OsTypeSubscribeShareFile:
		if err := h.collectSubscribeShareCAS(ctx, panClient, targetFolderID, file); err != nil {
			return err
		}
	case models.OsTypeShareFile:
		return fmt.Errorf("当前自动归集暂未处理普通分享文件")
	}

	ctx.Info("存储刷新链CAS自动归集成功",
		zap.String("name", file.Name),
		zap.String("targetFolderId", targetFolderID),
		zap.String("fileId", file.CloudId),
	)
	return nil
}
