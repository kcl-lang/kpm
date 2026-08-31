package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	remoteauth "oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

// newResolveFakeMetadata stands up an httptest server that mimics the
// GCE metadata endpoint well enough for GCPProvider to mint a token.
// It records the number of token mint calls so tests can assert that
// Resolve() called the provider each time it was invoked.
//
// The fake is intentionally lightweight — only the two routes the
// GCPProvider hits need to exist. Other paths return 404.
func newResolveFakeMetadata(t *testing.T) (*httptest.Server, *int32) {
	t.Helper()
	var tokenCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc("/computeMetadata/v1/instance/id", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		_, _ = w.Write([]byte("fake-instance"))
	})
	mux.HandleFunc("/computeMetadata/v1/instance/service-accounts/default/token", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenCalls, 1)
		w.Header().Set("Metadata-Flavor", "Google")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "ya29.test-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &tokenCalls
}

// pointGCPAtFakeServer sets GCE_METADATA_HOST to host:port so the
// default GCPProvider hits our httptest server instead of the real
// metadata server.
func pointGCPAtFakeServer(t *testing.T, srv *httptest.Server) {
	t.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse fake metadata URL: %v", err)
	}
	t.Setenv("GCE_METADATA_HOST", u.Host)
}

// TestResolver_NilSafe verifies that a nil Resolver behaves like ORAS's
// EmptyCredential — ORAS treats that as "no auth for this host".
func TestResolver_NilSafe(t *testing.T) {
	var r *Resolver
	cred, err := r.Resolve(context.Background(), "gcr.io")
	if err != nil {
		t.Fatalf("nil Resolve: %v", err)
	}
	if cred != remoteauth.EmptyCredential {
		t.Fatalf("nil Resolve: got %+v, want EmptyCredential", cred)
	}
}

// TestResolver_NoProviderConfigured verifies that with an empty
// ProviderStore and a populated ORAS fallback, Resolve reads the
// ORAS store.
func TestResolver_NoProviderConfigured(t *testing.T) {
	store := newMemStore(t)
	if err := store.Put(context.Background(), "gcr.io", remoteauth.Credential{Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := NewResolver(nil, store)
	cred, err := r.Resolve(context.Background(), "gcr.io")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cred.Username != "alice" || cred.Password != "secret" {
		t.Fatalf("Resolve: got %+v, want alice/secret", cred)
	}
}

// TestResolver_ProviderRoutedToCredentialFunc verifies that when the
// ProviderStore maps a host to the built-in GCP provider, Resolve
// calls GCPProvider.Credential() and returns its result instead of
// touching the ORAS fallback.
func TestResolver_ProviderRoutedToCredentialFunc(t *testing.T) {
	srv, _ := newResolveFakeMetadata(t)
	pointGCPAtFakeServer(t, srv)

	store := newMemStore(t)
	if err := store.Put(context.Background(), "gcr.io", remoteauth.Credential{Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	ps := OpenProviderStore(filepath.Join(t.TempDir(), "providers.json"))
	if err := ps.Set("gcr.io", "gcp"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	r := NewResolver(ps, store)
	cred, err := r.Resolve(context.Background(), "gcr.io")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cred.Username != "oauth2accesstoken" || cred.Password != "ya29.test-token" {
		t.Fatalf("Resolve: got %+v, want GCP-minted token (provider should override ORAS store)", cred)
	}
}

// TestResolver_ProviderMintCalledPerInvocation verifies that the
// provider's Credential function is called every time Resolve is
// invoked. This is what allows GCP tokens (~1h TTL) to be refreshed
// without us owning the TTL cache: ORAS caches the *Bearer token* on
// success, but when the cached token 401s, ORAS re-enters our
// CredentialFunc, which mints fresh.
func TestResolver_ProviderMintCalledPerInvocation(t *testing.T) {
	srv, tokenCalls := newResolveFakeMetadata(t)
	pointGCPAtFakeServer(t, srv)

	ps := OpenProviderStore(filepath.Join(t.TempDir(), "providers.json"))
	if err := ps.Set("gcr.io", "gcp"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	r := NewResolver(ps, newMemStore(t))
	for i := 0; i < 5; i++ {
		if _, err := r.Resolve(context.Background(), "gcr.io"); err != nil {
			t.Fatalf("Resolve: %v", err)
		}
	}
	if got := atomic.LoadInt32(tokenCalls); got != 5 {
		t.Fatalf("provider mints: got %d, want 5", got)
	}
}

// TestResolver_ProviderMintErrorSurfaces verifies that when the
// provider's Credential function errors, Resolve propagates it instead
// of falling through to the ORAS store. This is important: if GCP is
// down we want a clear "cannot mint" error, not silent use of an
// unrelated credential.
func TestResolver_ProviderMintErrorSurfaces(t *testing.T) {
	// Server that returns 404 on every path → GCPProvider surfaces
	// "not running on GCE/GKE" wrapped in ErrCredential.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	pointGCPAtFakeServer(t, srv)

	ps := OpenProviderStore(filepath.Join(t.TempDir(), "providers.json"))
	if err := ps.Set("gcr.io", "gcp"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	r := NewResolver(ps, newMemStore(t))
	_, err := r.Resolve(context.Background(), "gcr.io")
	if err == nil {
		t.Fatalf("expected error from GCP provider outside GCE, got nil")
	}
	if !strings.Contains(err.Error(), "not running on GCE/GKE") {
		t.Fatalf("error should mention GCE/GKE: %v", err)
	}
}

// TestResolver_UnknownProviderReturnsError verifies that an entry in
// the sidecar that points at a provider the current binary does not
// know about is a clear configuration error, not silent fallback.
func TestResolver_UnknownProviderReturnsError(t *testing.T) {
	ps := OpenProviderStore(filepath.Join(t.TempDir(), "providers.json"))
	if err := ps.Set("gcr.io", "does-not-exist"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	r := NewResolver(ps, newMemStore(t))
	_, err := r.Resolve(context.Background(), "gcr.io")
	if err == nil {
		t.Fatalf("expected error from unknown provider, got nil")
	}
}

// TestResolver_FallbackMissing verifies the contract for "host has no
// provider and the ORAS store has no entry" — Resolve returns
// EmptyCredential with no error, which ORAS treats as anonymous
// access (try the public endpoint, fail with a 401 → public).
func TestResolver_FallbackMissing(t *testing.T) {
	r := NewResolver(nil, newMemStore(t))
	cred, err := r.Resolve(context.Background(), "gcr.io")
	if err != nil {
		t.Fatalf("Resolve on empty fallback: %v", err)
	}
	if cred != remoteauth.EmptyCredential {
		t.Fatalf("Resolve on empty fallback: got %+v, want EmptyCredential", cred)
	}
}

// newMemStore returns an in-memory credentials.Store backed by a
// per-test temp file so multiple tests don't collide.
func newMemStore(t *testing.T) credentials.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "oras-config.json")
	s, err := credentials.NewStore(path, credentials.StoreOptions{
		AllowPlaintextPut: true,
	})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}
