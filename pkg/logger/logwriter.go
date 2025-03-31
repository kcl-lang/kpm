package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

type LogWriter struct {
	Level LogLevel
}

type LogLevel string

const (
	InfoLevel  LogLevel = "info"
	DebugLevel LogLevel = "debug"
)

func NewLogWriter() *LogWriter {
	level := os.Getenv("KPM_LOG_LEVEL")
	if level == "" {
		level = string(InfoLevel)
	}
	return &LogWriter{Level: LogLevel(level)}
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	if lw.Level == DebugLevel {

		buf := make([]byte, 1024)
		runtime.Stack(buf, false)

		debugMessage := fmt.Sprintf(
			"[DEBUG] %s\nTimestamp: %s\nStack Trace:\n%s\n",
			string(p),
			time.Now().Format(time.RFC3339),
			string(buf),
		)
		fmt.Fprint(os.Stdout, debugMessage)
		return len(debugMessage), nil
	}

	message := extractRelevantInfo(string(p))
	fmt.Fprintf(os.Stdout, "[INFO] %s", message)
	return len(p), nil
}

func extractRelevantInfo(message string) string {
	lines := strings.Split(message, "\n")
	var filteredLines []string

	for _, line := range lines {
		if strings.Contains(line, "[FAILED] Expected") ||
			strings.Contains(line, "In [It] at:") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}

	return strings.Join(filteredLines, "\n")
}
