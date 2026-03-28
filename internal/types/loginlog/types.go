package loginlog

type Event string

// 事件类型
const (
	EventLogin        Event = "login"
	EventRefreshToken Event = "refresh_token"
)

type Method string

// 事件来源（方法/渠道）
const (
	MethodWeb Method = "web"
	MethodAPI Method = "api"
	MethodApp Method = "app"
	MethodCLI Method = "cli"
)

type Status string

// 状态
const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
	StatusBlocked Status = "blocked"
)
