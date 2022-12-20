package config

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

type Logger struct {
	ZeroLogger *zerolog.Logger
}

// Log is exposed on the config as a drop-in replacement for our old logger
var Log Logger

// These functions are provided to reduce refactoring.
func (l *Logger) Debug(msg string, err ...error) {
	if len(err) == 1 {
		l.ZeroLogger.Debug().Err(err[0]).Msg(msg)
	}
	l.ZeroLogger.Debug().Msg(msg)
}

func (l *Logger) Info(msg string, err ...error) {
	if len(err) == 1 {
		l.ZeroLogger.Info().Err(err[0]).Msg(msg)
	}
	l.ZeroLogger.Info().Msg(msg)
}

func (l *Logger) Warn(msg string, err ...error) {
	if len(err) == 1 {
		l.ZeroLogger.Warn().Err(err[0]).Msg(msg)
	}
	l.ZeroLogger.Warn().Msg(msg)
}

func (l *Logger) Error(msg string, err ...error) {
	if len(err) == 1 {
		l.ZeroLogger.Error().Err(err[0]).Msg(msg)
	}
	l.ZeroLogger.Error().Msg(msg)
}

func (l *Logger) Fatal(msg string, err ...error) {
	if len(err) == 1 {
		l.ZeroLogger.Fatal().Err(err[0]).Msg(msg)
	}
	l.ZeroLogger.Fatal().Msg(msg)
}

func (l *Logger) Panic(msg string, err ...error) {
	if len(err) == 1 {
		l.ZeroLogger.Panic().Err(err[0]).Msg(msg)
	}
	l.ZeroLogger.Panic().Msg(msg)
}

func DoConfigureLogger(logPath string, logLevel string) {
	// TODO: I might be able to make this look cleaner
	writers := io.MultiWriter(os.Stdout)
	if len(logPath) > 0 {
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			file, err := os.Create(logPath)
			if err != nil {
				panic(err)
			}
			writers = io.MultiWriter(os.Stdout, file)
		} else {
			file, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
			if err != nil {
				panic(err)
			}
			writers = io.MultiWriter(os.Stdout, file)
		}
	}
	logger := zerolog.New(writers).With().Timestamp().Logger()
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
}
