package client

import (
	"fmt"

	"kcl-lang.io/kpm/pkg/reporter"
	"oras.land/oras-go/pkg/auth"
)

// LoginOci will login to the oci registry.
func (c *KpmClient) LoginOci(hostname, username, password string) error {
	credCli, err := c.GetCredsClient()
	if err != nil {
		return err
	}

	err = credCli.GetAuthClient().LoginWithOpts(
		[]auth.LoginOption{
			auth.WithLoginHostname(hostname),
			auth.WithLoginUsername(username),
			auth.WithLoginSecret(password),
		}...,
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
