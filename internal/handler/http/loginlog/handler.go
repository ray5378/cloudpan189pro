package loginlog

import (
	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	loginlogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/loginlog"
)

// Handler 定义 login 日志相关的 HTTP 处理器接口
type Handler interface {
	// List 登录日志列表
	List() httpcontext.HandlerFunc
	// Cleanup 登录日志清理
	Cleanup() httpcontext.HandlerFunc
}

var bi = httpcontext.NewBusinessGenerator(consts.BusCodeLoginLogStartCode)

var (
	codeListFailed = bi.Next("获取登录日志列表失败")
)

type handler struct {
	loginLogService loginlogSvi.Service
}

func NewHandler(loginLogService loginlogSvi.Service) Handler {
	return &handler{
		loginLogService: loginLogService,
	}
}
