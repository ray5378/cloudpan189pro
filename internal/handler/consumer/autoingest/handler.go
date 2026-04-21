package autoingest

import (
	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/taskcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/taskengine"

	appsessionSvi "github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	autoingestlogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/autoingestlog"
	autoingestplanSvi "github.com/xxcheng123/cloudpan189-share/internal/services/autoingestplan"
	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	storagefacadeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/storagefacade"
	virtualfileSvi "github.com/xxcheng123/cloudpan189-share/internal/services/virtualfile"
)

type Handler interface {
	RefreshSubscribe() taskcontext.HandlerFunc
}

type handler struct {
	svc                   bootstrap.ServiceContext
	taskEngine            taskengine.TaskEngine
	cloudbridgeService    cloudbridgeSvi.Service
	autoIngestPlanService autoingestplanSvi.Service
	authIngestLogService  autoingestlogSvi.Service
	storageFacadeService  storagefacadeSvi.Service
	virtualFileService    virtualfileSvi.Service
	cloudTokenService     cloudtokenSvi.Service
	mountPointService     mountpointSvi.Service
	appSessionService     appsessionSvi.Service
}

func NewHandler(
	svc bootstrap.ServiceContext,
	taskEngine taskengine.TaskEngine,
	cloudbridgeService cloudbridgeSvi.Service,
	autoIngestPlanService autoingestplanSvi.Service,
	authIngestLogService autoingestlogSvi.Service,
	storageFacadeService storagefacadeSvi.Service,
	virtualFileService virtualfileSvi.Service,
	cloudTokenService cloudtokenSvi.Service,
	mountPointService mountpointSvi.Service,
) Handler {
	return &handler{
		svc:                   svc,
		taskEngine:            taskEngine,
		cloudbridgeService:    cloudbridgeService,
		autoIngestPlanService: autoIngestPlanService,
		authIngestLogService:  authIngestLogService,
		storageFacadeService:  storageFacadeService,
		virtualFileService:    virtualFileService,
		cloudTokenService:     cloudTokenService,
		mountPointService:     mountPointService,
		appSessionService:     appsessionSvi.NewService(svc, cloudTokenService, mountPointService),
	}
}
