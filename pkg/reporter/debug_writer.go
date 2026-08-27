// Copyright 2026 The KCL Authors. All rights reserved.

package reporter

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// KPM_DEBUG_LOG_FILE is the well-known path (relative to the kpm home dir)
// where debug output is appended when debug mode is enabled. Exposed as a
// const so tests can override it via t.Setenv if needed.
const KPM_DEBUG_LOG_FILE = "kpm.log"

// DebugWriter is an io.Writer that tees every write to two destinations:
//
//  1. the primary writer (typically os.Stdout / os.Stderr), unchanged;
//  2. an on-disk log file located under <homePath>/kpm-debug.log, opened
//     once and held until Close.
//
// Writes are serialised so log lines do not interleave between threads.
// The on-disk writer is best-effort: any error opening or writing the log
// file is silently swallowed so debug mode cannot break the main output
// stream.
type DebugWriter struct {
	primary io.Writer
	mu      sync.Mutex
	file    io.WriteCloser
	path    string
	opened  bool
}

// NewDebugWriter wraps `primary` and lazily opens the on-disk log file at
// `homePath/KPM_DEBUG_LOG_FILE`. The returned writer is safe to use even if
// the log file cannot be opened — in that case it falls back to the
// primary writer alone.
//
// Pass `homePath == ""` to disable the on-disk log; in that case only the
// primary writer is used.
func NewDebugWriter(primary io.Writer, homePath string) *DebugWriter {
	w := &DebugWriter{primary: primary}
	if homePath != "" {
		w.path = filepath.Join(homePath, KPM_DEBUG_LOG_FILE)
	}
	return w
}

// Write implements io.Writer.
func (w *DebugWriter) Write(p []byte) (int, error) {
	n, err := w.primary.Write(p)
	w.appendToFile(p)
	return n, err
}

// appendToFile lazily opens the log file on the first write and appends.
// It never returns an error: callers are expected to use the return
// value of the primary Write (above) and any disk failure must not affect
// the user-facing output.
func (w *DebugWriter) appendToFile(p []byte) {
	if w.path == "" {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.opened {
		if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
			w.path = "" // disable disk logging on this error
			return
		}
		f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			w.path = ""
			return
		}
		w.file = f
		w.opened = true
		// Best-effort banner so users can grep the log file by session.
		_, _ = f.Write([]byte("\n--- kpm debug session " + time.Now().UTC().Format(time.RFC3339) + " ---\n"))
	}
	_, _ = w.file.Write(p)
}

// Path returns the resolved on-disk log path, or the empty string if disk
// logging is disabled / failed to initialise.
func (w *DebugWriter) Path() string {
	return w.path
}

// Close flushes and closes the on-disk log file. Safe to call multiple
// times. Errors are swallowed by design.
func (w *DebugWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.opened = false
	return err
}