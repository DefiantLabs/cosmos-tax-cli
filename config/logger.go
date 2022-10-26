package config

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lg "log"
)

var Logger *zap.Logger //Global logger

func DoConfigureLogger(logPath string, logLevel string) {
	//Logger
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

	al, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		lg.Fatalf("logger setup failure. Err: %v", err)
	}

	cfg.Level = al
	Logger, err = cfg.Build()
	if err != nil {
		lg.Fatalf("logger setup failure. Err: %v", err)
	}
}
