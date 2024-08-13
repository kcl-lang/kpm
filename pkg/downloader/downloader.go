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
	credsClient *CredClient
	HomePath  string
}

type Option func(*DownloadOptions)

func WithCredsClient(credsClient *CredClient) Option {
	return func(do *DownloadOptions) {
		do.credsClient = credsClient
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

func WithHomePath(homePath string) func(*DownloadOptions) {
    return func(o *DownloadOptions) {
        o.HomePath = homePath
    }
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

	repoPath := utils.JoinPath(ociSource.Reg, ociSource.Repo)

	var cred *remoteauth.Credential
	var err error
	if opts.credsClient != nil {
		cred, err = opts.credsClient.Credential(ociSource.Reg)
		if err != nil {
			return err
		}
	} else {
		cred = &remoteauth.Credential{}
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(&opts.Settings),
	)

	if err != nil {
		return err
	}

	ociCli.PullOciOptions.Platform = d.Platform

	if len(ociSource.Tag) == 0 {
		tagSelected, err := ociCli.TheLatestTag()
		if err != nil {
			return err
		}

		reporter.ReportMsgTo(
			fmt.Sprintf("the lastest version '%s' will be downloaded", tagSelected),
			opts.LogWriter,
		)

		ociSource.Tag = tagSelected
	}

	reporter.ReportMsgTo(
		fmt.Sprintf(
			"downloading '%s:%s' from '%s/%s:%s'",
			ociSource.Repo, ociSource.Tag, ociSource.Reg, ociSource.Repo, ociSource.Tag,
		),
		opts.LogWriter,
	)


    // Use new directory structure
    ociDir := filepath.Join(opts.HomePath, "oci")
    cacheDir := filepath.Join(ociDir, "cache")
    srcDir := filepath.Join(ociDir, "src")

    repoHash := utils.CalculateHash(ociSource.Reg + "/" + ociSource.Repo)
    cacheSubDir := filepath.Join(cacheDir, fmt.Sprintf("%s-%s", ociSource.Reg, repoHash))
    srcSubDir := filepath.Join(srcDir, fmt.Sprintf("%s-%s", ociSource.Reg, repoHash))

	if err := os.MkdirAll(cacheSubDir, 0755); err != nil {
        return err
    }
    if err := os.MkdirAll(srcSubDir, 0755); err != nil {
        return err
    }

	tarPath := filepath.Join(cacheSubDir, fmt.Sprintf("%s_%s.tar", ociSource.Repo, ociSource.Tag))
    err = ociCli.Pull(tarPath, ociSource.Tag)
    if err != nil {
        return err
    }

	extractPath := filepath.Join(srcSubDir, fmt.Sprintf("%s_%s", ociSource.Repo, ociSource.Tag))
	if utils.IsTar(tarPath) {
		err = utils.UnTarDir(tarPath, extractPath)
	} else {
		err = utils.ExtractTarball(tarPath, extractPath)
	}
	if err != nil {
		return fmt.Errorf("failed to untar the kcl package tar from '%s' into '%s'", tarPath, extractPath)
	}

	// After untar the downloaded kcl package tar file, remove the tar file.
	if utils.DirExists(tarPath) {
		rmErr := os.Remove(tarPath)
		if rmErr != nil {
			return fmt.Errorf("failed to remove the downloaded kcl package tar file '%s'", tarPath)
		}
	}

	return utils.MoveFile(extractPath, opts.LocalPath)
}

func (d *GitDownloader) Download(opts DownloadOptions) error {
	var msg string
	if len(opts.Source.Git.Tag) != 0 {
		msg = fmt.Sprintf("with tag '%s'", opts.Source.Git.Tag)
	}

	if len(opts.Source.Git.Commit) != 0 {
		msg = fmt.Sprintf("with commit '%s'", opts.Source.Git.Commit)
	}

	if len(opts.Source.Git.Branch) != 0 {
		msg = fmt.Sprintf("with branch '%s'", opts.Source.Git.Branch)
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("cloning '%s' %s", opts.Source.Git.Url, msg),
		opts.LogWriter,
	)

	gitDir := filepath.Join(opts.HomePath, "git")
    checkoutsDir := filepath.Join(gitDir, "checkouts")
    dbDir := filepath.Join(gitDir, "db")

	if err := os.MkdirAll(checkoutsDir, 0755); err != nil {
        return err
    }
    if err := os.MkdirAll(dbDir, 0755); err != nil {
        return err
    }

	// download the package from the git repo
	gitSource := opts.Source.Git
	if gitSource == nil {
		return errors.New("git source is nil")
	}

	repoHash := utils.CalculateHash(gitSource.Url)
    repoName := utils.ParseRepoNameFromGitUrl(gitSource.Url)
    bareRepoDir := filepath.Join(dbDir, fmt.Sprintf("%s-%s", repoName, repoHash))

	if !utils.DirExists(bareRepoDir) {
        _, err := git.CloneWithOpts(
            git.WithBare(true),
            git.WithRepoURL(gitSource.Url),
            git.WithLocalPath(bareRepoDir),
        )
        if err != nil {
            return err
        }
    }
    
	checkoutDir := filepath.Join(checkoutsDir, fmt.Sprintf("%s-%s", repoName, repoHash), gitSource.Commit)

	_, err := git.CloneWithOpts(
        git.WithRepoURL(bareRepoDir),
        git.WithCommit(gitSource.Commit),
        git.WithBranch(gitSource.Branch),
        git.WithTag(gitSource.Tag),
        git.WithLocalPath(checkoutDir),
    )

	if err != nil {
		return err
	}

	return utils.MoveFile(checkoutDir, opts.LocalPath)
}
