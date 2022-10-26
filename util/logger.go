package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// A Level is a logging priority. Higher levels are more important.
type Level int8

const (
	DebugLevel Level = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	PanicLevel
	FatalLevel
)

var lvlMapByLvl = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	PanicLevel: "PANIC",
	FatalLevel: "FATAL",
}
var lvlMapByName = map[string]Level{
	"DEBUG": DebugLevel,
	"INFO":  InfoLevel,
	"WARN":  WarnLevel,
	"ERROR": ErrorLevel,
	"PANIC": PanicLevel,
	"FATAL": FatalLevel,
}

type Logger struct {
	log *log.Logger
	lvl Level
}

func NewLogger() *Logger {
	lvlName := os.Getenv("LOG_LEVEL")
	// default to info if not provided from ENV
	if lvlName == "" {
		lvlName = "INFO"
	}
	lvl := lvlMapByName[lvlName]
	newLogger := Logger{
		lvl: lvl,
		log: log.New(os.Stderr, "", 0),
	}
	return &newLogger
}

func (lg *Logger) print(lvl Level, format string, args ...interface{}) {
	caller := findCaller(3)
	var msg string
	if format == "" {
		msg = fmt.Sprint(args...)
	} else {
		msg = fmt.Sprintf(format, args...)
	}
	timeStr := time.Now().UTC().Format(time.RFC3339Nano)
	msg = fmt.Sprintf("[%v]%v %v > %v", lvlMapByLvl[lvl], timeStr, caller, msg)
	switch lvl {
	case PanicLevel:
		lg.log.Panic(msg)
	case FatalLevel:
		lg.log.Fatal(msg)
	default:
		lg.log.Println(msg)
	}
}

func findCaller(distance int) string {
	_, file, line, ok := runtime.Caller(distance)
	if !ok {
		return "caller unknown"
	} else {
		file = filepath.Base(file)
	}
	return fmt.Sprintf("%v:%v", file, line)
}

// Debug will print a debug log msg
func (lg *Logger) Debug(args ...interface{}) {
	if lg.lvl <= DebugLevel {
		lg.print(DebugLevel, "", args...)
	}
}

// Debugf will print a formatted debug log message
func (lg *Logger) Debugf(format string, args ...interface{}) {
	if lg.lvl <= DebugLevel {
		lg.print(DebugLevel, format, args...)
	}
}

// Info will print an info log msg
func (lg *Logger) Info(args ...interface{}) {
	if lg.lvl <= InfoLevel {
		lg.print(InfoLevel, "", args...)
	}
}

// Infof will print a formatted info log message
func (lg *Logger) Infof(format string, args ...interface{}) {
	if lg.lvl <= InfoLevel {
		lg.print(InfoLevel, format, args...)
	}
}

// Warn will print a warning log msg
func (lg *Logger) Warn(args ...interface{}) {
	if lg.lvl <= WarnLevel {
		lg.print(WarnLevel, "", args...)
	}
}

// Warnf will print a formatted warning log message
func (lg *Logger) Warnf(format string, args ...interface{}) {
	if lg.lvl <= WarnLevel {
		lg.print(WarnLevel, format, args...)
	}
}

// Error will print an error log msg
func (lg *Logger) Error(args ...interface{}) {
	if lg.lvl <= ErrorLevel {
		lg.print(ErrorLevel, "", args...)
	}
}

// Errorf will print a formatted error log message
func (lg *Logger) Errorf(format string, args ...interface{}) {
	if lg.lvl <= ErrorLevel {
		lg.print(ErrorLevel, format, args...)
	}
}

// Panic will print a panic log msg
func (lg *Logger) Panic(args ...interface{}) {
	if lg.lvl <= PanicLevel {
		lg.print(PanicLevel, "", args...)
	}
}

// Panicf will print a formatted error log message
func (lg *Logger) Panicf(format string, args ...interface{}) {
	if lg.lvl <= PanicLevel {
		lg.print(PanicLevel, format, args...)
	}
}

// Fatal will print a fatal log msg
func (lg *Logger) Fatal(args ...interface{}) {
	if lg.lvl <= FatalLevel {
		lg.print(FatalLevel, "", args...)
	}
}

// Fatalf will print a formatted fatal log message
func (lg *Logger) Fatalf(format string, args ...interface{}) {
	if lg.lvl <= FatalLevel {
		lg.print(FatalLevel, format, args...)
	}
}

// SetOutput implements the SetOutput function of the underlying logger
// this can be used to point logs at a file for log shipping
// or can be pointed to a buffer for testing.
func (lg *Logger) SetOutput(w io.Writer) {
	lg.log.SetOutput(w)
}

// SetLvl will set the log level to what is provided.
// This can be used to override the level set in the ENV
func (lg *Logger) SetLvl(lvl Level) {
	lg.lvl = lvl
}
