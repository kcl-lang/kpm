package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicProvider_Name(t *testing.T) {
	p := &BasicProvider{Username: "u", Password: "p"}
	assert.Equal(t, "basic", p.Name())
}

func TestBasicProvider_Credential(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		registry string
	}{
		{
			name:     "username and password",
			username: "alice",
			password: "secret",
			registry: "ghcr.io",
		},
		{
			name:     "empty credentials (anonymous pull)",
			username: "",
			password: "",
			registry: "registry.example.com",
		},
		{
			name:     "registry is ignored",
			username: "bob",
			password: "hunter2",
			registry: "gcr.io/my-project",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &BasicProvider{Username: tc.username, Password: tc.password}
			cred, err := p.Credential(context.Background(), tc.registry)
			assert.NoError(t, err)
			assert.Equal(t, tc.username, cred.Username)
			assert.Equal(t, tc.password, cred.Password)
		})
	}
}

func TestByName_Basic(t *testing.T) {
	p, err := ByName("basic")
	assert.NoError(t, err)
	assert.Equal(t, "basic", p.Name())
}

func TestByName_Default(t *testing.T) {
	// Empty name falls back to basic for backwards compatibility.
	p, err := ByName("")
	assert.NoError(t, err)
	assert.Equal(t, "basic", p.Name())
}

func TestByName_GCP(t *testing.T) {
	p, err := ByName("gcp")
	assert.NoError(t, err)
	assert.Equal(t, "gcp", p.Name())
}

func TestByName_Unknown(t *testing.T) {
	_, err := ByName("aws")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCredential),
		"unknown provider should wrap ErrCredential, got %v", err)
}

func TestKnownProviders(t *testing.T) {
	names := KnownProviders()
	assert.Contains(t, names, "basic")
	assert.Contains(t, names, "gcp")
}
