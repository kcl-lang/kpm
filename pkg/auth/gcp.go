package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/compute/metadata"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// gcpAccessTokenUsername is the magic username GCR / Artifact Registry
// expect when handed a Bearer OAuth token. See:
// https://cloud.google.com/container/registry/authentication#token
const gcpAccessTokenUsername = "oauth2accesstoken"

// GCPProvider authenticates to Google Container Registry / Artifact
// Registry using a GCP access token minted from the GCE/GKE metadata
// server. This is how Workload Identity surfaces a federated identity
// to a workload running inside a GKE pod — no static credential
// required.
//
// Typical usage from a GKE pod with Workload Identity bound:
//
//	kpm login gcr.io/my-project --provider=gcp
//
// GCPProvider does not cache tokens; tokens have a ~1h TTL so callers
// should fetch a fresh credential before each OCI push/pull. For
// login this is fine — we just write the token at login time.
type GCPProvider struct {
	// Audience used when exchanging the metadata token for an OAuth
	// access token scoped to the registry. Empty means default
	// audience "https://<registry>".
	Audience string

	// metadataClient is the GCE metadata client. nil means use the
	// package default (169.254.169.254). Tests override this so the
	// client points at a httptest.Server.
	metadataClient *metadata.Client
}

// Name returns "gcp".
func (p *GCPProvider) Name() string { return "gcp" }

// Credential returns a Bearer-token credential for the given registry.
// It will:
//
//  1. Confirm we are running on GCE/GKE by querying the metadata
//     server for an instance id (returns 404 outside the metadata
//     server).
//  2. Mint an OAuth2 access token scoped to the requested audience.
//  3. Parse the JSON response and return it as auth.Credential with
//     the magic "oauth2accesstoken" username.
//
// All errors wrap ErrCredential so callers can distinguish "not on
// GCE" from "credential is wrong".
func (p *GCPProvider) Credential(ctx context.Context, registry string) (auth.Credential, error) {
	mc := p.metadataClient
	if mc == nil {
		mc = metadata.NewClient(nil)
	}

	audience := p.Audience
	if audience == "" {
		audience = "https://" + registry
	}

	// 1. Confirm we're on GCE / GKE (returns 404 outside the metadata server).
	// We call GetWithContext directly instead of InstanceIDWithContext
	// because the latter caches the result in a package-level
	// variable, which would leak between tests and across multiple
	// Credential() calls in the same process.
	if _, err := mc.GetWithContext(ctx, "instance/id"); err != nil {
		return auth.Credential{}, fmt.Errorf("kpm auth: not running on GCE/GKE: %w: %w", ErrCredential, err)
	}

	// 2. Mint an OAuth2 access token bound to the requested audience.
	tok, err := mc.GetWithContext(ctx, fmt.Sprintf("instance/service-accounts/default/token?audience=%s", audience))
	if err != nil {
		return auth.Credential{}, fmt.Errorf("kpm auth: fetch GCP access token: %w: %w", ErrCredential, err)
	}

	// 3. Parse JSON: {"access_token":"ya29.…","expires_in":3599,"token_type":"Bearer"}
	var t struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(tok), &t); err != nil {
		return auth.Credential{}, fmt.Errorf("kpm auth: parse GCP token response: %w: %w", ErrCredential, err)
	}
	if t.AccessToken == "" {
		return auth.Credential{}, fmt.Errorf("kpm auth: empty GCP access token: %w", ErrCredential)
	}

	return auth.Credential{
		Username: gcpAccessTokenUsername,
		Password: t.AccessToken,
	}, nil
}
