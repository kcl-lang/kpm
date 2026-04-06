package client

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogoutWithoutStoredCredential(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.NoError(t, err)

	kpmcli.settings.CredentialsFile = filepath.Join(t.TempDir(), "config.json")
	kpmcli.credsStore = nil

	err = kpmcli.LogoutOci("invalid_registry")
	assert.EqualError(t, err, "failed to logout 'invalid_registry'\nnot logged in\n")
}
