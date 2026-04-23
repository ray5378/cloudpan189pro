package casrestore

import (
	stdctx "context"
	"sync"

	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casrecord"
	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
)

// UploadRoute 表示秒传/上传时优先走哪条云盘路线。
// 注意：它描述的是“恢复链如何写入云端”，不是最终目录归属。
type UploadRoute string

const (
	// UploadRouteFamily 表示优先走家庭云秒传链路。是默认值。
	UploadRouteFamily UploadRoute = "family"
	// UploadRoutePerson 表示优先走个人云秒传链路。
	UploadRoutePerson UploadRoute = "person"
)

// DestinationType 表示恢复完成后文件最终位于哪类目录。
// 注意：它描述的是“最终目录归属”，不是秒传/上传路线。
type DestinationType string

const (
	// DestinationTypePerson 表示最终目录属于个人云。
	DestinationTypePerson DestinationType = "person"
	// DestinationTypeFamily 表示最终目录属于家庭云。
	DestinationTypeFamily DestinationType = "family"
)

// RestoreRequest 是 CAS 恢复请求。
// 这里有两个容易混淆的维度，必须同时保留：
// 1. UploadRoute: 秒传/上传时走哪条路线（默认 family）
// 2. DestinationType: 文件最终落在哪类目录（person/family）
//
// 注意：产品语义可以表达四种组合，但真正允许进入执行层的组合仍必须有 reference-backed 主链支撑。
// 当前 person -> family 仍无可直接照搬的参考链，因此应视为 unsupported，而不是拿 SDK 等价路径兜底。
type RestoreRequest struct {
	StorageID     int64
	MountPointID  int64
	TargetTokenID int64
	CasFileID     string
	CasFileName   string
	CasVirtualID  int64
	LocalCasPath  string

	// UploadRoute 决定秒传/上传时优先走哪条链路；默认 family。
	UploadRoute UploadRoute
	// DestinationType 决定最终目录属于个人云还是家庭云。
	DestinationType DestinationType
	// TargetFolderID 是最终目录 ID；它只表达目录，不表达路线。
	TargetFolderID string
	// FamilyID 在 family 路线/目标下可选传入，用于避免重复获取家庭列表。
	FamilyID int64
}

// RestoreResult 是恢复完成后的结果。
type RestoreResult struct {
	RestoredFileID   string
	RestoredFileName string
	TargetFolderID   string
	UploadRoute      UploadRoute
	DestinationType  DestinationType
	FamilyID         int64
	CasInfo          *casparser.CasInfo
}

type Service interface {
	EnsureRestored(ctx appctx.Context, req RestoreRequest) (*RestoreResult, error)
	EnsureRestoredFromLocalCAS(ctx appctx.Context, req RestoreRequest) (*RestoreResult, error)
}

type service struct {
	svc                bootstrap.ServiceContext
	recordSvc          casrecord.Service
	appSessionService  appsession.Service
	cloudBridgeService cloudbridgeSvi.Service
	inflightMu         sync.Mutex
	inflight           map[string]*restoreCall
}

type restoreCall struct {
	wg     sync.WaitGroup
	result *RestoreResult
	terr   error
}

var familyIDSessionHint = map[*appsession.Session]int64{}

func reqFamilyIDFromContext(session *appsession.Session) int64 {
	if session == nil {
		return 0
	}
	return familyIDSessionHint[session]
}

func NewService(svc bootstrap.ServiceContext) Service {
	cloudTokenSvc := cloudtokenSvi.NewService(svc)
	cloudBridgeSvc := cloudbridgeSvi.NewService(svc)
	mountPointSvc := mountpointSvi.NewService(svc, cloudTokenSvc, cloudBridgeSvc)
	return &service{
		svc:                svc,
		recordSvc:          casrecord.NewService(svc),
		appSessionService:  appsession.NewService(svc, cloudTokenSvc, mountPointSvc),
		cloudBridgeService: cloudBridgeSvc,
		inflight:           make(map[string]*restoreCall),
	}
}

func inflightKey(req RestoreRequest) string {
	return req.CasFileID + "::" + string(req.UploadRoute) + "::" + string(req.DestinationType) + "::" + req.TargetFolderID
}

func (s *service) withInflight(_ stdctx.Context, key string, fn func() (*RestoreResult, error)) (*RestoreResult, error) {
	s.inflightMu.Lock()
	if call, ok := s.inflight[key]; ok {
		s.inflightMu.Unlock()
		call.wg.Wait()
		return call.result, call.terr
	}
	call := &restoreCall{}
	call.wg.Add(1)
	s.inflight[key] = call
	s.inflightMu.Unlock()

	defer func() {
		s.inflightMu.Lock()
		delete(s.inflight, key)
		s.inflightMu.Unlock()
		call.wg.Done()
	}()

	call.result, call.terr = fn()
	return call.result, call.terr
}
