// Copyright 2026 The KCL Authors. All rights reserved.

package reporter

import (
	"errors"
	"log"
	"strings"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		in   string
		want LogLevel
	}{
		{"", LogLevelInfo},
		{"info", LogLevelInfo},
		{"INFO", LogLevelInfo},
		{"  info  ", LogLevelInfo},
		{"debug", LogLevelDebug},
		{"DEBUG", LogLevelDebug},
		{"error", LogLevelError},
		{"warn", LogLevelInfo},
		{"warning", LogLevelInfo},
		{"trace", LogLevelInfo}, // unknown → default
		{"garbage", LogLevelInfo},
	}
	for _, c := range cases {
		got := ParseLogLevel(c.in)
		if got != c.want {
			t.Errorf("ParseLogLevel(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestFatalFilteredMessage(t *testing.T) {
	underlying := errors.New("internal failure")

	cases := []struct {
		name     string
		level    LogLevel
		args     []any
		wantMsg  string
		wantBool bool
	}{
		{
			name:     "info suppresses underlying err",
			level:    LogLevelInfo,
			args:     []any{NewErrorEvent(FailedPush, underlying, "user-facing message")},
			wantMsg:  "user-facing message",
			wantBool: true,
		},
		{
			name:     "error suppresses underlying err",
			level:    LogLevelError,
			args:     []any{NewErrorEvent(FailedPush, underlying, "msg only")},
			wantMsg:  "msg only",
			wantBool: true,
		},
		{
			name:     "debug emits the full Error() string",
			level:    LogLevelDebug,
			args:     []any{NewErrorEvent(FailedPush, underlying, "msg")},
			wantBool: false,
		},
		{
			name:     "non-KpmEvent args fall through to log.Fatal",
			level:    LogLevelInfo,
			args:     []any{"plain string"},
			wantBool: false,
		},
		{
			name:     "KpmEvent with empty msg falls through",
			level:    LogLevelInfo,
			args:     []any{NewErrorEvent(FailedPush, underlying, "")},
			wantBool: false,
		},
		{
			name:     "multiple args fall through",
			level:    LogLevelInfo,
			args:     []any{"first", "second"},
			wantBool: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			msg, ok := fatalFilteredMessage(c.level, c.args)
			if ok != c.wantBool {
				t.Fatalf("ok = %v, want %v", ok, c.wantBool)
			}
			if c.wantMsg != "" && !strings.Contains(msg, c.wantMsg) {
				t.Errorf("msg = %q, want substring %q", msg, c.wantMsg)
			}
		})
	}
}

func TestInitReporterWithLevel_Prefix(t *testing.T) {
	// We only assert that the prefix carries the level name — exact flags are
	// not part of the public contract.
	InitReporterWithLevel(LogLevelDebug)
	if !strings.Contains(prefix(), "debug") {
		t.Errorf("debug prefix missing: %q", prefix())
	}
	InitReporterWithLevel(LogLevelInfo)
	if !strings.Contains(prefix(), "info") {
		t.Errorf("info prefix missing: %q", prefix())
	}
	InitReporterWithLevel(LogLevelError)
	if !strings.Contains(prefix(), "error") {
		t.Errorf("error prefix missing: %q", prefix())
	}
}

// prefix returns the active log prefix without leaking log.Logger details
// into the test file.
func prefix() string {
	return log.Prefix()
}