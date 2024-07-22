package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/thoas/go-funk"
	"oras.land/oras-go/pkg/auth"
	dockerauth "oras.land/oras-go/pkg/auth/docker"
	remoteauth "oras.land/oras-go/v2/registry/remote/auth"

	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/semver"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/errcode"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const OCI_SCHEME = "oci"
const DEFAULT_OCI_ARTIFACT_TYPE = "application/vnd.oci.image.layer.v1.tar"
const (
	OciErrorCodeNameUnknown  = "NAME_UNKNOWN"
	OciErrorCodeRepoNotFound = "NOT_FOUND"
)

// Login will login 'hostname' by 'username' and 'password'.
func Login(hostname, username, password string, setting *settings.Settings) error {

	authClient, err := dockerauth.NewClientWithDockerFallback(setting.CredentialsFile)

	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedLogin,
			err,
			fmt.Sprintf("failed to login '%s', please check registry, username and password is valid", hostname),
		)
	}

	err = authClient.LoginWithOpts(
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

// Logout will logout from registry.
func Logout(hostname string, setting *settings.Settings) error {

	authClient, err := dockerauth.NewClientWithDockerFallback(setting.CredentialsFile)

	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedLogout, err, fmt.Sprintf("failed to logout '%s'", hostname))
	}

	err = authClient.Logout(context.Background(), hostname)

	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedLogout, err, fmt.Sprintf("failed to logout '%s'", hostname))
	}

	return nil
}

// OciClient is mainly responsible for interacting with OCI registry
type OciClient struct {
	repo           *remote.Repository
	ctx            *context.Context
	logWriter      io.Writer
	settings       *settings.Settings
	cred           *remoteauth.Credential
	PullOciOptions *PullOciOptions
}

// OciClientOption configures how we set up the OciClient
type OciClientOption func(*OciClient) error

// WithSettings sets the kpm settings of the OciClient
func WithSettings(settings *settings.Settings) OciClientOption {
	return func(c *OciClient) error {
		c.settings = settings
		return nil
	}
}

// WithRepoPath sets the repo path of the OciClient
func WithRepoPath(repoPath string) OciClientOption {
	return func(c *OciClient) error {
		var err error
		c.repo, err = remote.NewRepository(repoPath)
		if err != nil {
			return fmt.Errorf("repository '%s' not found", repoPath)
		}
		return nil
	}
}

// WithCredential sets the credential of the OciClient
func WithCredential(credential *remoteauth.Credential) OciClientOption {
	return func(c *OciClient) error {
		c.cred = credential
		return nil
	}
}

// WithPlainHttp sets the plain http of the OciClient
func WithPlainHttp(plainHttp bool) OciClientOption {
	return func(c *OciClient) error {
		if c.repo == nil {
			return fmt.Errorf("repo is nil")
		}
		c.repo.PlainHTTP = plainHttp
		return nil
	}
}

type PullOciOptions struct {
	Platform string
	CopyOpts *oras.CopyOptions
}

func (ociClient *OciClient) SetLogWriter(writer io.Writer) {
	ociClient.logWriter = writer
}

func (ociClient *OciClient) GetReference() string {
	return ociClient.repo.Reference.String()
}

// NewOciClientWithOpts will new an OciClient with options.
func NewOciClientWithOpts(opts ...OciClientOption) (*OciClient, error) {
	client := &OciClient{}
	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	client.repo.Client = &remoteauth.Client{
		Client:     retry.DefaultClient,
		Cache:      remoteauth.DefaultCache,
		Credential: remoteauth.StaticCredential(client.repo.Reference.Host(), *client.cred),
	}

	isPlainHttp, force := client.settings.ForceOciPlainHttp()
	if force {
		client.repo.PlainHTTP = isPlainHttp
	} else {
		registry := client.repo.Reference.String()
		host, _, _ := net.SplitHostPort(registry)
		if host == "localhost" || registry == "localhost" {
			// not specified, defaults to plain http for localhost
			client.repo.PlainHTTP = true
		}
	}

	client.ctx = &ctx
	client.PullOciOptions = &PullOciOptions{
		CopyOpts: &oras.CopyOptions{
			CopyGraphOptions: oras.CopyGraphOptions{
				MaxMetadataBytes: DEFAULT_LIMIT_STORE_SIZE, // default is 64 MiB
			},
		},
	}

	return client, nil
}

