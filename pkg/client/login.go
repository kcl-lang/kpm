package client

import (
	"context"
	"fmt"

	"kcl-lang.io/kpm/pkg/reporter"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

// LoginOci will login to the oci registry.
func (c *KpmClient) LoginOci(hostname, username, password string) error {
	store, err := c.credentialStore()
	if err != nil {
		return err
	}

	cred := auth.Credential{
		Username: username,
		Password: password,
	}

	registry, err := remote.NewRegistry(hostname)
	if err != nil {
		return err
	}

	// The plain HTTP setting only controls the transport.
	defaultOciPlainHttp, forceOciPlainHttp := c.GetSettings().ForceOciPlainHttp()
	if defaultOciPlainHttp || forceOciPlainHttp {
		registry.PlainHTTP = true
	}

	err = credentials.Login(
		context.Background(),
		store,
		registry,
		cred,
	)

	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedLogin,
			err,
			fmt.Sprintf("failed to login '%s', please check registry, username and password is valid", hostname),
		)
	}

	return nil
}

// credentialStore returns the store used to persist registry credentials.
//
// Plaintext storage is always allowed: when the config file configures a
// credential helper (credsStore/credHelpers), oras-go delegates to it and
// plaintext is never touched; when no helper is available — the usual case
// on headless CI runners — the config file is the only backend, and
// refusing it makes `login` fail against every HTTPS registry with
// "putting plaintext credentials is disabled". This matches `docker login`,
// which writes a plaintext `auths` entry when no credsStore is set.
//
// Transport security is independent of the storage policy and remains
// governed by the plain HTTP settings.
// See https://github.com/kcl-lang/kpm/issues/769.
func (c *KpmClient) credentialStore() (*credentials.DynamicStore, error) {
	return credentials.NewStore(c.GetSettings().CredentialsFile, credentials.StoreOptions{
		AllowPlaintextPut: true,
	})
}
