package user

import (
	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/types/loginlog"
)

func (h *handler) RecordLog(eventType loginlog.Event) httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		ctx.Next()

		status := loginlog.StatusSuccess
		reason := ""

		if ctx.IsAborted() {
			status = loginlog.StatusFailed
			reason = ctx.GetErrorMsg()
		}

		log := &models.LoginLog{
			UserId:   ctx.GetInt64(consts.CtxKeyUserId),
			Username: ctx.GetString(consts.CtxKeyUsername),
			Addr:     ctx.ClientIP(),
			// TODO: 获取地理信息
			Location: "-",
			// 目前只有 web
			Method:    loginlog.MethodWeb,
			Event:     eventType,
			Status:    status,
			Reason:    reason,
			UserAgent: ctx.Request.UserAgent(),
			TraceId:   ctx.GetContext().ID(),
		}

		_, _ = h.loginLogService.Create(ctx.GetContext(), log)
	}
}
