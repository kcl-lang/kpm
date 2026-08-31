package downloader

import (
	"context"
	"fmt"
	"path/filepath"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"

	kpmauth "kcl-lang.io/kpm/pkg/auth"
)

// CredStore is the store to get the credentials.
type CredStore struct {
	store        credentials.Store
	credFilePath string
}

// LoadCredentialFile loads the credential file and return the CredStore.
// credFilePath is the path to the ORAS credential file; the provider
// sidecar lives next to it as "providers.json".
func LoadCredentialFile(credFilePath string) (*CredStore, error) {
	store, err := credentials.NewStore(credFilePath, credentials.StoreOptions{})
	if err != nil {
		return nil, err
	}

	dockerStore, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err != nil {
		return nil, err
	}

	return &CredStore{
		store:        credentials.NewStoreWithFallbacks(store, dockerStore),
		credFilePath: credFilePath,
	}, nil
}

// GetAuthStore returns the auth store.
func (cred *CredStore) GetAuthStore() credentials.Store {
	return cred.store
}

// Credential will reture the credential info cache in CredStore
func (cred *CredStore) Credential(hostName string) (*auth.Credential, error) {
	if len(hostName) == 0 {
		return nil, fmt.Errorf("hostName is empty")
	}
	credential, err := cred.store.Get(context.Background(), hostName)
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

// Resolver returns an [auth.CredentialFunc] that consults the kpm
// provider sidecar first and falls back to the underlying ORAS store.
// It is the CredentialFunc ORAS should use when calling the OCI
// client: providers like GCP Workload Identity return short-lived
// (~1h) tokens, and ORAS re-enters this function on every 401, which
// is exactly when we want to mint a fresh token.
func (cred *CredStore) Resolver() auth.CredentialFunc {
	sidecar := kpmauth.OpenProviderStore(filepath.Join(filepath.Dir(cred.credFilePath), "providers.json"))
	r := kpmauth.NewResolver(sidecar, cred.store)
	return r.Resolve
}
