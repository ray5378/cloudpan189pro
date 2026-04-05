package main

import (
	"fmt"
	stdhttp "net/http"
	_ "net/http/pprof"

	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/configs"
	"github.com/xxcheng123/cloudpan189-share/internal/handler/consumer"
	"github.com/xxcheng123/cloudpan189-share/internal/handler/dav"
	"github.com/xxcheng123/cloudpan189-share/internal/handler/http"
	"github.com/xxcheng123/cloudpan189-share/internal/handler/scheduler"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/httpx"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/shutdown"
	"go.uber.org/zap"
)

func main() {
	svc, err := bootstrap.New(configs.Get())
	if err != nil {
		panic(err)
	}

	defer svc.Close()

	logger := svc.GetLogger("main")
	taskEngine := svc.GetTaskEngine()
	httpEngine := svc.GetHTTPEngine()
	port := svc.GetPort()

	if err = consumer.Start(svc); err != nil {
		panic(err)
	}

	var (
		closeBar func()
	)

	// 启动scheduler
	if closeBar, err = scheduler.Start(svc); err != nil {
		panic(err)
	}

	http.Start(svc)
	dav.Start(svc)

	go func() {
		// pprof on localhost (disable by PPROF_DISABLE=1)
		if getenvDefault("PPROF_DISABLE", "0") != "1" {
			go func() { _ = stdhttp.ListenAndServe("127.0.0.1:6060", nil) }()
		}
		// replace gin.Engine.Run with custom http.Server (long timeouts, idle GC)
		server := httpx.NewServer(httpEngine, fmt.Sprintf(":%d", port))
		if err = server.ListenAndServe(); err != nil && err != stdhttp.ErrServerClosed {
			panic(err)
		}
	}()

	logger.Info("system running....")

	shutdown.Close(func() {
		logger.Info("close shutdown")

		if err = taskEngine.Stop(); err != nil {
			logger.Error("close task engine failed", zap.Error(err))
		}

		closeBar()

		logger.Info("close successful, bye~")
	})
}
