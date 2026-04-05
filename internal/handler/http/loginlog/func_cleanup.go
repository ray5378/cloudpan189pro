package loginlog

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
)

// Cleanup 登录日志清理（按 LOGINLOG_RETENTION_DAYS，默认15天）
func (h *handler) Cleanup() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		ret := 15
		if v := os.Getenv("LOGINLOG_RETENTION_DAYS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 { ret = n }
		}
		before := time.Now().Add(-time.Duration(ret) * 24 * time.Hour)
		deleted, err := h.loginLogService.CleanupOlderThan(ctx.GetContext(), before)
		if err != nil {
			ctx.Response(http.StatusBadRequest, http.StatusBadRequest, "cleanup failed", map[string]any{"error": err.Error()})
			return
		}
		ctx.Success(map[string]any{"deleted": deleted, "retentionDays": ret})
	}
}
