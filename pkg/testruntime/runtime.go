// Copyright 2024 The KCL Authors. All rights reserved.

// Package testruntime provides an in-memory OCI registry for offline KPM tests.
//
// The point of this package is to let kpm tests exercise the real downloader
// code path (pkg/oci, pkg/downloader) against a fake registry that needs no
// network, no Docker, and no shell scripts. Existing tests that currently
// shell out to scripts/reg.sh or hit ghcr.io directly can switch to:
//
//	func TestSomething(t *testing.T) {
//	    rt := testruntime.Start(t)
//	    testruntime.LoadTarFixture(t, rt, "myorg/lib", "0.1.0", "testdata/lib.tar")
//	    t.Setenv("KPM_REG", rt.RegistryURL())
//	    ... real test code ...
//	}
//
// The mock is intentionally minimal — it implements just enough of the OCI
// distribution v2 spec for the routes that oras-go/v2 actually calls during a
// Pull/Push. It is NOT a complete registry implementation; if you need auth,
// search, or other dist-spec features, use the real network or the Docker
// script-based fallback in pkg/mock.
package testruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/errdef"
)

// Store is the in-memory backing store. Tests can push fixtures into it
// before serving starts, or hand the *Runtime to production code that
// talks to it through the http server.
type Store = memory.Store

// Runtime is a handle to a running in-memory OCI registry. It is bound to
// the lifetime of the test via t.Cleanup; tests do not need to Close it.
type Runtime struct {
	t      *testing.T
	store  *Store
	server *httptest.Server

	mu          sync.Mutex
	manifest    map[string]map[string]ocispec.Descriptor // repo -> ref -> manifest
	tagIndex    map[string][]string                     // repo -> tags
	digestIndex map[string]ocispec.Descriptor           // digest -> descriptor
}

// Option configures a Runtime at Start time.
type Option func(*Runtime)

// Start boots an in-memory OCI registry on 127.0.0.1:<random-port>. The
// returned *Runtime:
//   - registers a t.Cleanup() to shut down the server,
//   - exposes RegistryURL() for setting KPM_REG / oci options,
//   - exposes Store() for direct seeding.
//
// Example:
//
//	rt := testruntime.Start(t)
//	t.Setenv("KPM_REG", rt.RegistryURL())
func Start(t *testing.T, opts ...Option) *Runtime {
	t.Helper()
	rt := &Runtime{
		store:       memory.New(),
		manifest:    map[string]map[string]ocispec.Descriptor{},
		tagIndex:    map[string][]string{},
		digestIndex: map[string]ocispec.Descriptor{},
	}
	for _, o := range opts {
		o(rt)
	}
	rt.server = httptest.NewServer(rt.handler())
	t.Cleanup(rt.server.Close)
	return rt
}

// RegistryURL returns the base URL (no /v2 suffix) the registry is reachable
// at. Suitable for KPM_REG.
func (r *Runtime) RegistryURL() string {
	return r.server.URL
}

// Store returns the underlying memory store. Tests that want to seed blobs
// or manifests without going through the HTTP layer can do so directly.
func (r *Runtime) Store() *Store { return r.store }

// Tag records that `ref` resolves to `desc` for `repo`. Call this after
// pushing a manifest into Store() so the /manifests/<ref> and /tags/list
// routes serve it.
func (r *Runtime) Tag(repo, ref string, desc ocispec.Descriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.manifest[repo]; !ok {
		r.manifest[repo] = map[string]ocispec.Descriptor{}
	}
	r.manifest[repo][ref] = desc
	r.tagIndex[repo] = append(r.tagIndex[repo], ref)
}

// PushBlob pushes body into the store under the given media type, indexes
// it for blob lookups (which only see the digest in the URL), and returns
// the descriptor.
func (r *Runtime) PushBlob(ctx context.Context, mediaType string, body []byte) (ocispec.Descriptor, error) {
	desc := content.NewDescriptorFromBytes(mediaType, body)
	if err := r.store.Push(ctx, desc, bytes.NewReader(body)); err != nil {
		return ocispec.Descriptor{}, err
	}
	r.mu.Lock()
	r.digestIndex[desc.Digest.String()] = desc
	r.mu.Unlock()
	return desc, nil
}

