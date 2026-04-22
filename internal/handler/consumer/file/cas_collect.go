package file

import (
	"fmt"
	"path"
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

func (h *handler) collectSubscribeShareCAS(ctx context.Context, panClient *cloudpan.PanClient, targetFolderID string, file *models.VirtualFile) error {
	shareID, ok := file.Addition.Int64(consts.FileAdditionKeyShareId)
	if !ok || shareID <= 0 {
		return fmt.Errorf("自动归集CAS失败: 缺少订阅分享ID")
	}
	fileID := strings.TrimSpace(file.CloudId)
	if fileID == "" {
		return fmt.Errorf("自动归集CAS失败: 缺少订阅文件ID")
	}

	if panClient == nil {
		return fmt.Errorf("自动归集CAS失败: 无法获取PanClient")
	}
	ctx.Info("CAS自动归集准备提交SHARE_SAVE任务(panClient)",
		zap.String("fileName", file.Name),
		zap.String("fileId", fileID),
		zap.Int64("shareId", shareID),
		zap.String("targetFolderId", targetFolderID),
	)
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
	ctx.Info("CAS自动归集提交SHARE_SAVE任务返回(panClient)",
		zap.String("fileName", file.Name),
		zap.String("taskId", taskID),
	)
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
		ctx.Info("CAS自动归集轮询SHARE_SAVE任务(panClient)",
			zap.String("taskId", taskID),
			zap.Int("taskStatus", int(result.TaskStatus)),
			zap.Int("failedCount", result.FailedCount),
		)
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

	ctx.Info("CAS自动归集开始获取目标运行时",
		zap.Int64("tokenId", cfg.CasTargetTokenId),
		zap.String("fileName", file.Name),
	)
	runtime, err := h.getOrCreateCASCollectRuntime(ctx, cfg.CasTargetTokenId)
	if err != nil {
		return fmt.Errorf("获取CAS目标运行时失败: %w", err)
	}
	if runtime.session == nil {
		return fmt.Errorf("获取CAS目标运行时失败: session为空")
	}
	ctx.Info("CAS自动归集已获取目标运行时",
		zap.Int64("tokenId", cfg.CasTargetTokenId),
		zap.Bool("hasSessionKey", strings.TrimSpace(runtime.session.Token.SessionKey) != ""),
		zap.Bool("hasFamilySessionKey", strings.TrimSpace(runtime.session.Token.FamilySessionKey) != ""),
		zap.Bool("hasAccessToken", strings.TrimSpace(runtime.session.Token.AccessToken) != ""),
	)
	panClient := runtime.panClient
	if panClient == nil {
		return fmt.Errorf("创建CAS目标PanClient失败")
	}

	targetFolderID := cfg.CasTargetFolderId
	if targetFolderID == "" {
		targetFolderID = "-11"
	}

	if cfg.CasAutoCollectPreservePath {
		var sourceDirPath string
		if file.ParentId > 0 {
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
			if folder != nil && folder.FileId != "" {
				targetFolderID = folder.FileId
				ctx.Info("CAS自动归集目录创建/复用成功",
					zap.String("relativeDir", relDir),
					zap.String("targetFolderId", targetFolderID),
				)
			}
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
