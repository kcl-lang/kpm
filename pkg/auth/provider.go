// Package auth provides pluggable credential providers for kpm.
//
// A Provider turns a registry hostname into an OCI credential that ORAS
// can use to authenticate. The package is designed so that adding a new
// cloud (AWS, Azure, ...) is just a new Provider implementation — the
// `kpm login` CLI logic stays the same.
//
// All providers must be safe to call from concurrent goroutines.
package auth

import (
	"context"
	"errors"

	"oras.land/oras-go/v2/registry/remote/auth"
)

// ErrCredential is the sentinel error wrapped by every Provider when it
// cannot mint a credential. Callers can use errors.Is to distinguish
// "provider could not get a credential" from "credential is wrong".
var ErrCredential = errors.New("kpm auth: cannot obtain credential")

// Provider turns a registry hostname into an OCI credential.
//
// Implementations MUST be safe to call from concurrent goroutines.
type Provider interface {
	// Name returns a short stable identifier used on the CLI flag
	// (e.g. "basic", "gcp"). It is also used as the auth key written
	// to the docker config.json so ORAS can pick it up again on
	// subsequent pulls.
	Name() string

	// Credential returns the credential for the given registry.
	// Errors should wrap ErrCredential so callers can distinguish
	// "cannot get credential" from "credential is wrong".
	Credential(ctx context.Context, registry string) (auth.Credential, error)
}

// ByName returns the Provider for a given name. Unknown names return an
// error wrapping ErrCredential so the CLI can surface a friendly
// "unknown provider" message.
func ByName(name string) (Provider, error) {
	switch name {
	case "basic", "":
		// Default to basic for backwards compatibility when no
		// provider is requested.
		return &BasicProvider{}, nil
	case "gcp":
		return &GCPProvider{}, nil
	default:
		return nil, errUnknownProvider(name)
	}
}

// KnownProviders returns the list of provider names known to kpm.
// Useful for help text and shell completion.
func KnownProviders() []string {
	return []string{"basic", "gcp"}
}

type unknownProviderError struct {
	name string
}

func (e *unknownProviderError) Error() string {
	return "kpm auth: unknown provider: " + e.name
}

// Unwrap so errors.Is(err, ErrUnknownProvider) works for callers that
// want to programmatically detect a bad provider name.
func (e *unknownProviderError) Unwrap() error { return ErrCredential }

func errUnknownProvider(name string) error {
	return &unknownProviderError{name: name}
}
