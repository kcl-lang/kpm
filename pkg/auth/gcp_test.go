package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/compute/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFakeMetadataServer starts an httptest.Server that mimics the GCE
// metadata server, with the response handler under test control via
// the supplied handler. The returned cleanup restores any previous
// GCE_METADATA_HOST env var.
func newFakeMetadataServer(t *testing.T, h http.HandlerFunc) (*httptest.Server, func()) {
	t.Helper()

	srv := httptest.NewServer(h)

	// The metadata package consults GCE_METADATA_HOST when building
	// URLs and prepends "http://" itself, so the env var must be
	// just host:port, no scheme. Save and restore around the test
	// so other tests aren't affected.
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	prev := os.Getenv("GCE_METADATA_HOST")
	require.NoError(t, os.Setenv("GCE_METADATA_HOST", u.Host))

	cleanup := func() {
		srv.Close()
		if prev == "" {
			_ = os.Unsetenv("GCE_METADATA_HOST")
		} else {
			_ = os.Setenv("GCE_METADATA_HOST", prev)
		}
	}
	return srv, cleanup
}

func TestGCPProvider_Name(t *testing.T) {
	p := &GCPProvider{}
	assert.Equal(t, "gcp", p.Name())
}

func TestGCPProvider_Credential_Success(t *testing.T) {
	var instanceIDCalls, tokenCalls int
	handler := func(w http.ResponseWriter, r *http.Request) {
		// The metadata client uses these header flags in real
		// GCE; mirroring them keeps the package's header checks
		// happy even though the package doesn't require them
		// for non-prod hosts.
		w.Header().Set("Metadata-Flavor", "Google")
		w.Header().Set("Server", "Metadata Server")

		switch {
		case strings.HasSuffix(r.URL.Path, "/instance/id"):
			instanceIDCalls++
			_, _ = w.Write([]byte("1234567890123456789"))
		case strings.Contains(r.URL.Path, "/service-accounts/default/token"):
			tokenCalls++
			aud := r.URL.Query().Get("audience")
			assert.Equal(t, "https://gcr.io/my-project", aud,
				"default audience should be https://<registry>")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"access_token": "ya29.test-access-token",
				"expires_in": 3599,
				"token_type": "Bearer"
			}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.String())
			w.WriteHeader(http.StatusNotFound)
		}
	}

	srv, cleanup := newFakeMetadataServer(t, handler)
	defer cleanup()
	_ = srv

	p := &GCPProvider{}
	cred, err := p.Credential(context.Background(), "gcr.io/my-project")
	require.NoError(t, err)
	assert.Equal(t, gcpAccessTokenUsername, cred.Username)
	assert.Equal(t, "ya29.test-access-token", cred.Password)
	assert.Equal(t, 1, instanceIDCalls)
	assert.Equal(t, 1, tokenCalls)
}

func TestGCPProvider_Credential_NotOnGCE(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		// GCE returns 404 for instance/id outside GCE/GKE.
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}

	_, cleanup := newFakeMetadataServer(t, handler)
	defer cleanup()

	p := &GCPProvider{}
	_, err := p.Credential(context.Background(), "gcr.io/my-project")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCredential),
		"not-on-GCE error should wrap ErrCredential, got %v", err)
	assert.Contains(t, err.Error(), "not running on GCE/GKE")
}

func TestGCPProvider_Credential_BadJSON(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		switch {
		case strings.HasSuffix(r.URL.Path, "/instance/id"):
			_, _ = w.Write([]byte("1234567890123456789"))
		case strings.Contains(r.URL.Path, "/service-accounts/default/token"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("not-json-at-all"))
		}
	}

	_, cleanup := newFakeMetadataServer(t, handler)
	defer cleanup()

	p := &GCPProvider{}
	_, err := p.Credential(context.Background(), "gcr.io/my-project")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCredential))
	assert.Contains(t, err.Error(), "parse GCP token response")
}

