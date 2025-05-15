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

	opts := []auth.LoginOption{
		auth.WithLoginHostname(hostname),
		auth.WithLoginUsername(username),
		auth.WithLoginSecret(password),
	}

	defaultOciPlainHttp, forceOciPlainHttp := c.GetSettings().ForceOciPlainHttp()

	if defaultOciPlainHttp || forceOciPlainHttp {
		opts = append(opts, auth.WithLoginInsecure())
	}

	err = credCli.GetAuthClient().LoginWithOpts(
		opts...,
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
