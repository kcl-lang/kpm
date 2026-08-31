package auth

import (
	"context"
	"fmt"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

// Resolver turns a registry hostname into an [auth.Credential] by first
// consulting a ProviderStore (so non-credential flows like GCP Workload
// Identity can mint fresh tokens on every call) and then falling back to
// the ORAS credential store.
//
// Resolver is the [auth.CredentialFunc] ORAS uses inside the OCI client,
// so it is invoked on every 401 retry — which is exactly when providers
// that return short-lived tokens (GCP OAuth tokens, ~1h TTL) want to
// mint a fresh credential.
type Resolver struct {
	// Providers maps host → provider name. nil is treated as an empty
	// store (everything falls through to fallback).
	Providers *ProviderStore
	// Fallback is the ORAS credential store consulted when no provider
	// is configured for the host. Must not be nil.
	Fallback credentials.Store
}

// NewResolver builds a Resolver from the given sidecar path and ORAS
// fallback store. It is the typical wiring inside the OCI client.
func NewResolver(providers *ProviderStore, fallback credentials.Store) *Resolver {
	return &Resolver{
		Providers: providers,
		Fallback:  fallback,
	}
}

// Resolve implements [auth.CredentialFunc]. ORAS calls it with the host
// of the request; we route through the configured provider when one is
// registered, otherwise fall back to the ORAS credential store.
//
// The returned credential is whatever the provider or the ORAS store
// gives us — the caller (ORAS) handles caching the resulting Bearer
// token for the lifetime of the process. Provider implementations
// therefore do not need their own TTL cache: every 401 that survives
// the ORAS cache will re-enter this function and trigger a fresh mint.
func (r *Resolver) Resolve(ctx context.Context, host string) (auth.Credential, error) {
	if r == nil {
		return auth.EmptyCredential, nil
	}
	if r.Providers != nil {
		name, err := r.Providers.Get(host)
		if err != nil {
			return auth.Credential{}, fmt.Errorf("kpm auth: read provider store for %q: %w", host, err)
		}
		if name != "" {
			p, err := ByName(name)
			if err != nil {
				return auth.Credential{}, fmt.Errorf("kpm auth: host %q mapped to provider %q: %w", host, name, err)
			}
			return p.Credential(ctx, host)
		}
	}
	if r.Fallback == nil {
		return auth.EmptyCredential, nil
	}
	cred, err := r.Fallback.Get(ctx, host)
	if err != nil {
		return auth.Credential{}, err
	}
	return cred, nil
}

// AsCredentialFunc returns the Resolver as an [auth.CredentialFunc] so
// callers can pass it directly to ORAS. Mostly a convenience for tests.
func (r *Resolver) AsCredentialFunc() auth.CredentialFunc {
	return r.Resolve
}
