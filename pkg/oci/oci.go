package oci

import (
	"context"
	"net/url"
	"path/filepath"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/semver"
	"kcl-lang.io/kpm/pkg/settings"
	"oras.land/oras-go/pkg/auth"
	dockerauth "oras.land/oras-go/pkg/auth/docker"
	remoteauth "oras.land/oras-go/v2/registry/remote/auth"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const OCI_SCHEME = "oci"
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

// OciClient is mainly responsible for interacting with OCI registry
type OciClient struct {
	repo *remote.Repository
	ctx  *context.Context
}

// NewOciClient will new an OciClient.
// regName is the registry. e.g. ghcr.io or docker.io.
// repoName is the repo name on registry.
func NewOciClient(regName, repoName string) (*OciClient, error) {

	repoPath, err := url.JoinPath(regName, repoName)
	if err != nil {
		return nil, err
	}
	repo, err := remote.NewRepository(repoPath)
	if err != nil {
		return nil, errors.FailedPullFromOci
	}
	ctx := context.Background()

	settings, err := settings.GetSettings()
	if err != nil {
		return nil, err
	}

	// Login
	credential, err := loadCredential(regName, settings)
	if err != nil {
		return nil, errors.FailedPushToOci
	}
	repo.Client = &remoteauth.Client{
		Client:     retry.DefaultClient,
		Cache:      remoteauth.DefaultCache,
		Credential: remoteauth.StaticCredential(repo.Reference.Host(), *credential),
	}

	return &OciClient{
		repo: repo,
		ctx:  &ctx,
	}, nil
}

// Pull will pull the oci artifacts from oci registry to local path.
func (ociClient *OciClient) Pull(localPath, tag string) error {
	// Create a file store
	fs, err := file.New(localPath)
	if err != nil {
		return errors.FailedPullFromOci
	}
	defer fs.Close()

	// Copy from the remote repository to the file store
	_, err = oras.Copy(*ociClient.ctx, ociClient.repo, tag, fs, tag, oras.DefaultCopyOptions)
	if err != nil {
		return errors.FailedPullFromOci
	}

	return nil
}

// TheLatestTag will return the latest tag of the kcl packages.
func (ociClient *OciClient) TheLatestTag() (string, error) {
	var tagSelected string

	err := ociClient.repo.Tags(*ociClient.ctx, "", func(tags []string) error {
		var err error
		tagSelected, err = semver.LatestVersion(tags)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", errors.FailedDownloadError
	}

	return tagSelected, nil
}

// Push will push the oci artifacts to oci registry from local path
func (ociClient *OciClient) Push(localPath, tag string) error {
	// 0. Create a file store
	fs, err := file.New(filepath.Dir(localPath))
	if err != nil {
		return err
	}
	defer fs.Close()

	// 1. Add files to a file store

	fileNames := []string{localPath}
	fileDescriptors := make([]v1.Descriptor, 0, len(fileNames))
	for _, name := range fileNames {
		// The file name of the pushed file cannot be a file path,
		// If the file name is a path, the path will be created during pulling.
		// During pulling, a file should be downloaded separately,
		// and a file path is created for each download, which is not good.
		fileDescriptor, err := fs.Add(*ociClient.ctx, filepath.Base(name), DEFAULT_OCI_ARTIFACT_TYPE, "")
		if err != nil {
			return err
		}
		fileDescriptors = append(fileDescriptors, fileDescriptor)
	}

	// 2. Pack the files and tag the packed manifest
	manifestDescriptor, err := oras.Pack(*ociClient.ctx, fs, DEFAULT_OCI_ARTIFACT_TYPE, fileDescriptors, oras.PackOptions{
		PackImageManifest: true,
	})
	if err != nil {
		return err
	}

	if err = fs.Tag(*ociClient.ctx, manifestDescriptor, tag); err != nil {
		return err
	}

	// 3. Copy from the file store to the remote repository
	desc, err := oras.Copy(*ociClient.ctx, fs, tag, ociClient.repo, tag, oras.DefaultCopyOptions)

	if err != nil {
		return err
	}

	reporter.Report("kpm: pushed [registry]", ociClient.repo.Reference)
	reporter.Report("kpm: digest:", desc.Digest)
	return nil
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

// Pull will pull the oci artifacts from oci registry to local path.
func Pull(localPath, hostName, repoName, tag string) error {
	ociClient, err := NewOciClient(hostName, repoName)
	if err != nil {
		return err
	}

	var tagSelected string
	if len(tag) == 0 {
		tagSelected, err = ociClient.TheLatestTag()
		if err != nil {
			return err
		}
		reporter.Report("kpm: the lastest version", tagSelected, "will be pulled.")
	} else {
		tagSelected = tag
	}

	return ociClient.Pull(localPath, tagSelected)
}

// Push will push the oci artifacts to oci registry from local path
func Push(localPath, hostName, repoName, tag string, settings *settings.Settings) error {
	// Create an oci client.
	ociClient, err := NewOciClient(hostName, repoName)
	if err != nil {
		return err
	}

	// Push the oci package by the oci client.
	return ociClient.Push(localPath, tag)
}
