package oci

import (
	"context"
	"path/filepath"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
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

const DEFAULT_OCI_ARTIFACT_TYPE = "application/vnd.oci.image.layer.v1.tar"

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

// Pull will pull the oci artifacts from oci registry to local path.
func Pull(localPath, hostName, repoName, tag string) error {
	// 0. Create a file store
	fs, err := file.New(localPath)
	if err != nil {
		return errors.FailedPullFromOci
	}
	defer fs.Close()

	// 1. Connect to a remote repository
	ctx := context.Background()
	repo, err := remote.NewRepository(filepath.Join(hostName, repoName))
	if err != nil {
		return errors.FailedPullFromOci
	}

	// 2. Login
	settings, err := settings.GetSettings()
	if err != nil {
		return err
	}
	credential, err := loadCredential(hostName, settings)
	if err != nil {
		return errors.FailedPullFromOci
	}
	repo.Client = &remoteauth.Client{
		Client:     retry.DefaultClient,
		Cache:      remoteauth.DefaultCache,
		Credential: remoteauth.StaticCredential(repo.Reference.Host(), *credential),
	}

	// 3. Copy from the remote repository to the file store
	_, err = oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions)
	if err != nil {
		return errors.FailedPullFromOci
	}

	return nil
}

// Push will push the oci artifacts to oci registry from local path
func Push(localPath, hostName, repoName, tag string, settings *settings.Settings) error {
	// 0. Create a file store
	fs, err := file.New(filepath.Dir(localPath))
	if err != nil {
		return err
	}
	defer fs.Close()
	ctx := context.Background()

	// 1. Add files to a file store

	fileNames := []string{localPath}
	fileDescriptors := make([]v1.Descriptor, 0, len(fileNames))
	for _, name := range fileNames {
		// The file name of the pushed file cannot be a file path,
		// If the file name is a path, the path will be created during pulling.
		// During pulling, a file should be downloaded separately,
		// and a file path is created for each download, which is not good.
		fileDescriptor, err := fs.Add(ctx, filepath.Base(name), DEFAULT_OCI_ARTIFACT_TYPE, "")
		if err != nil {
			return err
		}
		fileDescriptors = append(fileDescriptors, fileDescriptor)
	}

	// 2. Pack the files and tag the packed manifest
	manifestDescriptor, err := oras.Pack(ctx, fs, DEFAULT_OCI_ARTIFACT_TYPE, fileDescriptors, oras.PackOptions{
		PackImageManifest: true,
	})
	if err != nil {
		return err
	}

	if err = fs.Tag(ctx, manifestDescriptor, tag); err != nil {
		return err
	}

	// 3. Connect to a remote repository

	repo, err := remote.NewRepository(filepath.Join(hostName, repoName))
	if err != nil {
		panic(err)
	}

	// 4. Login
	credential, err := loadCredential(hostName, settings)
	if err != nil {
		return errors.FailedPushToOci
	}
	repo.Client = &remoteauth.Client{
		Client:     retry.DefaultClient,
		Cache:      remoteauth.DefaultCache,
		Credential: remoteauth.StaticCredential(repo.Reference.Host(), *credential),
	}

	// 5. Copy from the file store to the remote repository
	_, err = oras.Copy(ctx, fs, tag, repo, tag, oras.DefaultCopyOptions)
	return err
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
