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

// refreshCASDirCacheIfNeeded 按需从目标云盘目录拉取文件名并刷新本地缓存。
//
// 重要原则（不要随意改）：
// 1. 这里缓存的数据源必须来自目标云盘目录本身，不能改成本地成功记录推断。
// 2. 这里的缓存只应该服务于“开启自动刷新的存储”的自动转存去重。
// 3. 不要为了图省事，把所有存储/所有手动转存都接到这条缓存刷新链上。
// 4. 真相永远在云盘，本地缓存只是镜像；缓存过期/失真时，应以重新拉取云盘目录为准。
func (h *handler) refreshCASDirCacheIfNeeded(ctx context.Context, targetTokenID int64, targetFolderID string, runtime *casCollectRuntime) error {
	if h.casTargetCacheService == nil || runtime == nil || runtime.panClient == nil {
		return nil
	}
	needRefresh, err := h.casTargetCacheService.NeedsRefresh(ctx, targetTokenID, targetFolderID, 24*time.Hour)
	if err != nil || !needRefresh {
		return err
	}
	param := cloudpan.NewAppFileListParam()
	param.FileId = targetFolderID
	param.PageSize = 200
	result, apiErr := runtime.panClient.AppGetAllFileList(param)
	if apiErr != nil {
		return fmt.Errorf("刷新CAS目标目录缓存失败: %w", apiErr)
	}
	items := make([]*models.CasTargetDirCache, 0)
	now := time.Now()
	if result != nil {
		for _, fi := range result.FileList {
			if fi == nil {
				continue
			}
			items = append(items, &models.CasTargetDirCache{
				TargetTokenID:  targetTokenID,
				TargetFolderID: targetFolderID,
				FileName:       strings.TrimSpace(fi.FileName),
				IsDir:          fi.IsFolder,
				RefreshedAt:    now,
			})
		}
	}
	return h.casTargetCacheService.RefreshDir(ctx, targetTokenID, targetFolderID, items)
}

func (h *handler) getOrCreateCASCollectRuntime(ctx context.Context, tokenID int64) (*casCollectRuntime, error) {
	cacheKey := fmt.Sprintf("%s:%d", ctx.Trace.ID(), tokenID)
	if v, ok := h.casCollectRuntimeCache.Load(cacheKey); ok {
		if runtime, ok := v.(*casCollectRuntime); ok && runtime != nil {
			return runtime, nil
		}
	}

	session, err := h.appSessionService.GetByTokenID(ctx, tokenID)
	if err != nil {
		return nil, err
	}
	runtime := &casCollectRuntime{
		session:   session,
		panClient: buildPanClient(session),
	}
	h.casCollectRuntimeCache.Store(cacheKey, runtime)
	ctx.Info("CAS自动归集运行时缓存已建立",
		zap.Int64("tokenId", tokenID),
		zap.Bool("hasSession", session != nil),
	)
	return runtime, nil
}

