package taskengine

import "errors"

var (
	ErrEngineAlreadyRunning = errors.New("task engine already running")
	ErrEngineNotRunning     = errors.New("task engine not running")
	ErrBufferFull           = errors.New("task engine buffer full")
	ErrProcessorNotFound    = errors.New("no processor found for topic")
	ErrInvalidOptions       = errors.New("invalid options")
)
