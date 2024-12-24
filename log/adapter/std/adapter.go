package std

import (
	stdlog "log"

	"github.com/kbgod/lumex/log"
)

type LogLevel int

const (
	LevelDebug = iota
	LevelInfo
	LevelError
)

var _ log.Logger = (*LogAdapter)(nil)

type LogAdapter struct {
	level LogLevel
}

func NewAdapter(level LogLevel) log.Logger {
	return &LogAdapter{
		level: level,
	}
}

func (a *LogAdapter) Error(err error, message string, fields map[string]any) {
	if a.level > LevelError {
		return
	}
	stdlog.Println("[ERROR] ", err, message, fields)
}

func (a *LogAdapter) Info(message string, fields map[string]any) {
	if a.level > LevelInfo {
		return
	}
	stdlog.Println("[INFO] ", message, fields)
}

func (a *LogAdapter) Debug(message string, fields map[string]any) {
	if a.level > LevelDebug {
		return
	}
	stdlog.Println("[DEBUG] ", message, fields)
}