// handler routes distribution-spec requests to in-memory state.
//
// Routes implemented:
//   GET  /v2/                                          → 200 OK
//   GET  /v2/<repo>/tags/list                         → list of tags
//   HEAD|GET /v2/<repo>/manifests/<ref>               → manifest
//   HEAD|GET /v2/<repo>/blobs/<digest>                → blob bytes
//
// Anything else returns 404 so callers fail loudly instead of getting a
// silently-wrong "200 with empty body".
func (r *Runtime) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/", r.routeV2)
	return mux
}

func (r *Runtime) routeV2(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/v2/")
	if path == "" {
		// /v2/ ping
		w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
		w.WriteHeader(http.StatusOK)
		return
	}
	repo, rest, ok := splitRoute(path)
	if !ok {
		http.NotFound(w, req)
		return
	}
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, req)
		return
	}
	switch parts[0] {
	case "manifests":
		r.serveManifest(w, req, repo, parts[1])
	case "blobs":
		r.serveBlob(w, req, repo, parts[1])
	case "tags":
		if parts[1] == "list" {
			r.serveTagList(w, req, repo)
			return
		}
		http.NotFound(w, req)
	default:
		http.NotFound(w, req)
	}
}

// splitRoute peels the repo name off the front of a path. The repo name is
// the dotted/slashed bit before `/manifests/`, `/blobs/`, or `/tags/`.
//
// <repo>/manifests/<ref>
// <repo>/blobs/<digest>
// <repo>/tags/list
func splitRoute(path string) (repo, rest string, ok bool) {
	for _, sep := range []string{"/manifests/", "/blobs/", "/tags/"} {
		if i := strings.Index(path, sep); i >= 0 {
			return path[:i], path[i+1:], true
		}
	}
	return "", "", false
}

func (r *Runtime) serveManifest(w http.ResponseWriter, req *http.Request, repo, ref string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	refs, ok := r.manifest[repo]
	if !ok {
		writeError(w, http.StatusNotFound, errdef.ErrNotFound)
		return
	}
	desc, ok := refs[ref]
	if !ok {
		writeError(w, http.StatusNotFound, errdef.ErrNotFound)
		return
	}
	rc, err := r.store.Fetch(req.Context(), desc)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", desc.MediaType)
	w.Header().Set("Docker-Content-Digest", desc.Digest.String())
	if req.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		// Best-effort: client may have gone away mid-stream.
		return
	}
}

func (r *Runtime) serveBlob(w http.ResponseWriter, req *http.Request, repo, digest string) {
	r.mu.Lock()
	desc, ok := r.digestIndex[digest]
	r.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, errdef.ErrNotFound)
		return
	}
	rc, err := r.store.Fetch(req.Context(), desc)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Length", fmt.Sprintf("%d", desc.Size))
	w.Header().Set("Docker-Content-Digest", digest)
	if req.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		return
	}
}

func (r *Runtime) serveTagList(w http.ResponseWriter, req *http.Request, repo string) {
	r.mu.Lock()
	tags := append([]string(nil), r.tagIndex[repo]...)
	r.mu.Unlock()
	resp := struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}{Name: repo, Tags: tags}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// writeError emits a distribution-spec-shaped error body so oras-go parses
// it as an errcode.ErrorResponse instead of a generic HTTP error.
func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]string{{
			"code":    http.StatusText(status),
			"message": err.Error(),
		}},
	})
}

// urlFor returns the registry-relative URL for a (repo, ref) pair. Useful
// when test fixtures want to embed a fully-qualified oci:// URL.
func (r *Runtime) urlFor(repo, ref string) string {
	u := &url.URL{Scheme: "http", Host: strings.TrimPrefix(r.server.URL, "http://"), Path: "/v2/" + repo + "/manifests/" + ref}
	return u.String()
}