type casCollectRuntime struct {
	session   *appsession.Session
	panClient *cloudpan.PanClient
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

func waitForShareSaveTask(ctx context.Context, accessToken, taskID string, maxWait time.Duration) error {
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
		ctx.Info("CAS自动归集轮询SHARE_SAVE任务(accessToken直连)",
			zap.String("taskId", taskID),
			zap.Any("resCode", resp.ResCode),
			zap.String("resMessage", resp.ResMessage),
			zap.Int("taskStatus", resp.TaskStatus),
			zap.Int("failedCount", resp.FailedCount),
		)
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

func (h *handler) collectSubscribeShareCAS(ctx context.Context, runtime *casCollectRuntime, panClient *cloudpan.PanClient, targetFolderID string, file *models.VirtualFile) error {
	shareID, ok := file.Addition.Int64(consts.FileAdditionKeyShareId)
	if !ok || shareID <= 0 {
		return fmt.Errorf("自动归集CAS失败: 缺少订阅分享ID")
	}
	fileID := strings.TrimSpace(file.CloudId)
	if fileID == "" {
		return fmt.Errorf("自动归集CAS失败: 缺少订阅文件ID")
	}

	if runtime == nil || runtime.session == nil {
		return fmt.Errorf("自动归集CAS失败: 无法获取目标运行时会话")
	}
	// 注意：这里的“目录缓存去重”只允许对“开启了自动刷新的源存储”生效，不能扩成全局逻辑。
	if topFile, topErr := h.virtualFileService.QueryTop(ctx, file.ID); topErr == nil && topFile != nil {
		if mountPoint, mpErr := h.mountPointService.Query(ctx, topFile.ID); mpErr == nil && mountPoint != nil && mountPoint.EnableAutoRefresh {
			if err := h.refreshCASDirCacheIfNeeded(ctx, shared.SettingAddition.CasPersonTargetTokenId, targetFolderID, runtime); err != nil {
				ctx.Warn("刷新CAS目标目录缓存失败，继续尝试转存", zap.Error(err), zap.String("targetFolderId", targetFolderID))
			} else if exists, err := h.casTargetCacheService.Exists(ctx, shared.SettingAddition.CasPersonTargetTokenId, targetFolderID, file.Name); err == nil && exists {
				ctx.Info("CAS自动归集命中本地目录缓存，跳过已存在文件",
					zap.String("fileName", file.Name),
					zap.String("targetFolderId", targetFolderID),
				)
				return nil
			}
		}
	}
	accessToken := strings.TrimSpace(runtime.session.Token.AccessToken)
	if accessToken == "" {
		return fmt.Errorf("自动归集CAS失败: 目标运行时缺少AccessToken")
	}
	ctx.Info("CAS自动归集准备提交SHARE_SAVE任务(accessToken直提)",
		zap.String("fileName", file.Name),
		zap.String("fileId", fileID),
		zap.Int64("shareId", shareID),
		zap.String("targetFolderId", targetFolderID),
	)
	resp := new(batchTaskCreateResp)
	if err := doAccessTokenFormJSONRequest(accessToken, "https://api.cloud.189.cn/open/batch/createBatchTask.action", map[string]string{
		"type":           "SHARE_SAVE",
		"taskInfos":      fmt.Sprintf(`[{"fileId":"%s","fileName":"%s","isFolder":0}]`, fileID, strings.ReplaceAll(file.Name, `"`, `\"`)),
		"targetFolderId": targetFolderID,
		"shareId":        fmt.Sprintf("%d", shareID),
	}, 30*time.Second, resp); err != nil {
		return fmt.Errorf("自动归集CAS失败: 提交SHARE_SAVE任务失败: %w", err)
	}
	ctx.Info("CAS自动归集提交SHARE_SAVE任务返回(accessToken直提)",
		zap.String("fileName", file.Name),
		zap.Any("resCode", resp.ResCode),
		zap.String("resMessage", resp.ResMessage),
		zap.String("taskId", resp.TaskID),
	)
	if batchRespError(resp.ResCode) {
		return fmt.Errorf("自动归集CAS失败: 提交SHARE_SAVE任务失败: %s", resp.ResMessage)
	}
	if strings.TrimSpace(resp.TaskID) == "" {
		return fmt.Errorf("自动归集CAS失败: SHARE_SAVE未返回任务ID")
	}
	if err := waitForShareSaveTask(ctx, accessToken, resp.TaskID, 2*time.Minute); err != nil {
		return err
	}
	if h.casTargetCacheService != nil {
		if topFile, topErr := h.virtualFileService.QueryTop(ctx, file.ID); topErr == nil && topFile != nil {
			if mountPoint, mpErr := h.mountPointService.Query(ctx, topFile.ID); mpErr == nil && mountPoint != nil && mountPoint.EnableAutoRefresh {
				_ = h.casTargetCacheService.Upsert(ctx, &models.CasTargetDirCache{
					TargetTokenID:  shared.SettingAddition.CasPersonTargetTokenId,
					TargetFolderID: targetFolderID,
					FileName:       file.Name,
					IsDir:          false,
					RefreshedAt:    time.Now(),
				})
			}
		}
	}
	return nil
}

func (h *handler) tryCollectCASFromVirtualFile(ctx context.Context, file *models.VirtualFile) error {
	return h.tryCollectCASFromVirtualFileWithRetry(ctx, file, 0)
}

func (h *handler) tryCollectCASFromVirtualFileWithRetry(ctx context.Context, file *models.VirtualFile, retryCount int) error {
	cfg := shared.SettingAddition
	if !cfg.CasTargetEnabled || !cfg.CasAutoCollectEnabled {
		return nil
	}
	if cfg.CasPersonTargetTokenId <= 0 {
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

	ctx.Info("CAS自动归集开始获取目标运行时",
		zap.Int64("tokenId", cfg.CasPersonTargetTokenId),
		zap.String("fileName", file.Name),
	)
	runtime, err := h.getOrCreateCASCollectRuntime(ctx, cfg.CasPersonTargetTokenId)
	if err != nil {
		return fmt.Errorf("获取CAS目标运行时失败: %w", err)
	}
	if runtime.session == nil {
		return fmt.Errorf("获取CAS目标运行时失败: session为空")
	}
	ctx.Info("CAS自动归集已获取目标运行时",
		zap.Int64("tokenId", cfg.CasPersonTargetTokenId),
		zap.Bool("hasSessionKey", strings.TrimSpace(runtime.session.Token.SessionKey) != ""),
		zap.Bool("hasFamilySessionKey", strings.TrimSpace(runtime.session.Token.FamilySessionKey) != ""),
		zap.Bool("hasAccessToken", strings.TrimSpace(runtime.session.Token.AccessToken) != ""),
	)
	panClient := runtime.panClient
	if panClient == nil {
		return fmt.Errorf("创建CAS目标PanClient失败")
	}

	targetFolderID := cfg.CasPersonTargetFolderId
	if targetFolderID == "" {
		targetFolderID = "-11"
	}

	if cfg.CasAutoCollectPreservePath {
		var sourceDirPath string
		if runtimePath, ok := file.Addition.String(consts.FileAdditionKeySourceDirPath); ok && strings.TrimSpace(runtimePath) != "" {
			sourceDirPath = strings.TrimSpace(runtimePath)
		}
		if sourceDirPath == "" && file.ParentId > 0 {
			if parentFullPath, parentErr := h.virtualFileService.CalFullPath(ctx, file.ParentId); parentErr == nil {
				sourceDirPath = strings.TrimSpace(parentFullPath)
			}
		}
		if sourceDirPath == "" {
			fullPath, fullPathErr := h.virtualFileService.CalFullPath(ctx, file.ID)
			if fullPathErr == nil {
				sourceDirPath = strings.TrimSpace(path.Dir(fullPath))
			}
		}
		relDir := strings.Trim(strings.TrimPrefix(sourceDirPath, "/"), " ")
		if relDir != "" && relDir != "." {
			ctx.Info("CAS自动归集准备创建归集目录",
				zap.String("sourceDirPath", sourceDirPath),
				zap.String("relativeDir", relDir),
				zap.String("baseTargetFolderId", targetFolderID),
			)
			folder, apiErr := panClient.AppMkdirRecursive(0, targetFolderID, relDir, 0, strings.Split(relDir, "/"))
			if apiErr != nil {
				return fmt.Errorf("创建CAS归集目录失败: %w", apiErr)
			}
			// 关键约束：只要要求保留路径，就必须先拿到最终目标目录 ID。
			// 不能在目录未确认成功时继续提交文件转存，否则文件会落到上一级甚至根目录。
			if folder == nil || strings.TrimSpace(folder.FileId) == "" {
				return fmt.Errorf("创建CAS归集目录失败: 未返回最终目标目录ID relativeDir=%s", relDir)
			}
			targetFolderID = strings.TrimSpace(folder.FileId)
			ctx.Info("CAS自动归集目录创建/复用成功",
				zap.String("relativeDir", relDir),
				zap.String("targetFolderId", targetFolderID),
			)
			ctx.Info("CAS自动归集确认最终目标目录",
				zap.String("fileName", file.Name),
				zap.String("finalTargetFolderId", targetFolderID),
				zap.String("relativeDir", relDir),
			)
		} else {
			ctx.Info("CAS自动归集未生成相对目录，回退保存到基目录",
				zap.Int64("fileId", file.ID),
				zap.Int64("parentId", file.ParentId),
				zap.String("fileName", file.Name),
			)
		}
	}

	switch file.OsType {
	case models.OsTypeSubscribeShareFile:
		if err := h.collectSubscribeShareCAS(ctx, runtime, panClient, targetFolderID, file); err != nil {
			// 这里是“订阅 .cas 自动转存”的 SHARE_SAVE 主链。
			// 失败后不写缓存，并且只安排一次“5 分钟后重新发起新的 SHARE_SAVE”延迟重试。
			h.scheduleRetryCASCollect(ctx, file, retryCount, err)
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
