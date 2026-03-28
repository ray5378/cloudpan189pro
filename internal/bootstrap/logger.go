package bootstrap

import (
	"os"
	"runtime"

	"github.com/xxcheng123/cloudpan189-share/internal/configs"
	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	logger2 "github.com/xxcheng123/cloudpan189-share/internal/pkgs/logger"
	"go.uber.org/zap"
)

func initLogger(c *configs.RuntimeConfig) (logger *zap.Logger, err error) {
	var options = []logger2.Option{
		logger2.WithTimeLayout(consts.TimeFormat),
		logger2.WithFileRotationP(c.LogFile),
		logger2.WithOutputInConsole(),
	}

	if logLevel := os.Getenv(consts.EnvKeyLogLevel); logLevel != "" {
		switch logLevel {
		case zap.DebugLevel.String():
			options = append(options, logger2.WithDebugLevel())
		case zap.WarnLevel.String():
			options = append(options, logger2.WithWarnLevel())
		case zap.ErrorLevel.String():
			options = append(options, logger2.WithErrorLevel())
		default:
			options = append(options, logger2.WithInfoLevel())
		}
	}

	logger, err = logger2.NewJSONLogger(options...)
	if err != nil {
		return nil, err
	}

	// 记录二进制构建信息
	logger.Info("binary build info",
		zap.String("build date", c.BuildDate),
		zap.String("go_version", runtime.Version()),
		zap.String("git_commit", c.Commit),
		zap.String("git_branch", c.GitBranch),
		zap.String("git_summary", c.GitSummary),
		zap.String("version", c.Version),
		zap.String("log_level", logger.Level().String()),
	)

	if c.Version != "" {
		logger = logger.With(zap.String("ver", c.Version))
	}

	return logger, nil
}
