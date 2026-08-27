// Copyright 2024 The KCL Authors. All rights reserved.
package testruntime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
)

// pushManifest marshals and stores a manifest in the memory store, returning
// its descriptor. Tests use this instead of relying on oras.Pack, which would
// pull in a much bigger surface area.
func pushManifest(t *testing.T, rt *Runtime, m ocispec.Manifest) ocispec.Descriptor {
	t.Helper()
	body, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	desc := content.NewDescriptorFromBytes(m.MediaType, body)
	if err := rt.Store().Push(context.Background(), desc, strings.NewReader(string(body))); err != nil {
		t.Fatalf("push manifest: %v", err)
	}
	return desc
}

// TestStart_PingIsOk verifies the /v2/ ping returns 200 and a distribution
// API version header, which is the first thing oras-go does when talking
// to a registry.
func TestStart_PingIsOk(t *testing.T) {
	rt := Start(t)
	resp, err := http.Get(rt.RegistryURL() + "/v2/")
	if err != nil {
		t.Fatalf("ping: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ping status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Docker-Distribution-API-Version"); got != "registry/2.0" {
		t.Fatalf("ping api-version = %q, want %q", got, "registry/2.0")
	}
}

// TestRoundTrip_ManifestAndBlob verifies the mock can serve a manifest + blob
// pair that we push into the underlying store. This is the only path oras-go
// Pull hits.
func TestRoundTrip_ManifestAndBlob(t *testing.T) {
	rt := Start(t)

	// Build a fake manifest with one config blob and one layer blob.
	blob := []byte("hello kpm")
	blobDesc, err := rt.PushBlob(context.Background(), ocispec.MediaTypeImageLayerGzip, blob)
	if err != nil {
		t.Fatalf("push blob: %v", err)
	}
	config := []byte(`{"architecture":"amd64"}`)
	configDesc, err := rt.PushBlob(context.Background(), ocispec.MediaTypeImageConfig, config)
	if err != nil {
		t.Fatalf("push config: %v", err)
	}
	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    configDesc,
		Layers:    []ocispec.Descriptor{blobDesc},
	}
	manifestDesc := pushManifest(t, rt, manifest)

	rt.Tag("myorg/lib", "0.1.0", manifestDesc)

	// Fetch the manifest via HTTP — this is the path oras-go uses.
	url := rt.RegistryURL() + "/v2/myorg/lib/manifests/0.1.0"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET manifest: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("manifest status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Docker-Content-Digest"); got != manifestDesc.Digest.String() {
		t.Fatalf("manifest digest header = %q, want %q", got, manifestDesc.Digest.String())
	}

	// Fetch the blob via HTTP.
	blobURL := rt.RegistryURL() + "/v2/myorg/lib/blobs/" + blobDesc.Digest.String()
	resp2, err := http.Get(blobURL)
	if err != nil {
		t.Fatalf("GET blob: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("blob status = %d, want 200", resp2.StatusCode)
	}
	got, _ := io.ReadAll(resp2.Body)
	if string(got) != string(blob) {
		t.Fatalf("blob body = %q, want %q", got, blob)
	}
}

// TestTagsList verifies the /tags/list route returns the tags we registered.
func TestTagsList(t *testing.T) {
	rt := Start(t)
	desc := pushManifest(t, rt, ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
	})
	rt.Tag("myorg/lib", "0.1.0", desc)
	rt.Tag("myorg/lib", "0.2.0", desc)

	resp, err := http.Get(rt.RegistryURL() + "/v2/myorg/lib/tags/list")
	if err != nil {
		t.Fatalf("GET tags/list: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tags/list status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	for _, want := range []string{`"0.1.0"`, `"0.2.0"`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("tags/list body missing %s\nbody: %s", want, string(body))
		}
	}
}

// TestHEAD verifies HEAD requests (used by oras-go to discover digest
// + size before issuing GET) succeed for known references and 404 for
// unknown ones.
func TestHEAD(t *testing.T) {
	rt := Start(t)
	blob := []byte("head-payload")
	blobDesc, err := rt.PushBlob(context.Background(), ocispec.MediaTypeImageLayer, blob)
	if err != nil {
		t.Fatalf("PushBlob: %v", err)
	}

	// Known blob
	req, _ := http.NewRequest(http.MethodHead, rt.RegistryURL()+"/v2/myorg/lib/blobs/"+blobDesc.Digest.String(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HEAD blob: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HEAD known blob status = %d, want 200", resp.StatusCode)
	}

	// Unknown blob → 404
	req2, _ := http.NewRequest(http.MethodHead, rt.RegistryURL()+"/v2/myorg/lib/blobs/sha256:0000000000000000000000000000000000000000000000000000000000000000", nil)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("HEAD unknown blob: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("HEAD unknown blob status = %d, want 404", resp2.StatusCode)
	}
}