package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/mock"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
)

// pushWithForce - helper function for push operations with force parameter
func pushWithForce(kpmcli *KpmClient, pushedModPath string, force bool) error {
	return kpmcli.Push(
		WithPushModPath(pushedModPath),
		WithPushForce(force),
		WithPushSource(
			downloader.Source{
				Oci: &downloader.Oci{
					Reg:  "localhost:5002",
					Repo: "test/push_0",
				},
			},
		),
	)
}

func waitForRegistry(t *testing.T, addr string) {
	t.Helper()

	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get("http://" + addr + "/v2/")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("registry %s did not become ready in time", addr)
}

func TestPush(t *testing.T) {
	testFunc := func(t *testing.T, kpmcli *KpmClient) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping test on Windows")
		}
		enableLocalRegistryPlainHTTP(t, kpmcli)

		err := mock.StartDockerRegistry()
		if err != nil {
			t.Errorf("Error starting docker registry: %v", err)
		}

		defer func() {
			err = mock.CleanTestEnv()
			if err != nil {
				t.Errorf("Error stopping docker registry: %v", err)
			}
		}()
		waitForRegistry(t, "localhost:5002")

		kpmcli.SetInsecureSkipTLSverify(true)
		err = kpmcli.LoginOci("localhost:5002", "test", "1234")
		if err != nil {
			t.Errorf("Error logging in to docker registry: %v", err)
		}

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		testDir := getTestDir("test_push")
		pushedModPath := filepath.Join(testDir, "push_0")

		// === Test 1: First push (should succeed) ===
		t.Log("=== Test 1: First push should succeed ===")
		err = pushWithForce(kpmcli, pushedModPath, false)

		if err != (*reporter.KpmEvent)(nil) {
			t.Errorf("Error: First push should succeed: %v", err)
		}

		assert.Contains(t, buf.String(), "package 'push_0' will be pushed")
		assert.Contains(t, buf.String(), "pushed [registry] localhost:5002/test/push_0")
		assert.Contains(t, buf.String(), "digest: sha256:")

		repo, err := remote.NewRepository("localhost:5002/test/push_0")
		assert.NoError(t, err)
		repo.PlainHTTP = true
		repo.Client = &auth.Client{
			Client: retry.DefaultClient,
			Cache:  auth.DefaultCache,
			Credential: auth.StaticCredential("localhost:5002", auth.Credential{
				Username: "test",
				Password: "1234",
			}),
		}

		_, manifestContent, err := oras.FetchBytes(context.Background(), repo, "0.0.1", oras.DefaultFetchBytesOptions)
		if assert.NoError(t, err) {
			var manifest v1.Manifest
			if assert.NoError(t, json.Unmarshal(manifestContent, &manifest)) {
				assert.Equal(t, "application/vnd.oci.image.layer.v1.tar+gzip", manifest.ArtifactType)
				if assert.Len(t, manifest.Layers, 1) {
					assert.Equal(t, "application/vnd.oci.image.layer.v1.tar+gzip", manifest.Layers[0].MediaType)
					assert.Equal(t, "push_0_0.0.1.tgz", manifest.Layers[0].Annotations["org.opencontainers.image.title"])
				}
			}
		}

		// Clean the buffer for the next test
		buf.Reset()

		// === Test 2: Second push with force (should overwrite) ===
		t.Log("=== Test 2: Second push with force should overwrite ===")
		err = pushWithForce(kpmcli, pushedModPath, true)

		if err != (*reporter.KpmEvent)(nil) {
			t.Errorf("Error: Second push with force should succeed: %v", err)
		}

		assert.Contains(t, buf.String(), "package 'push_0' will be pushed")
		assert.Contains(t, buf.String(), "package version '0.0.1' already exists, force pushing")
		assert.Contains(t, buf.String(), "pushed [registry] localhost:5002/test/push_0")

		// Clean the buffer for the next test
		buf.Reset()

		// === Test 3: Third push without force (should fail) ===
		t.Log("=== Test 3: Third push without force should fail ===")
		err = pushWithForce(kpmcli, pushedModPath, false)

		// Check that this operation failed
		if err == (*reporter.KpmEvent)(nil) {
			t.Errorf("Third push without force should fail, but it succeeded")
		} else {
			t.Logf("Expected error occurred: %v", err)

			assert.Contains(t, buf.String(), "package 'push_0' will be pushed")
			assert.Contains(t, err.Error(), "already exists")
		}

		testPushModPath := filepath.Join(testDir, "test_pushed_mod")

		err = kpmcli.Init(
			WithInitModPath(testPushModPath),
		)
		if err != nil {
			t.Errorf("Error initializing kcl package: %v", err)
		}

		testMod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(testPushModPath),
		)

		err = kpmcli.Add(
			WithAddKclPkg(testMod),
			WithAddSourceUrl("oci://localhost:5002/test/push_0"),
			WithAddModSpec(&downloader.ModSpec{
				Name:    "push_0",
				Version: "0.0.1",
			}),
		)

		if err != nil {
			t.Errorf("Error adding kcl package: %v", err)
		}

	}
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestPush", TestFunc: testFunc}})
}

// Simple unit test for the Force option
func TestPushForceOption(t *testing.T) {
	tests := []struct {
		name     string
		force    bool
		expected bool
	}{
		{
			name:     "Force enabled",
			force:    true,
			expected: true,
		},
		{
			name:     "Force disabled",
			force:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &PushOptions{}

			option := WithPushForce(tt.force)
			err := option(opts)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, opts.Force, "Force option should be set to %v", tt.expected)
		})
	}
}
