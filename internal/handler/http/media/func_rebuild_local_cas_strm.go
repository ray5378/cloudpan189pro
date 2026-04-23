package media

import "github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"

func (h *handler) RebuildLocalCASSTRM() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		if h.localSTRMService == nil {
			ctx.Fail(codeRebuildFailed)
			return
		}
		result, err := h.localSTRMService.ScanAndEnsureAll(ctx.GetContext())
		if err != nil {
			ctx.Fail(codeRebuildFailed.WithError(err))
			return
		}
		ctx.Success(result)
	}
}
