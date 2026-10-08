package client

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/mock"
	"oras.land/oras-go/v2/registry/remote/auth"
)

func TestLogin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping TestModCheckerCheck_WithTrustedSum test on Windows")
	}

	// Start the local Docker registry required for testing
	err := mock.StartDockerRegistry()
	assert.Equal(t, err, nil)

	defer func() {
		os.Unsetenv("OCI_REG_PLAIN_HTTP")
		err = mock.CleanTestEnv()
		if err != nil {
			t.Errorf("Error stopping docker registry: %v", err)
		}
	}()

	os.Setenv("OCI_REG_PLAIN_HTTP", "ON")
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	err = kpmcli.LoginOci("172.88.0.8:5002", "test", "1234")
	assert.Equal(t, err, nil)
}

// TestLoginCredentialStorePlaintext reproduces the headless-CI scenario
// from https://github.com/kcl-lang/kpm/issues/769: with no credential
// helper configured and without OCI_REG_PLAIN_HTTP, login must still be
// able to persist credentials into the config file instead of failing
// with "putting plaintext credentials is disabled".
func TestLoginCredentialStorePlaintext(t *testing.T) {
	credFile := filepath.Join(t.TempDir(), "config.json")
	kpmcli := &KpmClient{}
	kpmcli.settings.CredentialsFile = credFile

	store, err := kpmcli.credentialStore()
	assert.Equal(t, nil, err)

	want := auth.Credential{Username: "user", Password: "token"}
	assert.Equal(t, nil, store.Put(context.Background(), "ghcr.io", want))

	got, err := store.Get(context.Background(), "ghcr.io")
	assert.Equal(t, nil, err)
	assert.Equal(t, want, got)

	// The credential is persisted as a plaintext `auths` entry, exactly
	// like `docker login` does without a credsStore.
	data, err := os.ReadFile(credFile)
	assert.Equal(t, nil, err)
	var cfg struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	assert.Equal(t, nil, json.Unmarshal(data, &cfg))
	assert.Contains(t, cfg.Auths, "ghcr.io")
}
