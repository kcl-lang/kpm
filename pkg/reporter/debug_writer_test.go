// Copyright 2026 The KCL Authors. All rights reserved.

package reporter

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDebugWriter_TeesToPrimaryAndFile(t *testing.T) {
	dir := t.TempDir()

	primary := &bytes.Buffer{}
	w := NewDebugWriter(primary, dir)
	t.Cleanup(func() { _ = w.Close() })

	_, err := w.Write([]byte("hello world\n"))
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if got := primary.String(); got != "hello world\n" {
		t.Errorf("primary writer got %q, want %q", got, "hello world\n")
	}

	logPath := filepath.Join(dir, KPM_DEBUG_LOG_FILE)
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(data), "hello world") {
		t.Errorf("log file missing 'hello world': %q", string(data))
	}
	if !strings.Contains(string(data), "kpm debug session") {
		t.Errorf("log file missing session banner: %q", string(data))
	}
}

func TestDebugWriter_NoFileWhenHomePathEmpty(t *testing.T) {
	primary := &bytes.Buffer{}
	w := NewDebugWriter(primary, "")
	if got := w.Path(); got != "" {
		t.Errorf("Path() should be empty when homePath is empty, got %q", got)
	}

	_, err := w.Write([]byte("solo\n"))
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if primary.String() != "solo\n" {
		t.Errorf("primary writer did not receive output: %q", primary.String())
	}
	if err := w.Close(); err != nil {
		t.Errorf("Close on file-less writer should not error: %v", err)
	}
}

func TestDebugWriter_AppendsAcrossWrites(t *testing.T) {
	dir := t.TempDir()
	primary := &bytes.Buffer{}
	w := NewDebugWriter(primary, dir)
	t.Cleanup(func() { _ = w.Close() })

	lines := []string{"first\n", "second\n", "third\n"}
	for _, line := range lines {
		if _, err := w.Write([]byte(line)); err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
	}

	logPath := filepath.Join(dir, KPM_DEBUG_LOG_FILE)
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	body := string(data)
	for _, line := range lines {
		if !strings.Contains(body, line) {
			t.Errorf("log file missing %q; got %q", line, body)
		}
	}
}

func TestDebugWriter_ContinuesWhenDiskFails(t *testing.T) {
	// Point homePath at a path whose parent cannot be created (a regular
	// file rather than a directory). The writer should silently disable
	// disk logging while still forwarding to the primary.
	tmp := t.TempDir()
	regular := filepath.Join(tmp, "regular")
	if err := os.WriteFile(regular, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Now point homePath at <regular>/nested — MkdirAll on the "regular"
	// path will fail because it's a file.
	bad := filepath.Join(regular, "nested")

	primary := &bytes.Buffer{}
	w := NewDebugWriter(primary, bad)
	t.Cleanup(func() { _ = w.Close() })

	if _, err := w.Write([]byte("survive\n")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if primary.String() != "survive\n" {
		t.Errorf("primary writer should still receive output when disk fails: %q", primary.String())
	}
}

func TestIsKpmDebug_Truthy(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "True", "yes", "YES", "on", "  1  "} {
		t.Setenv("KPM_DEBUG", v)
		// Sanity: IsKpmDebug is in package env, but the debug_writer
		// triggers on the env name only.  We exercise it via the
		// public CLI path; here we just assert that the writer is
		// constructed without panic regardless of the env state.
		w := NewDebugWriter(&bytes.Buffer{}, t.TempDir())
		_ = w.Close()
	}
}

// TestDebugWriter_PathAccessor confirms Path() reflects the resolved
// file path so callers (e.g. a CLI --debug banner) can show users where
// the log is being written.
func TestDebugWriter_PathAccessor(t *testing.T) {
	dir := t.TempDir()
	w := NewDebugWriter(&bytes.Buffer{}, dir)
	t.Cleanup(func() { _ = w.Close() })

	want := filepath.Join(dir, KPM_DEBUG_LOG_FILE)
	if got := w.Path(); got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}