func TestGCPProvider_Credential_EmptyToken(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		switch {
		case strings.HasSuffix(r.URL.Path, "/instance/id"):
			_, _ = w.Write([]byte("1234567890123456789"))
		case strings.Contains(r.URL.Path, "/service-accounts/default/token"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"expires_in":3599,"token_type":"Bearer"}`))
		}
	}

	_, cleanup := newFakeMetadataServer(t, handler)
	defer cleanup()

	p := &GCPProvider{}
	_, err := p.Credential(context.Background(), "gcr.io/my-project")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCredential))
	assert.Contains(t, err.Error(), "empty GCP access token")
}

func TestGCPProvider_Credential_TokenFetchFails(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		switch {
		case strings.HasSuffix(r.URL.Path, "/instance/id"):
			_, _ = w.Write([]byte("1234567890123456789"))
		case strings.Contains(r.URL.Path, "/service-accounts/default/token"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"server error"}`))
		}
	}

	_, cleanup := newFakeMetadataServer(t, handler)
	defer cleanup()

	p := &GCPProvider{}
	_, err := p.Credential(context.Background(), "gcr.io/my-project")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCredential))
	assert.Contains(t, err.Error(), "fetch GCP access token")
}

func TestGCPProvider_CustomAudience(t *testing.T) {
	var seenAudience string
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		switch {
		case strings.HasSuffix(r.URL.Path, "/instance/id"):
			_, _ = w.Write([]byte("1234567890123456789"))
		case strings.Contains(r.URL.Path, "/service-accounts/default/token"):
			seenAudience = r.URL.Query().Get("audience")
			_, _ = w.Write([]byte(`{"access_token":"ya29.custom-aud","expires_in":3599,"token_type":"Bearer"}`))
		}
	}

	_, cleanup := newFakeMetadataServer(t, handler)
	defer cleanup()

	p := &GCPProvider{Audience: "https://my-registry.example.com"}
	cred, err := p.Credential(context.Background(), "gcr.io/my-project")
	require.NoError(t, err)
	assert.Equal(t, "ya29.custom-aud", cred.Password)
	assert.Equal(t, "https://my-registry.example.com", seenAudience)
}

func TestGCPProvider_WithCustomMetadataClient(t *testing.T) {
	// Demonstrates that tests can pass a custom *metadata.Client
	// without relying on the GCE_METADATA_HOST env var.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		switch {
		case strings.HasSuffix(r.URL.Path, "/instance/id"):
			_, _ = w.Write([]byte("1234567890123456789"))
		case strings.Contains(r.URL.Path, "/service-accounts/default/token"):
			_, _ = w.Write([]byte(`{"access_token":"ya29.injected","expires_in":3599,"token_type":"Bearer"}`))
		}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	// Build an http.Client whose Transport rewrites the host to
	// point at our test server. This is the standard "round tripper
	// rewriter" pattern used in code that has to point an
	// http.Client at a host different from the URL.
	realTransport := http.DefaultTransport
	injected := &http.Client{
		Transport: rewriteHostTransport{inner: realTransport, target: u.Host},
	}
	mc := metadata.NewClient(injected)

	p := &GCPProvider{metadataClient: mc}
	cred, err := p.Credential(context.Background(), "gcr.io/my-project")
	require.NoError(t, err)
	assert.Equal(t, "ya29.injected", cred.Password)
	assert.Equal(t, gcpAccessTokenUsername, cred.Username)
}

// rewriteHostTransport is a RoundTripper that rewrites every request's
// host:port to `target` so the underlying transport talks to the
// httptest server instead of 169.254.169.254.
type rewriteHostTransport struct {
	inner  http.RoundTripper
	target string
}

func (t rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone so we don't mutate the caller's request.
	r2 := req.Clone(req.Context())
	r2.URL.Host = t.target
	r2.URL.Scheme = "http"
	r2.Host = t.target
	return t.inner.RoundTrip(r2)
}
