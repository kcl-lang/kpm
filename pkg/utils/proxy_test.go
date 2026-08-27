// Copyright 2026 The KCL Authors. All rights reserved.

package utils

import (
	"net/http"
	"net/url"
	"testing"
)

// clearProxyEnv clears every standard proxy env var so tests do not
// inherit state from the host shell. t.Setenv records the original value
// and restores it on test exit.
func clearProxyEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY",
		"http_proxy", "https_proxy", "no_proxy",
	} {
		t.Setenv(name, "")
	}
}

// TestProxyFunc_KpmProxyOverrides covers the primary contract:
// KPM_PROXY must shadow the standard HTTP_PROXY/HTTPS_PROXY env vars
// that go's http.ProxyFromEnvironment reads.
func TestProxyFunc_KpmProxyOverrides(t *testing.T) {
	clearProxyEnv(t)
	t.Setenv("KPM_PROXY", "http://kpm-proxy.local:8080")
	t.Setenv("HTTPS_PROXY", "http://standard-proxy.local:3128")

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	got, err := ProxyFunc(req)
	if err != nil {
		t.Fatalf("ProxyFunc returned error: %v", err)
	}
	if got == nil {
		t.Fatalf("expected non-nil proxy URL")
	}
	if got.String() != "http://kpm-proxy.local:8080" {
		t.Errorf("got %q, want kpm override", got.String())
	}
}

func TestProxyFunc_DirectBypassesAllProxies(t *testing.T) {
	clearProxyEnv(t)
	t.Setenv("KPM_PROXY", "direct")
	t.Setenv("HTTPS_PROXY", "http://standard-proxy.local:3128")

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	got, err := ProxyFunc(req)
	if err != nil {
		t.Fatalf("ProxyFunc returned error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil (direct), got %q", got.String())
	}
}

func TestProxyFunc_InvalidURLReturnsError(t *testing.T) {
	clearProxyEnv(t)
	t.Setenv("KPM_PROXY", "://not a url")

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	got, err := ProxyFunc(req)
	if err == nil {
		t.Fatalf("expected error for malformed URL, got %v", got)
	}
	_ = got
}

func TestProxyFunc_TrimsWhitespace(t *testing.T) {
	clearProxyEnv(t)
	t.Setenv("KPM_PROXY", "  http://trim.local:8080  ")

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	got, err := ProxyFunc(req)
	if err != nil {
		t.Fatalf("ProxyFunc returned error: %v", err)
	}
	if got == nil {
		t.Fatalf("expected non-nil proxy URL")
	}
	want, _ := url.Parse("http://trim.local:8080")
	if got.String() != want.String() {
		t.Errorf("got %q, want %q (whitespace must be trimmed)", got.String(), want.String())
	}
}

func TestProxyFunc_EmptyKpmProxyFallsThrough(t *testing.T) {
	// With KPM_PROXY unset/empty, ProxyFunc must fall through to
	// http.ProxyFromEnvironment. We don't assert the exact result
	// because Go's proxy env cache is process-wide and not
	// t.Setenv-friendly; we only verify the call does not panic
	// or return an error.
	clearProxyEnv(t)
	t.Setenv("KPM_PROXY", "")

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	if _, err := ProxyFunc(req); err != nil {
		t.Fatalf("ProxyFunc returned error: %v", err)
	}
}

func TestNewProxyAwareClient_HasProxyFunc(t *testing.T) {
	c := NewProxyAwareClient(0)
	if c.Transport == nil {
		t.Fatalf("Transport is nil")
	}
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.Transport)
	}
	if tr.Proxy == nil {
		t.Errorf("Transport.Proxy is nil; expected utils.ProxyFunc")
	}
}

func TestNewProxyAwareClient_AppliesTimeout(t *testing.T) {
	c := NewProxyAwareClient(1234)
	if c.Timeout.Nanoseconds() != 1234 {
		t.Errorf("Timeout = %v, want 1234ns", c.Timeout)
	}
}

// NOTE on test coverage:
// We intentionally do not test the "no KPM_PROXY" branch against the
// standard HTTP_PROXY/HTTPS_PROXY env vars with a specific URL, because
// Go's http.ProxyFromEnvironment caches the env on first read via
// sync.Once (see src/net/http/proxy.go). Once initialised, subsequent
// t.Setenv calls in the same test process cannot influence the cached
// value. The KPM_PROXY-specific tests above exercise ProxyFunc in
// isolation; the "fall through" path is covered by
// TestProxyFunc_EmptyKpmProxyFallsThrough.