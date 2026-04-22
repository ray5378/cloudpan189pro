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

func (h *handler) collectSubscribeShareCAS(ctx context.Context, panClient *cloudpan.PanClient, targetFolderID string, file *models.VirtualFile) error {
	shareID, ok := file.Addition.Int64(consts.FileAdditionKeyShareId)
	if !ok || shareID <= 0 {
		return fmt.Errorf("自动归集CAS失败: 缺少订阅分享ID")
	}
	if strings.TrimSpace(file.CloudId) == "" {
		return fmt.Errorf("自动归集CAS失败: 缺少订阅文件ID")
	}

	taskID, apiErr := panClient.CreateBatchTask(&cloudpan.BatchTaskParam{
		TypeFlag: cloudpan.BatchTaskTypeShareSave,
		TaskInfos: cloudpan.BatchTaskInfoList{
			&cloudpan.BatchTaskInfo{
				FileId:   strings.TrimSpace(file.CloudId),
				FileName: file.Name,
				IsFolder: 0,
			},
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
