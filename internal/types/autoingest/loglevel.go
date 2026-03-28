package autoingest

type LogLevel string

func (l LogLevel) String() string {
	return string(l)
}

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)
