package httpcontext

import (
	"github.com/gin-gonic/gin"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"go.uber.org/zap"
)

const (
	traceHeaderKey = "Trace-Id"
)

type Context struct {
	*gin.Context
	stdContext context.Context

	errors        []error
	errMsg        string
	prohibitWrite bool
}

func (c *Context) GetContext() context.Context {
	return c.stdContext
}

func newContext(ginContext *gin.Context, logger *zap.Logger) *Context {
	return &Context{
		Context:    ginContext,
		stdContext: context.NewContext(ginContext, context.WithLogger(logger), context.WithTraceId(ginContext.Request.Header.Get(traceHeaderKey))),
	}
}
