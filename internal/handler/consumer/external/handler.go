package external

import (
	"encoding/json"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/taskcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/taskengine"
	filetasklogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/filetasklog"
	storagefacadeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/storagefacade"
	"github.com/xxcheng123/cloudpan189-share/internal/types/topic"
)

type Handler interface {
	CreateStorage() taskcontext.HandlerFunc
}

type handler struct {
	storageFacadeService storagefacadeSvi.Service
	fileTaskLogService   filetasklogSvi.Service
	taskEngine           taskengine.TaskEngine
}

func NewHandler(sf storagefacadeSvi.Service, ftl filetasklogSvi.Service, te taskengine.TaskEngine) Handler {
	return &handler{storageFacadeService: sf, fileTaskLogService: ftl, taskEngine: te}
}

type payload struct {
	TaskId int64                                  `json:"taskId"`
	Req    *storagefacadeSvi.CreateStorageRequest `json:"req"`
}

func (h *handler) CreateStorage() taskcontext.HandlerFunc {
	return func(ctx *taskcontext.Context) error {
		var p payload
		if err := ctx.Unmarshal(&p); err != nil {
			return err
		}

		c := ctx.GetContext()

		// 标记 running
		_ = h.fileTaskLogService.Running(c, filetasklogSvi.NewLogID(p.TaskId))

		if p.Req == nil {
			_ = h.fileTaskLogService.WithErrorAndFail(c, filetasklogSvi.NewLogID(p.TaskId),
				&json.SyntaxError{Offset: 0},
			)
			return nil
		}

		// 执行创建
		vfId, err := h.storageFacadeService.CreateStorage(c, p.Req)
		if err != nil {
			_ = h.fileTaskLogService.WithErrorAndFail(c, filetasklogSvi.NewLogID(p.TaskId), err)
			return err
		}

		// 创建成功后，立即触发一次深度刷新
		taskReq := &topic.FileScanFileRequest{
			FileId: vfId,
			Deep:   true,
		}
		taskBody, _ := json.Marshal(taskReq)
		if err := h.taskEngine.PushMessage(c, taskReq.Topic(), taskBody); err != nil {
			_ = h.fileTaskLogService.WithErrorAndFail(c, filetasklogSvi.NewLogID(p.TaskId), err)
			return err
		}

		// completed
		_ = h.fileTaskLogService.Completed(c, filetasklogSvi.NewLogID(p.TaskId))
		return nil
	}
}

// ensure import used
var _ = json.RawMessage{}
var _ = json.SyntaxError{}
