package main

import (
	"github.com/kardianos/service"
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"sync"
)

// Name inspects the project name
var Name = "dohproxy"

// Version inspects the project version
var Version = "custom"

type program struct {
	cliContext *cli.Context
}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) run() {
	servers := LoadServersFromConfig(p.cliContext.String("config"))
	for _, s := range servers {
		go func(s Server) {
			if err := s.Serve(); err != nil {
				zap.L().Fatal("failed to start server", zap.String("type", s.Type()), zap.String("address", s.Address()), zap.Error(err))
			}
		}(s)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	<-make(chan struct{})
	wg.Done()
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	return nil
}

func initLog(stdout, stderr string, level zapcore.Level) {
	zap.NewProductionConfig()
	logger, _ := zap.Config{
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		Level:            zap.NewAtomicLevelAt(level),
		OutputPaths:      []string{stdout},
		ErrorOutputPaths: []string{stderr},
	}.Build()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)
}

func runApp(c *cli.Context) {
	svcConfig := &service.Config{
		Name:        "dohproxy",
		DisplayName: "DOH Proxy",
		Description: "A DNS-over-Https proxy and router",
	}
	if c.IsSet("config") {
		pwd, err := os.Getwd()
		if err != nil {
			zap.L().Fatal("get current working directory failed")
		}
		svcConfig.WorkingDirectory = pwd
		svcConfig.Arguments = []string{"--from-service", "--config", c.String("config")}
	}

	prg := &program{
		cliContext: c,
	}

	svc, err := service.New(prg, svcConfig)
	if err != nil {
		svc = nil
		zap.L().Fatal("new service error")
	}

	if c.IsSet("service") {
		if svc == nil {
			zap.L().Fatal("Built-in service installation is not supported on this platform")
		}
		if err := service.Control(svc, c.String("service")); err != nil {
			zap.L().Fatal("service operation failed", zap.String("op", c.String("service")), zap.Error(err))
		}
		switch c.String("service") {
		case "install":
			zap.L().Info("Installed as a service. Use `--service start` to start")
		case "uninstall":
			zap.L().Info("Service uninstalled")
		case "start":
			zap.L().Info("Service started")
		case "stop":
			zap.L().Info("Service stopped")
		case "restart":
			zap.L().Info("Service restarted")
		}
		return
	}

	if svc != nil && c.Bool("from-service") {
		svc.Run()
	} else {
		prg.run()
	}
}

func main() {
	initLog("stdout", "stderr", zapcore.DebugLevel)

	// parse flags
	if err := getApp().Run(os.Args); err != nil {
		zap.L().Fatal("flag unexpected error", zap.Error(err))
	}
}
