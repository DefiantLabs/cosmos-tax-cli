package config

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger //Global logger

func DoConfigureLogger(logPath string, logLevel string) {
	//Logger
	var logErr error
	cfg := zap.Config{
		OutputPaths: []string{logPath},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "message",
			LevelKey:    "level",
			EncodeLevel: zapcore.LowercaseLevelEncoder,
			EncodeTime:  zapcore.ISO8601TimeEncoder,
		},
		Encoding: "json",
		Level:    zap.NewAtomicLevel(),
	}

	al, logErr := zap.ParseAtomicLevel(logLevel)
	if logErr != nil {
		fmt.Println("logger setup failure")
		os.Exit(1)
	}
	cfg.Level = al
	Logger, logErr = cfg.Build()

	if logErr != nil {
		fmt.Println("logger setup failure")
		os.Exit(1)
	}
}
