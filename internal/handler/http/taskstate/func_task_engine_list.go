package taskstate

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/taskengine"
)

type (
	taskEngineListResponse struct {
		Stats        taskengine.TaskStats   `json:"stats"`        // 任务引擎统计信息
		RunningTasks []*taskengine.TaskInfo `json:"runningTasks"` // 正在运行的任务列表
		PendingTasks []*taskengine.TaskInfo `json:"pendingTasks"` // 待处理的任务列表
		IsRunning    bool                   `json:"isRunning"`    // 引擎是否正在运行
	}
)

// TaskEngineList 获取任务引擎状态和运行中的任务列表
// @Summary 获取任务引擎状态
// @Description 获取任务引擎的统计信息和当前正在运行的任务列表
// @Tags 任务状态管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} httpcontext.Response{data=taskEngineListResponse} "获取任务引擎状态成功"
// @Failure 400 {object} httpcontext.Response "获取任务引擎状态失败，code=7003"
// @Failure 401 {object} httpcontext.Response "未授权访问"
// @Failure 403 {object} httpcontext.Response "权限不足"
// @Router /api/task_state/task_engine/list [get]
func (h *handler) TaskEngineList() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		// 检查任务引擎是否可用
		if h.taskEngine == nil {
			ctx.Fail(codeGetTaskEngineStatsFailed.WithError(nil))

			return
		}

		// 获取任务引擎统计信息
		stats := h.taskEngine.GetStats()

		// 获取正在运行的任务列表
		runningTasks := h.taskEngine.GetRunningTasks()

		// 获取待处理的任务列表
		pendingTasks := h.taskEngine.GetPendingTasks()

		// 检查引擎是否正在运行
		isRunning := h.taskEngine.IsRunning()

		response := &taskEngineListResponse{
			Stats:        stats,
			RunningTasks: runningTasks,
			PendingTasks: pendingTasks,
			IsRunning:    isRunning,
		}

		ctx.Success(response)
	}
}
