package scheduler

import (
	errors2 "errors"
	stdContext "context"

	"github.com/pkg/errors"
	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	autoingestlogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/autoingestlog"
	autoingestplanSvi "github.com/xxcheng123/cloudpan189-share/internal/services/autoingestplan"
	appsessionSvi "github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	casrecordSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrecord"
	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	filetasklogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/filetasklog"
	loginlogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/loginlog"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	virtualfileSvi "github.com/xxcheng123/cloudpan189-share/internal/services/virtualfile"
)

type Scheduler interface {
	Start(ctx context.Context) error
	Stop()
}

var (
	ErrSchedulerRunning = errors.New("scheduler is running")
)

func Start(svc bootstrap.ServiceContext) (func(), error) {
	const handlerName = "scheduler"

	var (
		logger = svc.GetLogger(handlerName)
		err    []error
		ctx    = context.NewContext(stdContext.Background(), context.WithLogger(logger))
	)

	cloudTokenService := cloudtokenSvi.NewService(svc)
	cloudBridgeService := cloudbridgeSvi.NewService(svc)
	fileTaskLogService := filetasklogSvi.NewService(svc)
	mountPointService := mountpointSvi.NewService(svc, cloudTokenService, cloudBridgeService)
	autoIngestPlanService := autoingestplanSvi.NewService(svc)
	autoIngestLogService := autoingestlogSvi.NewService(svc)
	virtualFileService := virtualfileSvi.NewService(svc)
	loginLogService := loginlogSvi.NewService(svc)
	casRecordService := casrecordSvi.NewService(svc)
	appSessionService := appsessionSvi.NewService(svc, cloudTokenService, mountPointService)
	taskEngine := svc.GetTaskEngine()

	fileTaskLogCheckScheduler := NewFileTaskLogCheckScheduler(fileTaskLogService)
	if e := fileTaskLogCheckScheduler.Start(ctx); e != nil { err = append(err, e) }

	refreshFileScheduler := NewRefreshFileScheduler(mountPointService, fileTaskLogService, virtualFileService, cloudTokenService, taskEngine)
	if e := refreshFileScheduler.Start(ctx); e != nil { err = append(err, e) }

	autoIngestRefreshScheduler := NewAutoIngestRefreshScheduler(taskEngine, autoIngestPlanService, autoIngestLogService)
	if e := autoIngestRefreshScheduler.Start(ctx); e != nil { err = append(err, e) }

	vacuumScheduler := NewVacuumScheduler(svc)
	if e := vacuumScheduler.Start(ctx); e != nil { err = append(err, e) }

	memTrimScheduler := NewMemTrimScheduler()
	if e := memTrimScheduler.Start(ctx); e != nil { err = append(err, e) }

	refreshCloudTokenScheduler := NewRefreshCloudTokenScheduler(cloudTokenService)
	if e := refreshCloudTokenScheduler.Start(ctx); e != nil { err = append(err, e) }

	cleanupTaskLogScheduler := NewCleanupTaskLogScheduler(fileTaskLogService)
	if e := cleanupTaskLogScheduler.Start(ctx); e != nil { err = append(err, e) }

	cleanupLoginLogScheduler := NewCleanupLoginLogScheduler(loginLogService)
	if e := cleanupLoginLogScheduler.Start(ctx); e != nil { err = append(err, e) }

	recycleRestoredCASScheduler := NewRecycleRestoredCASScheduler(casRecordService, appSessionService, mountPointService, cloudTokenService)
	if e := recycleRestoredCASScheduler.Start(ctx); e != nil { err = append(err, e) }

	schedulers := []Scheduler{
		fileTaskLogCheckScheduler,
		refreshFileScheduler,
		autoIngestRefreshScheduler,
		vacuumScheduler,
		memTrimScheduler,
		refreshCloudTokenScheduler,
		cleanupTaskLogScheduler,
		cleanupLoginLogScheduler,
		recycleRestoredCASScheduler,
	}

	return closeBar(schedulers), errors2.Join(err...)
}

func closeBar(schedulers []Scheduler) func() {
	return func() {
		for _, scheduler := range schedulers {
			scheduler.Stop()
		}
	}
}
