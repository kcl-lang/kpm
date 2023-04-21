package oci

import (
	"context"

	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/settings"
	"oras.land/oras-go/pkg/auth"
	dockerauth "oras.land/oras-go/pkg/auth/docker"
)

// Login will login 'hostname' by 'username' and 'password'.
func Login(hostname, username, password string, setting *settings.Settings) error {

	authClient, err := dockerauth.NewClientWithDockerFallback(setting.CredentialsFile)

	if err != nil {
		return errors.FailedLogin
	}

	err = authClient.LoginWithOpts(
		[]auth.LoginOption{
			auth.WithLoginHostname(hostname),
			auth.WithLoginUsername(username),
			auth.WithLoginSecret(password),
		}...,
	)

	if err != nil {
		return errors.FailedLogin
	}

	reporter.Report("kpm: Login Succeeded")
	return nil
}

// Logout will logout from registry.
func Logout(hostname string, setting *settings.Settings) error {

	authClient, err := dockerauth.NewClientWithDockerFallback(setting.CredentialsFile)

	if err != nil {
		return errors.FailedLogout
	}

	err = authClient.Logout(context.Background(), hostname)

	if err != nil {
		return errors.FailedLogout
	}

	reporter.Report("kpm: Logout Succeeded")
	return nil
}
