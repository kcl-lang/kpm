// Copyright 2026 The KCL Authors. All rights reserved.

package utils

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// KPM_PROXY is the environment variable kpm checks for an explicit proxy
// override. When set, it takes precedence over the standard HTTP_PROXY /
// HTTPS_PROXY / NO_PROXY variables consumed by http.ProxyFromEnvironment.
//
// Supported forms:
//
//	KPM_PROXY=http://proxy.local:3128
//	KPM_PROXY=https://user:pass@proxy.local:3128
//
// Set KPM_PROXY=direct to disable any inherited proxy for this process.
const KPM_PROXY = "KPM_PROXY"

// ProxyFunc returns an http.RoundTripper proxy function that combines:
//   - The standard Go behaviour (HTTP_PROXY/HTTPS_PROXY/NO_PROXY via
//     http.ProxyFromEnvironment), unless overridden by $KPM_PROXY.
//   - An optional explicit override via $KPM_PROXY, or "direct" to bypass
//     every proxy.
//
// The function is safe to assign directly to http.Transport.Proxy.
func ProxyFunc(req *http.Request) (*url.URL, error) {
	if v := strings.TrimSpace(os.Getenv(KPM_PROXY)); v != "" {
		if strings.EqualFold(v, "direct") {
			return nil, nil
		}
		u, err := url.Parse(v)
		if err != nil {
			return nil, err
		}
		return u, nil
	}
	return http.ProxyFromEnvironment(req)
}

// NewProxyAwareClient returns an *http.Client whose transport honours the
// KPM_PROXY env var on top of the standard HTTP_PROXY/HTTPS_PROXY/NO_PROXY
// variables. Pass timeout=0 for no client-side timeout.
func NewProxyAwareClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy: ProxyFunc,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}