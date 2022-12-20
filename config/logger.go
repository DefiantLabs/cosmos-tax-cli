package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"go.uber.org/zap"
)

type Logger struct {
	ZeroLogger *zerolog.Logger
}

// Log is exposed on the config as a drop-in replacement for our old logger
var Log Logger

func (l *Logger) Fatal(msg string, err ...zap.Field) {
	l.ZeroLogger.Fatal().Msg(fmt.Sprint(msg, err))
}

func (l *Logger) Error(msg string, err ...zap.Field) {
	l.ZeroLogger.Error().Msg(fmt.Sprint(msg, err))
}

func (l *Logger) Debug(msg string, err ...zap.Field) {
	l.ZeroLogger.Debug().Msg(fmt.Sprint(msg, err))
}

func (l *Logger) Warn(msg string, err ...zap.Field) {
	l.ZeroLogger.Warn().Msg(fmt.Sprint(msg, err))
}

func (l *Logger) Info(msg string, err ...zap.Field) {
	l.ZeroLogger.Info().Msg(fmt.Sprint(msg, err))
}

func DoConfigureLogger(logPath string, logLevel string) {
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	Log.ZeroLogger = &logger

	// Set the log level (default to info)
	switch strings.ToLower(logLevel) {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Cmds like this can be called to modify the logger time format
	// zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// FIXME: figure out how to log to a file.
}