// NewOciClient will new an OciClient.
// regName is the registry. e.g. ghcr.io or docker.io.
// repoName is the repo name on registry.
// Deprecated: use NewOciClientWithOpts instead.
func NewOciClient(regName, repoName string, settings *settings.Settings) (*OciClient, error) {
	// Login
	credential, err := loadCredential(regName, settings)
	if err != nil {
		return nil, reporter.NewErrorEvent(
			reporter.FailedLoadCredential,
			err,
			fmt.Sprintf("failed to load credential for '%s' from '%s'.", regName, settings.CredentialsFile),
		)
	}

	return NewOciClientWithOpts(
		WithRepoPath(utils.JoinPath(regName, repoName)),
		WithCredential(credential),
		WithSettings(settings),
	)
}

// The default limit of the store size is 64 MiB.
const DEFAULT_LIMIT_STORE_SIZE = 64 * 1024 * 1024

// Pull will pull the oci artifacts from oci registry to local path.
func (ociClient *OciClient) Pull(localPath, tag string) error {
	// Create a file store
	fs, err := file.NewWithFallbackLimit(localPath, DEFAULT_LIMIT_STORE_SIZE)
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedCreateStorePath, err, "Failed to create store path ", localPath)
	}
	defer fs.Close()
	copyOpts := ociClient.PullOciOptions.CopyOpts
	copyOpts.FindSuccessors = ociClient.PullOciOptions.Successors
	_, err = oras.Copy(*ociClient.ctx, ociClient.repo, tag, fs, tag, *copyOpts)
	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedGetPkg,
			err,
			fmt.Sprintf("failed to get package with '%s' from '%s'", tag, ociClient.repo.Reference.String()),
		)
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
		return "", reporter.NewErrorEvent(
			reporter.FailedSelectLatestVersion,
			err,
			fmt.Sprintf("failed to select latest version from '%s'", ociClient.repo.Reference.String()),
		)
	}

	return tagSelected, nil
}

// RepoIsNotExist will check if the error is caused by the repo not found.
func RepoIsNotExist(err error) bool {
	errRes, ok := err.(*errcode.ErrorResponse)
	if ok {
		if len(errRes.Errors) == 1 &&
			// docker.io and gchr.io will return NAME_UNKNOWN
			(errRes.Errors[0].Code == OciErrorCodeNameUnknown ||
				// harbor will return NOT_FOUND
				errRes.Errors[0].Code == OciErrorCodeRepoNotFound) {
			return true
		}
	}
	return false
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
		if RepoIsNotExist(err) {
			return false, nil
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
	return ociClient.PushWithOciManifest(localPath, tag, &opt.OciManifestOptions{})
}

// PushWithManifest will push the oci artifacts to oci registry from local path
func (ociClient *OciClient) PushWithOciManifest(localPath, tag string, opts *opt.OciManifestOptions) *reporter.KpmEvent {
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

	// 2. Pack the files, tag the packed manifest and add metadata as annotations
	packOpts := oras.PackManifestOptions{
		ManifestAnnotations: opts.Annotations,
		Layers:              fileDescriptors,
	}
	manifestDescriptor, err := oras.PackManifest(*ociClient.ctx, fs, oras.PackManifestVersion1_1_RC4, DEFAULT_OCI_ARTIFACT_TYPE, packOpts)

	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedPush, err, fmt.Sprintf("failed to pack package in '%s'", localPath))
	}

	if err = fs.Tag(*ociClient.ctx, manifestDescriptor, tag); err != nil {
		return reporter.NewErrorEvent(reporter.FailedPush, err, fmt.Sprintf("failed to tag package with tag '%s'", tag))
	}

	// 3. Copy from the file store to the remote repository
	desc, err := oras.Copy(*ociClient.ctx, fs, tag, ociClient.repo, tag, oras.DefaultCopyOptions)

	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedPush, err, fmt.Sprintf("failed to push '%s'", ociClient.repo.Reference))
	}

	reporter.ReportMsgTo(fmt.Sprintf("pushed [registry] %s", ociClient.repo.Reference), ociClient.logWriter)
	reporter.ReportMsgTo(fmt.Sprintf("digest: %s", desc.Digest), ociClient.logWriter)
	return nil
}

