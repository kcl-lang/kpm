package oci

import (
	"context"
	"fmt"
	"path/filepath"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/thoas/go-funk"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/semver"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
	"oras.land/oras-go/pkg/auth"
	dockerauth "oras.land/oras-go/pkg/auth/docker"
	remoteauth "oras.land/oras-go/v2/registry/remote/auth"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/errcode"
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
func NewOciClient(regName, repoName string) (*OciClient, *reporter.KpmEvent) {
	repoPath := utils.JoinPath(regName, repoName)
	repo, err := remote.NewRepository(repoPath)

	if err != nil {
		return nil, reporter.NewErrorEvent(
			reporter.RepoNotFound,
			err,
			fmt.Sprintf("repository '%s' not found.", repoPath),
		)
	}
	ctx := context.Background()

	settings := settings.GetSettings()
	if err != nil {
		return nil, settings.ErrorEvent
	}
	repo.PlainHTTP = settings.DefaultOciPlainHttp()

	// Login
	credential, err := loadCredential(regName, settings)
	if err != nil {
		return nil, reporter.NewErrorEvent(
			reporter.FailedLoadCredential,
			err,
			fmt.Sprintf("failed to load credential for '%s' from '%s'.", regName, settings.CredentialsFile),
		)
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
func (ociClient *OciClient) Pull(localPath, tag string) *reporter.KpmEvent {
	// Create a file store
	fs, err := file.New(localPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedCreateStorePath, err, "Failed to create store path ", localPath)
	}
	defer fs.Close()

	// Copy from the remote repository to the file store
	_, err = oras.Copy(*ociClient.ctx, ociClient.repo, tag, fs, tag, oras.DefaultCopyOptions)
	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedGetPkg,
			err,
			fmt.Sprintf("failed to get package with '%s' from '%s'.", tag, ociClient.repo.Reference.String()),
		)
	}

	return nil
}

// TheLatestTag will return the latest tag of the kcl packages.
func (ociClient *OciClient) TheLatestTag() (string, *reporter.KpmEvent) {
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
		return "", reporter.NewErrorEvent(
			reporter.FailedSelectLatestVersion,
			err,
			fmt.Sprintf("failed to select latest version from '%s'.", ociClient.repo.Reference.String()),
		)
	}

	return tagSelected, nil
}

// ContainsTag will check if the tag exists in the repo.
func (ociClient *OciClient) ContainsTag(tag string) (bool, *reporter.KpmEvent) {
	var exists bool

	err := ociClient.repo.Tags(*ociClient.ctx, "", func(tags []string) error {
		exists = funk.ContainsString(tags, tag)
		return nil
	})

	if err != nil {
		// If the repo with tag is not found, return false.
		errRes, ok := err.(*errcode.ErrorResponse)
		if ok {
			if len(errRes.Errors) == 1 && errRes.Errors[0].Code == errcode.ErrorCodeNameUnknown {
				return false, nil
			}
		}
		// If the user not login, return error.
		return false, reporter.NewErrorEvent(
			reporter.FailedGetPackageVersions,
			err,
			fmt.Sprintf("failed to access '%s'", ociClient.repo.Reference.String()),
		)
	}

	return exists, nil
}

// Push will push the oci artifacts to oci registry from local path
func (ociClient *OciClient) Push(localPath, tag string) *reporter.KpmEvent {
	// 0. Create a file store
	fs, err := file.New(filepath.Dir(localPath))
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedPush, err, "Failed to load store path ", localPath)
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
			return reporter.NewErrorEvent(reporter.FailedPush, err, fmt.Sprintf("Failed to add file '%s'", name))
		}
		fileDescriptors = append(fileDescriptors, fileDescriptor)
	}

	// 2. Pack the files and tag the packed manifest
	manifestDescriptor, err := oras.Pack(*ociClient.ctx, fs, DEFAULT_OCI_ARTIFACT_TYPE, fileDescriptors, oras.PackOptions{
		PackImageManifest: true,
	})
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedPush, err, fmt.Sprintf("Failed to pack package in '%s'", localPath))
	}

	if err = fs.Tag(*ociClient.ctx, manifestDescriptor, tag); err != nil {
		return reporter.NewErrorEvent(reporter.FailedPush, err, fmt.Sprintf("Failed to tag package with tag '%s'", tag))
	}

	// 3. Copy from the file store to the remote repository
	desc, err := oras.Copy(*ociClient.ctx, fs, tag, ociClient.repo, tag, oras.DefaultCopyOptions)

	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedPush, err, fmt.Sprintf("Failed to push '%s'", ociClient.repo.Reference))
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
func Pull(localPath, hostName, repoName, tag string) *reporter.KpmEvent {
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
		reporter.ReportEventToStdout(
			reporter.NewEvent(reporter.SelectLatestVersion, "the lastest version '", tagSelected, "' will be pulled."),
		)
	} else {
		tagSelected = tag
	}

	reporter.ReportEventToStdout(
		reporter.NewEvent(
			reporter.Pulling,
			fmt.Sprintf("pulling '%s:%s' from '%s'.", repoName, tagSelected, utils.JoinPath(hostName, repoName)),
		),
	)
	return ociClient.Pull(localPath, tagSelected)
}

// Push will push the oci artifacts to oci registry from local path
func Push(localPath, hostName, repoName, tag string, settings *settings.Settings) *reporter.KpmEvent {
	// Create an oci client.
	ociClient, err := NewOciClient(hostName, repoName)
	if err != nil {
		return err
	}

	exist, err := ociClient.ContainsTag(tag)
	if err != (*reporter.KpmEvent)(nil) {
		return err
	}

	if exist {
		return reporter.NewErrorEvent(
			reporter.PkgTagExists,
			fmt.Errorf("package version '%s' already exists", tag),
		)
	}

	// Push the oci package by the oci client.
	return ociClient.Push(localPath, tag)
}
