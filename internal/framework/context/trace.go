package context

import (
	"sync"
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/consts"
)

var _ Trace = (*trace)(nil)

type Trace interface {
	ID() string
	WithRequest(req *Request) Trace
	WithResponse(resp *Response) Trace
	AppendSQL(sql *SQL) Trace
	AppendThirdPartyRequest(req *ThirdPartyRequest) Trace
}

func newTrace(id string) Trace {
	if id == "" {
		id = generateUniqueID()
	}

	return &trace{
		TraceId: id,
	}
}

type trace struct {
	mu                 sync.Mutex
	TraceId            string               `json:"trace_id"`             // 链路ID
	Request            *Request             `json:"request,omitempty"`    // 请求信息
	Response           *Response            `json:"response,omitempty"`   // 返回信息
	SQLs               []*SQL               `json:"sqls"`                 // 执行的 SQL 信息
	ThirdPartyRequests []*ThirdPartyRequest `json:"third_party_requests"` // 第三方请求信息
	Messages           []*Message           `json:"messages"`
	Success            bool                 `json:"success"`      // 请求结果 true or false
	CostSeconds        float64              `json:"cost_seconds"` // 执行时长(单位秒)
}

// Request 请求信息
type Request struct {
	TTL        string      `json:"ttl"`         // 请求超时时间
	Method     string      `json:"method"`      // 请求方式
	DecodedURL string      `json:"decoded_url"` // 请求地址
	Header     interface{} `json:"header"`      // 请求 Header 信息
	Body       interface{} `json:"body"`        // 请求 Body 信息
}

// Response 响应信息
type Response struct {
	Header      interface{} `json:"header"`        // Header 信息
	Body        interface{} `json:"body"`          // Body 信息
	HttpCode    int         `json:"http_code"`     // HTTP 状态码
	HttpCodeMsg string      `json:"http_code_msg"` // HTTP 状态码信息
	CostSeconds float64     `json:"cost_seconds"`  // 执行时间(单位秒)
}
type SQL struct {
	Time        string  `json:"time"`          // 时间，格式：2006-01-02 15:04:05
	Stack       string  `json:"stack"`         // 文件地址和行号
	SQL         string  `json:"sql"`           // SQL 语句
	Rows        int64   `json:"rows_affected"` // 影响行数
	CostSeconds float64 `json:"cost_seconds"`  // 执行时长(单位秒)
}

// ThirdPartyRequest 第三方请求信息
type ThirdPartyRequest struct {
	Time         string      `json:"time"`          // 时间，格式：2006-01-02 15:04:05
	Stack        string      `json:"stack"`         // 文件地址和行号
	Method       string      `json:"method"`        // 请求方法
	URL          string      `json:"url"`           // 请求URL
	RequestBody  interface{} `json:"request_body"`  // 请求体
	ResponseBody interface{} `json:"response_body"` // 响应体
	StatusCode   int         `json:"status_code"`   // HTTP状态码
	StatusMsg    string      `json:"status_msg"`    // HTTP状态信息
	CostSeconds  float64     `json:"cost_seconds"`  // 执行时长(单位秒)
}

type Message struct {
	Time    string `json:"time"`
	Content string `json:"content"`
}

// ID 唯一标识符
func (t *trace) ID() string {
	return t.TraceId
}

// WithRequest 设置 request 信息
func (t *trace) WithRequest(req *Request) Trace {
	t.Request = req
	return t
}

// WithResponse 设置 response 信息
func (t *trace) WithResponse(resp *Response) Trace {
	t.Response = resp
	return t
}

// AppendSQL 追加 SQL 执行日志
func (t *trace) AppendSQL(sql *SQL) Trace {
	if sql == nil {
		return t
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.SQLs = append(t.SQLs, sql)

	return t
}

// AppendThirdPartyRequest 追加第三方请求日志
func (t *trace) AppendThirdPartyRequest(req *ThirdPartyRequest) Trace {
	if req == nil {
		return t
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.ThirdPartyRequests = append(t.ThirdPartyRequests, req)

	return t
}

func (t *trace) AppendMessage(content string) Trace {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Messages = append(t.Messages, &Message{
		Time:    time.Now().Format(consts.TimeFormat),
		Content: content,
	})

	return t
}
