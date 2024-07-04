package downloader

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"kcl-lang.io/kpm/pkg/git"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
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
}

type Option func(*DownloadOptions)

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

// Downloader is the interface for downloading a package.
type Downloader interface {
	Download(opts DownloadOptions) error
}

// DepDownloader is the downloader for the package.
// Only support the OCI and git source.
type DepDownloader struct {
	*OciDownloader
	*GitDownloader
}

// GitDownloader is the downloader for the git source.
type GitDownloader struct{}

// OciDownloader is the downloader for the OCI source.
type OciDownloader struct {
	Platform string
}

func NewOciDownloader(platform string) *DepDownloader {
	return &DepDownloader{
		OciDownloader: &OciDownloader{
			Platform: platform,
		},
	}
}

func (d *DepDownloader) Download(opts DownloadOptions) error {
	// Dispatch the download to the specific downloader by package source.
	if opts.Source.Oci != nil || opts.Source.Registry != nil {
		if opts.Source.Registry != nil {
			opts.Source.Oci = opts.Source.Registry.Oci
		}
		if d.OciDownloader == nil {
			d.OciDownloader = &OciDownloader{}
		}
		return d.OciDownloader.Download(opts)
	}

	if opts.Source.Git != nil {
		if d.GitDownloader == nil {
			d.GitDownloader = &GitDownloader{}
		}
		return d.GitDownloader.Download(opts)
	}
	return nil
}

// Platform option struct.
type Platform struct {
	PlatformSpec string
	Platform     *v1.Platform
}

func (d *OciDownloader) Download(opts DownloadOptions) error {
	// download the package from the OCI registry
	ociSource := opts.Source.Oci
	if ociSource == nil {
		return errors.New("oci source is nil")
	}

	localPath := opts.LocalPath

	ociCli, err := oci.NewOciClient(ociSource.Reg, ociSource.Repo, &opts.Settings)
	if err != nil {
		return err
	}

	ociCli.PullOciOptions.Platform = d.Platform

	reporter.ReportMsgTo(
		fmt.Sprintf(
			"downloading '%s:%s' from '%s/%s:%s'",
			ociSource.Repo, ociSource.Tag, ociSource.Reg, ociSource.Repo, ociSource.Tag,
		),
		opts.LogWriter,
	)

	err = ociCli.Pull(localPath, ociSource.Tag)
	if err != nil {
		return err
	}

	matches, _ := filepath.Glob(filepath.Join(localPath, "*.tar"))
	if matches == nil || len(matches) != 1 {
		// then try to glob tgz file
		matches, _ = filepath.Glob(filepath.Join(localPath, "*.tgz"))
		if matches == nil || len(matches) != 1 {
			return fmt.Errorf("failed to find the downloaded kcl package tar file in '%s'", localPath)
		}
	}

	tarPath := matches[0]
	if utils.IsTar(tarPath) {
		err = utils.UnTarDir(tarPath, localPath)
	} else {
		err = utils.ExtractTarball(tarPath, localPath)
	}
	if err != nil {
		return fmt.Errorf("failed to untar the kcl package tar from '%s' into '%s'", tarPath, localPath)
	}

	// After untar the downloaded kcl package tar file, remove the tar file.
	if utils.DirExists(tarPath) {
		rmErr := os.Remove(tarPath)
		if rmErr != nil {
			return fmt.Errorf("failed to remove the downloaded kcl package tar file '%s'", tarPath)
		}
	}

	return err
}

func (d *GitDownloader) Download(opts DownloadOptions) error {
	var msg string
	if len(opts.Source.Git.Tag) != 0 {
		msg = fmt.Sprintf("with tag '%s'", opts.Source.Git.Tag)
	}

	if len(opts.Source.Git.Commit) != 0 {
		msg = fmt.Sprintf("with commit '%s'", opts.Source.Git.Commit)
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("cloning '%s' %s", opts.Source.Git.Url, msg),
		opts.LogWriter,
	)
	// download the package from the git repo
	gitSource := opts.Source.Git
	if gitSource == nil {
		return errors.New("git source is nil")
	}

	_, err := git.CloneWithOpts(
		git.WithCommit(gitSource.Commit),
		git.WithBranch(gitSource.Branch),
		git.WithTag(gitSource.Tag),
		git.WithRepoURL(gitSource.Url),
		git.WithLocalPath(opts.LocalPath),
	)

	if err != nil {
		return err
	}

	return nil
}
