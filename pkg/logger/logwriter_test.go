package logger

import (
	"os"
	"testing"
)

func TestLogWriterInfo(t *testing.T) {
	os.Setenv("KPM_LOG_LEVEL", "info")
	writer := NewLogWriter()

	message := "Test Info Message"
	n, err := writer.Write([]byte(message))
	if err != nil || n != len(message) {
		t.Errorf("expected %d bytes written, got %d, error: %v", len(message), n, err)
	}
}

func TestLogWriterDebug(t *testing.T) {
	os.Setenv("KPM_LOG_LEVEL", "debug")
	writer := NewLogWriter()

	message := "Test Debug Message"
	n, err := writer.Write([]byte(message))
	if err != nil || n != len(message) {
		t.Errorf("expected %d bytes written, got %d, error: %v", len(message), n, err)
	}
}
