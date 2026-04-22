package media

import (
	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/taskengine"
	appsessionSvi "github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	casrecordSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrecord"
	casrestoreSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrestore"
	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	mediaconfigSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mediaconfig"
	mediafileSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mediafile"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	settingSvi "github.com/xxcheng123/cloudpan189-share/internal/services/setting"
	verifySvi "github.com/xxcheng123/cloudpan189-share/internal/services/verify"
	virtualfileSvi "github.com/xxcheng123/cloudpan189-share/internal/services/virtualfile"
)

// Handler 定义 media 相关的 HTTP 处理器接口
type Handler interface {
	ConfigInit() httpcontext.HandlerFunc
	ConfigInfo() httpcontext.HandlerFunc
	ConfigUpdate() httpcontext.HandlerFunc
	ConfigToggle() httpcontext.HandlerFunc
	Clear() httpcontext.HandlerFunc
	RebuildStrmFile() httpcontext.HandlerFunc
	RestoreCas() httpcontext.HandlerFunc
	PlayCas() httpcontext.HandlerFunc
}

var bi = httpcontext.NewBusinessGenerator(consts.BusCodeMediaStartCode)

var (
	codeConfigQueryFailed  = bi.Next("查询媒体配置失败")
	codeConfigInitFailed   = bi.Next("初始化媒体配置失败")
	codeConfigUpdateFailed = bi.Next("更新媒体配置失败")
	codeConfigToggleFailed = bi.Next("切换媒体配置启用状态失败")

	codeMediaNotEnabled  = bi.Next("媒体功能未启用")
	codeClearFailed      = bi.Next("清理媒体文件失败")
	codeRebuildFailed    = bi.Next("重建strm文件失败")
	codeRestoreCasFailed = bi.Next("恢复CAS文件失败")
)

type handler struct {
	mediaConfigService mediaconfigSvi.Service
	mediaFileService   mediafileSvi.Service
	mountpointService  mountpointSvi.Service
	virtualfileService virtualfileSvi.Service
	verifyService      verifySvi.Service
	casRestoreService  casrestoreSvi.Service
	casRecordService   casrecordSvi.Service
	cloudTokenService  cloudtokenSvi.Service
	cloudBridgeService cloudbridgeSvi.Service
	appSessionService  appsessionSvi.Service
	settingService     settingSvi.Service
	taskEngine         taskengine.TaskEngine
}

func NewHandler(
	mediaConfigService mediaconfigSvi.Service,
	mediaFileService mediafileSvi.Service,
	mountpointService mountpointSvi.Service,
	virtualfileService virtualfileSvi.Service,
	verifyService verifySvi.Service,
	casRestoreService casrestoreSvi.Service,
	casRecordService casrecordSvi.Service,
	cloudTokenService cloudtokenSvi.Service,
	cloudBridgeService cloudbridgeSvi.Service,
	appSessionService appsessionSvi.Service,
	settingService settingSvi.Service,
	taskEngine taskengine.TaskEngine,
) Handler {
	return &handler{
		mediaConfigService: mediaConfigService,
		mediaFileService:   mediaFileService,
		mountpointService:  mountpointService,
		virtualfileService: virtualfileService,
		verifyService:      verifyService,
		casRestoreService:  casRestoreService,
		casRecordService:   casRecordService,
		cloudTokenService:  cloudTokenService,
		cloudBridgeService: cloudBridgeService,
		appSessionService:  appSessionService,
		settingService:     settingService,
		taskEngine:         taskEngine,
	}
}
