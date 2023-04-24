package oci

import (
	"context"
	"path/filepath"

	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/settings"
	"oras.land/oras-go/pkg/auth"
	dockerauth "oras.land/oras-go/pkg/auth/docker"
	remoteauth "oras.land/oras-go/v2/registry/remote/auth"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/retry"
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

const KCP_PKG_TAR = "*.tar"

// Pull will pull the oci atrifacts from oci registry to local path.
func Pull(localPath, hostName, repoName, tag string, settings *settings.Settings) (string, error) {
	// 0. Create a file store
	fs, err := file.New(localPath)
	if err != nil {
		return "", errors.FailedPullFromOci
	}
	defer fs.Close()

	// 1. Connect to a remote repository
	ctx := context.Background()
	repo, err := remote.NewRepository(filepath.Join(hostName, repoName))
	if err != nil {
		return "", errors.FailedPullFromOci
	}

	// 2. Login
	credential, err := loadCredential(hostName, settings)
	if err != nil {
		return "", errors.FailedPullFromOci
	}
	repo.Client = &remoteauth.Client{
		Client:     retry.DefaultClient,
		Cache:      remoteauth.DefaultCache,
		Credential: remoteauth.StaticCredential(repo.Reference.Host(), *credential),
	}

	// 3. Copy from the remote repository to the file store
	_, err = oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions)
	if err != nil {
		return "", errors.FailedPullFromOci
	}

	// 4.Get the (*.tar) file path.
	matches, err := filepath.Glob(filepath.Join(localPath, KCP_PKG_TAR))
	if err != nil && len(matches) != 1 {
		return "", errors.FailedPullFromOci
	}

	return matches[0], nil
}

func loadCredential(hostName string, settings *settings.Settings) (*remoteauth.Credential, error) {
	authClient, err := dockerauth.NewClientWithDockerFallback(settings.CredentialsFile)
	if err != nil {
		return nil, err
	}
	dockerClient, _ := authClient.(*dockerauth.Client)
	username, password, err := dockerClient.Credential(hostName)
	if err != nil {
		return nil, err
	}

	return &remoteauth.Credential{
		Username: username,
		Password: password,
	}, nil
}
