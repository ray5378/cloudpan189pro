package setting

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	filetasklogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/filetasklog"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	virtualfileSvi "github.com/xxcheng123/cloudpan189-share/internal/services/virtualfile"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"github.com/xxcheng123/cloudpan189-share/internal/types/topic"
)

func splitAutoDeleteKeywords(raw string) []string {
	parts := strings.Split(raw, "|")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, strings.ToLower(part))
	}
	return result
}

func matchAutoDeleteKeyword(logItem *models.FileTaskLog, keywords []string) (bool, string) {
	if logItem == nil || len(keywords) == 0 {
		return false, ""
	}
	fields := []string{logItem.Title, logItem.Desc, logItem.ErrorMsg, logItem.Result}
	for _, keyword := range keywords {
		for _, field := range fields {
			if strings.Contains(strings.ToLower(field), keyword) {
				return true, keyword
			}
		}
	}
	return false, ""
}

// RunAutoDeleteInvalidStorageOnce 手动执行一次“自动删除失效存储”逻辑。
func (h *handler) RunAutoDeleteInvalidStorageOnce() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		cfg := shared.SettingAddition
		if !cfg.AutoDeleteInvalidStorageEnabled {
			ctx.Fail(codeModifyAdditionFailed.WithError(fmt.Errorf("自动删除失效存储未启用")))
			return
		}

		mounts, err := h.mountPointService.List(ctx.GetContext(), &mountpointSvi.ListRequest{NoPaginate: true})
		if err != nil {
			ctx.Fail(codeQueryFailed.WithError(err))
			return
		}
		if len(mounts) == 0 {
			ctx.Success(map[string]any{"count": 0, "message": "没有可检测的存储节点"})
			return
		}

		allFileIDs := make([]int64, 0, len(mounts))
		for _, mp := range mounts {
			if mp == nil {
				continue
			}
			allFileIDs = append(allFileIDs, mp.FileId)
		}

		lastLogs, err := h.fileTaskLogService.LatestByFileIDs(ctx.GetContext(), allFileIDs)
		if err != nil {
			ctx.Fail(codeQueryFailed.WithError(err))
			return
		}

		keywords := splitAutoDeleteKeywords(cfg.AutoDeleteInvalidStorageKeywords)
		deleteIDs := make([]int64, 0)
		deleteReasons := make(map[int64]string)
		rule2Candidates := make([]*models.MountPoint, 0)
		for _, mp := range mounts {
			if mp == nil {
				continue
			}
			lastLog := lastLogs[mp.FileId]
			matchedKeyword, matchedKeywordText := matchAutoDeleteKeyword(lastLog, keywords)
			if matchedKeyword {
				deleteIDs = append(deleteIDs, mp.FileId)
				deleteReasons[mp.FileId] = fmt.Sprintf("命中自动删除关键词: %s", matchedKeywordText)
				continue
			}

			if !mp.EnableAutoRefresh || !mp.IsInAutoRefreshPeriod() {
				rule2Candidates = append(rule2Candidates, mp)
			}
		}

		rule2FileIDs := make([]int64, 0, len(rule2Candidates))
		for _, mp := range rule2Candidates {
			if mp == nil {
				continue
			}
			rule2FileIDs = append(rule2FileIDs, mp.FileId)
		}

		fileCountMap := make(map[int64]int64, len(rule2FileIDs))
		if len(rule2FileIDs) > 0 {
			const batchSize = 200
			for start := 0; start < len(rule2FileIDs); start += batchSize {
				end := start + batchSize
				if end > len(rule2FileIDs) {
					end = len(rule2FileIDs)
				}
				batch := rule2FileIDs[start:end]

				counts, err := h.virtualFileService.GroupCountByTopId(ctx.GetContext(), &virtualfileSvi.GroupCountByTopIdRequest{TopIdList: batch})
				if err != nil {
					ctx.Fail(codeQueryFailed.WithError(err))
					return
				}
				for _, item := range counts {
					if item == nil {
						continue
					}
					fileCountMap[item.TopId] = item.Count
				}
			}
		}

		for _, mp := range rule2Candidates {
			if mp == nil {
				continue
			}
			lastLog := lastLogs[mp.FileId]
			latestRefreshSucceeded := lastLog != nil && lastLog.Status == models.StatusCompleted
			if fileCountMap[mp.FileId] == 0 && latestRefreshSucceeded {
				deleteIDs = append(deleteIDs, mp.FileId)
				deleteReasons[mp.FileId] = "未启用自动刷新或已过期，且文件数量为0、最新刷新成功"
			}
		}

		scheduleKey := time.Now().Format("2006-01-02 15:04:05 手动触发")
		if len(deleteIDs) == 0 {
			ctx.Success(map[string]any{"count": 0, "message": "本次未命中可删除节点"})
			return
		}

		for _, fileID := range deleteIDs {
			reason := deleteReasons[fileID]
			tracker, createErr := h.fileTaskLogService.Create(
				ctx.GetContext(),
				"自动删除失效存储",
				"自动删除失效存储",
				filetasklogSvi.WithFile(fileID),
				filetasklogSvi.WithDesc(reason),
			)
			if createErr == nil && tracker != nil {
				_ = h.fileTaskLogService.Completed(ctx.GetContext(), tracker)
			}
		}

		taskReq := &topic.FileBatchDeleteRequest{IDs: deleteIDs}
		body, _ := json.Marshal(taskReq)
		taskCtx := ctx.GetContext().WithValue(consts.CtxKeyInvokeHandlerName, "手动触发自动删除失效存储")
		if err := h.taskEngine.PushMessage(taskCtx, taskReq.Topic(), body); err != nil {
			ctx.Fail(codeModifyAdditionFailed.WithError(err))
			return
		}

		ctx.Success(map[string]any{
			"count":       len(deleteIDs),
			"scheduleKey": scheduleKey,
			"deleteIds":   deleteIDs,
		})
	}
}
