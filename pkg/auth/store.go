package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// ProviderStore persists a host → provider-name mapping in a small JSON
// sidecar file so that `kpm login` (writer) and `kpm pull` (reader) can
// share state about which OCI registries are owned by a non-static
// credential provider (e.g. GCP Workload Identity).
//
// The sidecar lives next to the existing ORAS credential store at
// `<kpm-home>/.kpm/config/providers.json`. ORAS itself does not read
// this file — kpm consults it via Resolve() before falling back to the
// ORAS store.
//
// Concurrency: safe for use from multiple goroutines. The store lazily
// loads from disk on first access; writes re-serialise the map. There is
// no cross-process locking — if two kpm processes run against the same
// home at the same time, last writer wins. This matches ORAS's own
// behaviour for the config.json it owns.
type ProviderStore struct {
	// path is the on-disk location of the sidecar file.
	path string

	// mu protects loaded, providers, and loadErr. It is held in write
	// mode during Set/Delete/save and read mode during Get.
	mu sync.RWMutex
	// loaded records whether we've already attempted to read the file.
	// If false and the file doesn't exist yet, the store is just an
	// empty map.
	loaded bool
	// providers maps registry hostname → provider name (e.g. "gcp").
	providers map[string]string
	// loadErr is the sticky error from the first load attempt, if any.
	// Subsequent Get/Set calls will surface it so callers see real I/O
	// failures instead of silently treating the file as empty.
	loadErr error
}

// providerStoreFile is the on-disk schema.
//
// We version the format so that future migrations are explicit. The
// current version only has the Providers map.
type providerStoreFile struct {
	// Version is the schema version. Currently always 1.
	Version int `json:"version"`
	// Providers maps registry hostname → provider name.
	Providers map[string]string `json:"providers"`
}

// currentProviderStoreVersion is the only version this codebase knows.
const currentProviderStoreVersion = 1

// OpenProviderStore returns a store backed by path. The file at path
// is not read until the first Get/Set/Delete call. If the file does
// not exist yet, the store behaves as empty until something is written.
//
// path is expected to point at a regular file path, not a directory.
func OpenProviderStore(path string) *ProviderStore {
	return &ProviderStore{
		path:      path,
		providers: map[string]string{},
	}
}

// load reads the sidecar from disk if not already loaded. The caller
// must hold s.mu (read or write).
func (s *ProviderStore) load() error {
	if s.loaded {
		return s.loadErr
	}
	s.loaded = true

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// First-time use: leave providers empty and treat as
			// loaded. We do NOT set loadErr here so that subsequent
			// writes succeed even when no file exists yet.
			return nil
		}
		s.loadErr = err
		return err
	}

	var f providerStoreFile
	if err := json.Unmarshal(data, &f); err != nil {
		s.loadErr = fmt.Errorf("kpm auth: parse %s: %w", s.path, err)
		return s.loadErr
	}

	if f.Version != currentProviderStoreVersion {
		s.loadErr = fmt.Errorf("kpm auth: %s: unsupported provider store version %d (want %d)", s.path, f.Version, currentProviderStoreVersion)
		return s.loadErr
	}

	if f.Providers != nil {
		s.providers = f.Providers
	}
	return nil
}

// save serialises the in-memory map to disk. Caller must hold s.mu in
// write mode.
func (s *ProviderStore) save() error {
	f := providerStoreFile{
		Version:   currentProviderStoreVersion,
		Providers: s.providers,
	}
	data, err := json.MarshalIndent(&f, "", "  ")
	if err != nil {
		return fmt.Errorf("kpm auth: marshal provider store: %w", err)
	}

	// Atomic-ish write: ensure the parent directory exists, then
	// write to a sibling temp file, fsync, rename. This avoids
	// leaving a half-written file behind if kpm is killed mid-write.
	if dir := filepath.Dir(s.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("kpm auth: create %s: %w", dir, err)
		}
	}
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("kpm auth: write %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("kpm auth: rename %s -> %s: %w", tmpPath, s.path, err)
	}
	return nil
}

// Get returns the provider name registered for host. The empty string
// means "no provider configured" — the caller should fall back to the
// ORAS credential store.
//
// The returned error is non-nil only if the sidecar exists but could
// not be read or parsed. A missing sidecar is not an error.
func (s *ProviderStore) Get(host string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.load(); err != nil {
		return "", err
	}
	return s.providers[host], nil
}

// Set records that host is owned by providerName. Calling Set with the
// same host overwrites the previous provider. Set is the only operation
// that creates the sidecar on disk; if the file does not exist yet,
// Set creates it (and its parent directory).
func (s *ProviderStore) Set(host, providerName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return err
	}

	if s.providers[host] == providerName {
		// No-op: already configured. Avoid touching the file
		// unnecessarily so we don't disturb a concurrent reader.
		return nil
	}
	s.providers[host] = providerName
	return s.save()
}

// Delete removes the entry for host. It is a no-op (no error) if the
// host is not currently registered, so callers can call Delete
// unconditionally on logout.
func (s *ProviderStore) Delete(host string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return err
	}

	if _, ok := s.providers[host]; !ok {
		return nil
	}
	delete(s.providers, host)
	return s.save()
}
