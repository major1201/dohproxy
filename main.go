package main

import (
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"os/signal"
	"syscall"
)

// AppVer means the project's version
var AppVer = "0.1.0"

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
	logger.Info("logger initialized")
}

func runApp(c *cli.Context) {
	servers := LoadServersFromConfig(c.String("config"))
	for _, s := range servers {
		go func(s Server) {
			if err := s.Serve(); err != nil {
				zap.L().Fatal("failed to start server", zap.String("type", s.Type()), zap.String("address", s.Address()), zap.Error(err))
			}
		}(s)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	zap.L().Warn("Signal received, stopping", zap.String("signal", s.String()))
}

func main() {
	initLog("stdout", "stderr", zapcore.DebugLevel)

	// parse flags
	if err := getApp().Run(os.Args); err != nil {
		zap.L().Fatal("flag unexpected error", zap.Error(err))
	}
}
