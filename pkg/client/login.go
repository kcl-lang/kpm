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
	defaultOciPlainHttp, forceOciPlainHttp := c.GetSettings().ForceOciPlainHttp()

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

	// Handle plain HTTP setting
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

// credentialStore returns the ORAS credential store that LoginOci
// persists credentials through.
//
// AllowPlaintextPut is enabled unconditionally. In ORAS the plain-text
// config file is only the last-resort backend: server-specific
// credential helpers (credHelpers) and the credentials store (credsStore)
// always take precedence, and the flag merely allows the fallback when no
// helper is configured — the same behaviour as the Docker and ORAS CLIs.
// Gating it on the plain-HTTP setting made `kpm registry login` fail
// with "putting plaintext credentials is disabled" on headless CI
// runners (no keyring) against HTTPS-only registries such as ghcr.io and
// docker.io, because that flag also forces plain-HTTP transport.
// See #769.
func (c *KpmClient) credentialStore() (credentials.Store, error) {
	return credentials.NewStore(c.GetSettings().CredentialsFile, credentials.StoreOptions{
		AllowPlaintextPut: true,
	})
}
