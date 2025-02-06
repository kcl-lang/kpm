package logger

import (
	"fmt"
	"os"
)

// LogWriter is a custom log writer with log level control.
type LogWriter struct {
	Level LogLevel
}

// LogLevel represents the log level.
type LogLevel string

const (
	InfoLevel  LogLevel = "info"
	DebugLevel LogLevel = "debug"
)

// NewLogWriter creates a new LogWriter with the given level.
func NewLogWriter() *LogWriter {
	level := os.Getenv("KPM_LOG_LEVEL")
	if level == "" {
		level = string(InfoLevel)
	}
	return &LogWriter{Level: LogLevel(level)}
}

// Write implements the io.Writer interface.
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	if lw.Level == DebugLevel {
		fmt.Fprintf(os.Stdout, "[DEBUG] %s", string(p))
	} else {
		fmt.Fprintf(os.Stdout, "[INFO] %s", string(p))
	}
	return len(p), nil
}
