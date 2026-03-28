package external

import (
	httpctx "github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/taskengine"
	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	filetasklogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/filetasklog"
	settingSvi "github.com/xxcheng123/cloudpan189-share/internal/services/setting"
	storagefacadeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/storagefacade"
)

type Handler interface {
	CreateStorage() httpctx.HandlerFunc
}

type handler struct {
	cloudBridgeService   cloudbridgeSvi.Service
	storageFacadeService storagefacadeSvi.Service
	settingService       settingSvi.Service
	fileTaskLogService   filetasklogSvi.Service
	taskEngine           taskengine.TaskEngine
}

func NewHandler(cb cloudbridgeSvi.Service, sf storagefacadeSvi.Service, st settingSvi.Service, ftl filetasklogSvi.Service, te taskengine.TaskEngine) Handler {
	return &handler{cloudBridgeService: cb, storageFacadeService: sf, settingService: st, fileTaskLogService: ftl, taskEngine: te}
}