// FetchManifestIntoJsonStr will fetch the manifest and return it into json string.
func (ociClient *OciClient) FetchManifestIntoJsonStr(opts opt.OciFetchOptions) (string, error) {
	fetchOpts := opts.FetchBytesOptions
	_, manifestContent, err := oras.FetchBytes(*ociClient.ctx, ociClient.repo, opts.Tag, fetchOpts)
	if err != nil {
		return "", err
	}

	return string(manifestContent), nil
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
func Pull(localPath, hostName, repoName, tag string, settings *settings.Settings) error {
	ociClient, err := NewOciClient(hostName, repoName, settings)
	if err != nil {
		return err
	}

	var tagSelected string
	if len(tag) == 0 {
		tagSelected, err = ociClient.TheLatestTag()
		if err != nil {
			return err
		}
		reporter.ReportMsgTo(
			fmt.Sprintf("the lastest version '%s' will be pulled", tagSelected),
			os.Stdout,
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
func Push(localPath, hostName, repoName, tag string, settings *settings.Settings) error {
	// Create an oci client.
	ociClient, err := NewOciClient(hostName, repoName, settings)
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

func GetAllImageTags(imageName string) ([]string, error) {
	sysCtx := &types.SystemContext{}
	ref, err := docker.ParseReference("//" + strings.TrimPrefix(imageName, "oci://"))
	if err != nil {
		log.Fatalf("Error parsing reference: %v", err)
	}

	tags, err := docker.GetRepositoryTags(context.Background(), sysCtx, ref)
	if err != nil {
		log.Fatalf("Error getting tags: %v", err)
	}
	return tags, nil
}

const (
	MediaTypeConfig           = "application/vnd.docker.container.image.v1+json"
	MediaTypeManifestList     = "application/vnd.docker.distribution.manifest.list.v2+json"
	MediaTypeManifest         = "application/vnd.docker.distribution.manifest.v2+json"
	MediaTypeForeignLayer     = "application/vnd.docker.image.rootfs.foreign.diff.tar.gzip"
	MediaTypeArtifactManifest = "application/vnd.oci.artifact.manifest.v1+json"
)

// Successors returns the nodes directly pointed by the current node.
// In other words, returns the "children" of the current descriptor.
func (popts *PullOciOptions) Successors(ctx context.Context, fetcher content.Fetcher, node v1.Descriptor) ([]v1.Descriptor, error) {
	switch node.MediaType {
	case v1.MediaTypeImageManifest:
		content, err := content.FetchAll(ctx, fetcher, node)
		if err != nil {
			return nil, err
		}
		var manifest v1.Manifest
		if err := json.Unmarshal(content, &manifest); err != nil {
			return nil, err
		}
		var nodes []v1.Descriptor
		if manifest.Subject != nil {
			nodes = append(nodes, *manifest.Subject)
		}
		nodes = append(nodes, manifest.Config)
		return append(nodes, manifest.Layers...), nil
	case v1.MediaTypeImageIndex:
		content, err := content.FetchAll(ctx, fetcher, node)
		if err != nil {
			return nil, err
		}

		var index v1.Index
		if err := json.Unmarshal(content, &index); err != nil {
			return nil, err
		}
		var nodes []v1.Descriptor
		if index.Subject != nil {
			nodes = append(nodes, *index.Subject)
		}

		for _, manifest := range index.Manifests {
			if manifest.Platform != nil && len(popts.Platform) != 0 {
				pullPlatform, err := ParsePlatform(popts.Platform)
				if err != nil {
					return nil, err
				}
				if !reflect.DeepEqual(manifest.Platform, pullPlatform) {
					continue
				} else {
					nodes = append(nodes, manifest)
				}
			} else {
				nodes = append(nodes, manifest)
			}
		}
		return nodes, nil
	}
	return nil, nil
}

func ParsePlatform(platform string) (*v1.Platform, error) {
	// OS[/Arch[/Variant]][:OSVersion]
	// If Arch is not provided, will use GOARCH instead
	var platformStr string
	var p v1.Platform
	platformStr, p.OSVersion, _ = strings.Cut(platform, ":")
	parts := strings.Split(platformStr, "/")
	switch len(parts) {
	case 3:
		p.Variant = parts[2]
		fallthrough
	case 2:
		p.Architecture = parts[1]
	case 1:
		p.Architecture = runtime.GOARCH
	default:
		return nil, fmt.Errorf("failed to parse platform %q: expected format os[/arch[/variant]]", platform)
	}
	p.OS = parts[0]
	if p.OS == "" {
		return nil, fmt.Errorf("invalid platform: OS cannot be empty")
	}
	if p.Architecture == "" {
		return nil, fmt.Errorf("invalid platform: Architecture cannot be empty")
	}

	return &p, nil
}
