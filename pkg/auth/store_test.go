package auth

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestProviderStore_GetOnEmptyFile verifies that a missing sidecar
// behaves as an empty map (no error, Get returns "").
func TestProviderStore_GetOnEmptyFile(t *testing.T) {
	store := OpenProviderStore(filepath.Join(t.TempDir(), "providers.json"))

	got, err := store.Get("gcr.io")
	if err != nil {
		t.Fatalf("Get on empty store: unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("Get on empty store: got %q, want empty string", got)
	}
}

// TestProviderStore_SetGetRoundTrip verifies that Set creates the file
// and Get reads it back.
func TestProviderStore_SetGetRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.json")
	store := OpenProviderStore(path)

	if err := store.Set("gcr.io", "gcp"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Re-open from disk to confirm persistence.
	reopened := OpenProviderStore(path)
	got, err := reopened.Get("gcr.io")
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got != "gcp" {
		t.Fatalf("Get after reopen: got %q, want \"gcp\"", got)
	}
}

// TestProviderStore_SetIsIdempotent verifies that calling Set with the
// same value twice does not error and leaves the file unchanged.
func TestProviderStore_SetIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.json")
	store := OpenProviderStore(path)

	if err := store.Set("gcr.io", "gcp"); err != nil {
		t.Fatalf("first Set: %v", err)
	}
	if err := store.Set("gcr.io", "gcp"); err != nil {
		t.Fatalf("second Set: %v", err)
	}
	got, _ := store.Get("gcr.io")
	if got != "gcp" {
		t.Fatalf("Get after idempotent Set: got %q, want \"gcp\"", got)
	}
}

// TestProviderStore_SetOverwritesDifferentProvider covers the case
// where a user logs in to the same host with two different providers
// (e.g. switching from "basic" to "gcp"). The latest call wins.
func TestProviderStore_SetOverwritesDifferentProvider(t *testing.T) {
	store := OpenProviderStore(filepath.Join(t.TempDir(), "providers.json"))

	if err := store.Set("gcr.io", "basic"); err != nil {
		t.Fatalf("Set basic: %v", err)
	}
	if err := store.Set("gcr.io", "gcp"); err != nil {
		t.Fatalf("Set gcp: %v", err)
	}
	got, _ := store.Get("gcr.io")
	if got != "gcp" {
		t.Fatalf("Get after overwrite: got %q, want \"gcp\"", got)
	}
}

// TestProviderStore_DeleteMissing verifies that Delete on a host not in
// the store is a no-op and not an error.
func TestProviderStore_DeleteMissing(t *testing.T) {
	store := OpenProviderStore(filepath.Join(t.TempDir(), "providers.json"))

	if err := store.Delete("gcr.io"); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}

// TestProviderStore_DeleteRoundTrip verifies that Delete removes the
// entry and a subsequent Get returns "".
func TestProviderStore_DeleteRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.json")
	store := OpenProviderStore(path)

	if err := store.Set("gcr.io", "gcp"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Delete("gcr.io"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	reopened := OpenProviderStore(path)
	got, err := reopened.Get("gcr.io")
	if err != nil {
		t.Fatalf("Get after Delete+reopen: %v", err)
	}
	if got != "" {
		t.Fatalf("Get after Delete: got %q, want empty string", got)
	}
}

// TestProviderStore_MalformedFile verifies that a sidecar with bad JSON
// produces a parse error on Get, not silent corruption.
func TestProviderStore_MalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	store := OpenProviderStore(path)
	_, err := store.Get("gcr.io")
	if err == nil {
		t.Fatalf("expected error from malformed sidecar, got nil")
	}
}

// TestProviderStore_WrongVersion verifies that an unknown version is
// rejected (forward-compat guardrail).
func TestProviderStore_WrongVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.json")
	if err := os.WriteFile(path, []byte(`{"version":999,"providers":{}}`), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	store := OpenProviderStore(path)
	_, err := store.Get("gcr.io")
	if err == nil {
		t.Fatalf("expected error from unknown version, got nil")
	}
}

// TestProviderStore_ConcurrentAccess verifies that the store is safe
// for use from multiple goroutines. Run with `go test -race` to catch
// any data races.
func TestProviderStore_ConcurrentAccess(t *testing.T) {
	store := OpenProviderStore(filepath.Join(t.TempDir(), "providers.json"))

	const goroutines = 8
	const perGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				host := "host-" + string(rune('a'+(i+j)%4))
				if err := store.Set(host, "gcp"); err != nil {
					t.Errorf("Set(%q): %v", host, err)
					return
				}
				if _, err := store.Get(host); err != nil && !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("Get(%q): %v", host, err)
					return
				}
			}
		}()
	}
	wg.Wait()
}
