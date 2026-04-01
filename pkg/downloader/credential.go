package downloader

import (
	"context"
	"fmt"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

// CredStore is the store to get the credentials.
type CredStore struct {
	store credentials.Store
}

// LoadCredentialFile loads the credential file and return the CredStore.
func LoadCredentialFile(filepath string) (*CredStore, error) {
	store, err := credentials.NewStore(filepath, credentials.StoreOptions{})
	if err != nil {
		return nil, err
	}

	return &CredStore{
		store: store,
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
