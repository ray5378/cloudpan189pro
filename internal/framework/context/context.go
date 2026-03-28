package context

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Context struct {
	ctx context.Context
	*zap.Logger
	Trace
}

type Option struct {
	TraceId string
	Logger  *zap.Logger
}

type OptionFunc func(*Option)

func WithLogger(logger *zap.Logger) OptionFunc {
	return func(c *Option) {
		if logger != nil {
			c.Logger = logger
		}
	}
}

func WithTraceId(traceId string) OptionFunc {
	return func(c *Option) {
		if traceId != "" {
			c.TraceId = traceId
		}
	}
}

func NewContext(ctx context.Context, opts ...OptionFunc) Context {
	option := &Option{
		TraceId: generateUniqueID(),
		Logger:  zap.NewNop(),
	}

	for _, opt := range opts {
		opt(option)
	}

	c := Context{
		ctx:    ctx,
		Trace:  newTrace(option.TraceId),
		Logger: option.Logger.With(zap.String("trace_id", option.TraceId)),
	}

	return c
}

var _ context.Context = new(Context)

func (c Context) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c Context) Err() error {
	return c.ctx.Err()
}

func (c Context) Value(key any) any {
	return c.ctx.Value(key)
}

type CancelFunc = context.CancelFunc

func (c Context) WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	stdCtx, cancel := context.WithCancel(parent.ctx)
	ctx = Context{
		ctx:    stdCtx,
		Trace:  parent.Trace,
		Logger: parent.Logger,
	}

	return ctx, cancel
}

func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	stdCtx, cancel := context.WithCancel(parent.ctx)
	ctx = Context{
		ctx:    stdCtx,
		Trace:  parent.Trace,
		Logger: parent.Logger,
	}

	return ctx, cancel
}

func (c Context) WithValue(k, v any) Context {
	return Context{
		ctx:    context.WithValue(c.ctx, k, v),
		Trace:  c.Trace,
		Logger: c.Logger,
	}
}

func (c Context) GetString(key string) (string, bool) {
	if v := c.Value(key); v != nil {
		if str, ok := v.(string); ok {
			return str, true
		}

		return "", false
	}

	return "", false
}

func (c Context) String(key string, defaultValue string) string {
	if str, ok := c.GetString(key); ok {
		return str
	}

	return defaultValue
}
