package taskstate

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
)

// CleanupFileLogs 触发任务日志清理（按环境变量保留天数）
// @Summary 触发任务日志清理
// @Description 按 TASKLOG_RETENTION_DAYS（默认15）清理早于该天数的任务日志
// @Tags 任务状态管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} httpcontext.Response "清理触发完成"
// @Failure 400 {object} httpcontext.Response "清理失败"
// @Router /api/task_state/file_log/cleanup [post]
func (h *handler) CleanupFileLogs() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		retention := 15
		if v := os.Getenv("TASKLOG_RETENTION_DAYS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				retention = n
			}
		}
		before := time.Now().Add(-time.Duration(retention) * 24 * time.Hour)
		deleted, err := h.fileTaskLogService.CleanupOlderThan(ctx.GetContext(), before)
		if err != nil {
			ctx.Response(http.StatusBadRequest, http.StatusBadRequest, "cleanup failed", map[string]any{"error": err.Error()})
			return
		}
		ctx.Success(map[string]any{"deleted": deleted, "retentionDays": retention})
	}
}
