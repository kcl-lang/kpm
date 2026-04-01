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
	// Allow plaintext credentials for plain HTTP registries
	defaultOciPlainHttp, forceOciPlainHttp := c.GetSettings().ForceOciPlainHttp()
	allowPlaintext := false
	if defaultOciPlainHttp || forceOciPlainHttp {
		allowPlaintext = true
	}

	store, err := credentials.NewStore(c.GetSettings().CredentialsFile, credentials.StoreOptions{
		AllowPlaintextPut: allowPlaintext,
	})
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
