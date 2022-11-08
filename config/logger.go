package config

import (
	lg "log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger // Global logger

func DoConfigureLogger(logPath string, logLevel string) {
	// only log to path if set
	outputPaths := []string{"stdout"}
	if len(logPath) > 0 {
		outputPaths = append(outputPaths, logPath)
	}

	// Logger
	cfg := zap.Config{
		OutputPaths: outputPaths,
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "message",
			LevelKey:    "level",
			EncodeLevel: zapcore.LowercaseLevelEncoder,
			EncodeTime:  zapcore.ISO8601TimeEncoder,
			TimeKey:     "timestamp",
		},
		Encoding: "json",
		Level:    zap.NewAtomicLevel(),
	}

	al, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		lg.Fatalf("logger setup failure. Err: %v", err)
	}

	cfg.Level = al
	Log, err = cfg.Build()
	if err != nil {
		lg.Fatalf("logger setup failure. Err: %v", err)
	}
}
