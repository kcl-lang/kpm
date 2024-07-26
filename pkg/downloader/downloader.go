package downloader

import (
	"io"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"kcl-lang.io/kpm/pkg/settings"
	remoteauth "oras.land/oras-go/v2/registry/remote/auth"
)

// DownloadOptions is the options for downloading a package.
type DownloadOptions struct {
	// LocalPath is the local path to download the package.
	LocalPath string
	// Source is the source of the package. including git, oci, local.
	Source Source
	// Settings is the default settings and authrization information.
	Settings settings.Settings
	// LogWriter is the writer to write the log.
	LogWriter io.Writer
	// credsClient is the client to get the credentials.
	CredsClient CredentialClient
	// Platform is the platform for OCI downloads
    Platform string
}

type Option func(*DownloadOptions)

func WithCredsClient(credsClient *CredClient) Option {
	return func(do *DownloadOptions) {
		do.CredsClient = credsClient
	}
}

func WithLogWriter(logWriter io.Writer) Option {
	return func(do *DownloadOptions) {
		do.LogWriter = logWriter
	}
}

func WithSettings(settings settings.Settings) Option {
	return func(do *DownloadOptions) {
		do.Settings = settings
	}
}

func WithLocalPath(localPath string) Option {
	return func(do *DownloadOptions) {
		do.LocalPath = localPath
	}
}

func WithSource(source Source) Option {
	return func(do *DownloadOptions) {
		do.Source = source
	}
}

func NewDownloadOptions(opts ...Option) *DownloadOptions {
	do := &DownloadOptions{}
	for _, opt := range opts {
		opt(do)
	}
	return do
}

func WithPlatform(platform string) Option {
    return func(do *DownloadOptions) {
        do.Platform = platform
    }
}

// Downloader is the interface for downloading a package.
type Downloader interface {
	Download(opts DownloadOptions) error
}

type CredentialClient interface {
    Credential(registry string) (*remoteauth.Credential, error)
}

// New DependencySystem interface
type DependencySystem interface {
    Get(opts DownloadOptions) error
}

// DepDownloader is the downloader for the package.
// Only support the OCI and git source.
type DepDownloader struct {
    dependencySystem DependencySystem
    Platform         string
}

// New constructor for DepDownloader
func NewDepDownloader(platform string, ds DependencySystem) *DepDownloader {
    return &DepDownloader{
        dependencySystem: ds,
        Platform:         platform,
    }
}

func (d *DepDownloader) Download(opts DownloadOptions) error {
	// If platform is not set in options, use the one from DepDownloader
    if opts.Platform == "" {
        opts.Platform = d.Platform
    }
    return d.dependencySystem.Get(opts)
}

// Platform option struct.
type Platform struct {
	PlatformSpec string
	Platform     *v1.Platform
}
