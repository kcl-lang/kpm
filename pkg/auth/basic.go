package auth

import (
	"context"

	"oras.land/oras-go/v2/registry/remote/auth"
)

// BasicProvider is the default provider used by `kpm login`. It simply
// hands back the username/password it was constructed with. It exists
// so that the existing username/password flow is just another Provider
// implementation — no special-casing in the login command.
//
// Concurrency: the zero value is safe to use from multiple goroutines
// because Username and Password are immutable after construction.
type BasicProvider struct {
	// Username is the registry username.
	Username string
	// Password is the registry password or identity token.
	Password string
}

// Name returns "basic".
func (p *BasicProvider) Name() string { return "basic" }

// Credential returns the username/password pair the provider was
// constructed with. The registry argument is ignored — BasicProvider
// does not vary its credential by host.
func (p *BasicProvider) Credential(_ context.Context, _ string) (auth.Credential, error) {
	return auth.Credential{
		Username: p.Username,
		Password: p.Password,
	}, nil
}
