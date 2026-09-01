package client

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/mock"
	"kcl-lang.io/kpm/pkg/settings"
	remoteauth "oras.land/oras-go/v2/registry/remote/auth"
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

// Regression test for #769: storing login credentials must work with the
// default HTTPS transport (plain HTTP off). Previously the credential
// store only allowed plaintext puts when OCI_REG_PLAIN_HTTP was on,
// which made `kpm registry login` fail with "putting plaintext
// credentials is disabled" on headless CI runners without a credential
// helper, against HTTPS-only registries such as ghcr.io and docker.io.
func TestLoginOciStoresCredentialsWithoutPlainHttp(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "config.json")

	// Seed an existing auths entry so ORAS does not auto-detect a
	// platform credential helper (which would depend on the host
	// machine's setup and make the test non-deterministic).
	seed := `{"auths":{"registry.example.com":{"auth":"dXNlcjpwYXNz"}}}`
	assert.NoError(t, os.WriteFile(credFile, []byte(seed), 0o600))

	kpmcli := &KpmClient{
		settings: settings.Settings{CredentialsFile: credFile},
	}

	// The default: HTTPS transport, no forced plain HTTP.
	def, forced := kpmcli.GetSettings().ForceOciPlainHttp()
	assert.False(t, def)
	assert.False(t, forced)

	store, err := kpmcli.credentialStore()
	assert.NoError(t, err)

	// Before the fix this failed with ErrPlaintextPutDisabled.
	err = store.Put(context.Background(), "ghcr.io", remoteauth.Credential{
		Username: "user",
		Password: "token",
	})
	assert.NoError(t, err)

	// The credential is persisted and read back from the plain-text
	// config file.
	cred, err := store.Get(context.Background(), "ghcr.io")
	assert.NoError(t, err)
	assert.Equal(t, "user", cred.Username)
	assert.Equal(t, "token", cred.Password)
}
