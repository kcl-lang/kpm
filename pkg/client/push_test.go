package client

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"

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

func TestPush(t *testing.T) {
	testFunc := func(t *testing.T, kpmcli *KpmClient) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping test on Windows")
		}
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
