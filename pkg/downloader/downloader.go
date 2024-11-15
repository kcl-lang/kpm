package downloader

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	gogit "github.com/go-git/go-git/v5"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/otiai10/copy"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/features"
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
	// CachePath is the cache path to download the package.
	CachePath string
	// EnableCache is the flag to enable the cache.
	// If `EnableCache` is false, this will not result in increasing disk usage.
	EnableCache bool
	// Source is the source of the package. including git, oci, local.
	Source Source
	// Settings is the default settings and authrization information.
	Settings settings.Settings
	// LogWriter is the writer to write the log.
	LogWriter io.Writer
	// credsClient is the client to get the credentials.
	credsClient *CredClient
	// InsecureSkipTLSverify is the flag to skip the verification of the certificate.
	InsecureSkipTLSverify bool
}

type Option func(*DownloadOptions)

func WithInsecureSkipTLSverify(insecureSkipTLSverify bool) Option {
	return func(do *DownloadOptions) {
		do.InsecureSkipTLSverify = insecureSkipTLSverify
	}
}

func WithCachePath(cachePath string) Option {
	return func(do *DownloadOptions) {
		do.CachePath = cachePath
	}
}

func WithEnableCache(enableCache bool) Option {
	return func(do *DownloadOptions) {
		do.EnableCache = enableCache
	}
}

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

// Downloader is the interface for downloading a package.
type Downloader interface {
	Download(opts *DownloadOptions) error
	// Get the latest version of the remote source
	// For the git source, it will return the latest commit
	// For the OCI source, it will return the latest tag
	LatestVersion(opts *DownloadOptions) (string, error)
}

func (d *DepDownloader) LatestVersion(opts *DownloadOptions) (string, error) {
	if opts.Source.Oci != nil {
		if d.OciDownloader == nil {
			d.OciDownloader = &OciDownloader{}
		}
		return d.OciDownloader.LatestVersion(opts)
	}

	if opts.Source.Git != nil {
		if d.GitDownloader == nil {
			d.GitDownloader = &GitDownloader{}
		}
		return d.GitDownloader.LatestVersion(opts)
	}

	return "", errors.New("source is nil")
}

// DepDownloader is the downloader for the package.
// Only support the OCI and git source.
type DepDownloader struct {
	*OciDownloader
	*GitDownloader
}

// GitDownloader is the downloader for the git source.
type GitDownloader struct{}

func (d *GitDownloader) LatestVersion(opts *DownloadOptions) (string, error) {
	// TODO：supports fetch the latest commit from the git bare repo,
	// after totally transfer to the new storage.
	// refer to cargo: https://github.com/rust-lang/cargo/blob/3dedb85a25604bdbbb8d3bf4b03162961a4facd0/crates/cargo-util-schemas/src/core/source_kind.rs#L133
	var repo *gogit.Repository
	if ok, err := features.Enabled(features.SupportNewStorage); err == nil && !ok && opts.EnableCache {
		var err error
		tmp, err := os.MkdirTemp("", "")
		if err != nil {
			return "", err
		}
		tmp = filepath.Join(tmp, constants.GitScheme)

		defer func() {
			err = os.RemoveAll(tmp)
		}()

		repo, err = git.CloneWithOpts(
			git.WithCommit(opts.Source.Commit),
			git.WithBranch(opts.Source.Branch),
			git.WithTag(opts.Source.Git.Tag),
			git.WithRepoURL(opts.Source.Git.Url),
			git.WithLocalPath(tmp),
		)

		if err != nil {
			return "", err
		}
	} else {
		// Get the latest commit from the git repository cache
		cacheFullPath := opts.CachePath
		// If the cache bare git repository exists, fetch the latest commit from the cache.
		if git.IsGitBareRepo(cacheFullPath) {
			err := git.Fetch(cacheFullPath)
			if err != nil {
				return "", err
			}
			repo, err = gogit.PlainOpen(cacheFullPath)
			if err != nil {
				return "", err
			}
		} else {
			// If not, clone the bare repository from the remote git repository, update the cache.
			cloneOpts := []git.CloneOption{
				git.WithCommit(opts.Source.Git.Commit),
				git.WithBranch(opts.Source.Git.Branch),
				git.WithTag(opts.Source.Git.Tag),
			}

			repo, err = git.CloneWithOpts(
				append(
					cloneOpts,
					git.WithRepoURL(opts.Source.Git.Url),
					git.WithLocalPath(cacheFullPath),
					git.WithBare(true),
				)...,
			)
			if err != nil {
				return "", err
			}
		}
	}
	ref, err := repo.Head()
	if err != nil {
		return "", err
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", err
	}

	return commit.Hash.String()[:7], nil
}

// OciDownloader is the downloader for the OCI source.
type OciDownloader struct {
	Platform string
}

func (d *OciDownloader) LatestVersion(opts *DownloadOptions) (string, error) {
	// download the package from the OCI registry
	ociSource := opts.Source.Oci
	if ociSource == nil {
		return "", errors.New("oci source is nil")
	}

	repoPath := utils.JoinPath(ociSource.Reg, ociSource.Repo)

	var cred *remoteauth.Credential
	var err error
	if opts.credsClient != nil {
		cred, err = opts.credsClient.Credential(ociSource.Reg)
		if err != nil {
			return "", err
		}
	} else {
		cred = &remoteauth.Credential{}
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(&opts.Settings),
		oci.WithInsecureSkipTLSverify(opts.InsecureSkipTLSverify),
	)

	if err != nil {
		return "", err
	}

	ociCli.PullOciOptions.Platform = d.Platform

	return ociCli.TheLatestTag()
}

func NewOciDownloader(platform string) *DepDownloader {
	return &DepDownloader{
		OciDownloader: &OciDownloader{
			Platform: platform,
		},
	}
}

func (d *DepDownloader) Download(opts *DownloadOptions) error {

	// create a tmp dir to download the oci package.
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return fmt.Errorf("failed to create a temp dir: %w", err)
	}
	if opts.Source.Git != nil {
		tmpDir = filepath.Join(tmpDir, constants.GitScheme)
	}
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	localPath := opts.LocalPath
	cacheFullPath := opts.CachePath
	if ok, err := features.Enabled(features.SupportNewStorage); err == nil && !ok && opts.EnableCache {
		if utils.DirExists(cacheFullPath) &&
			// If the version in modspec is empty, meanings the latest version is needed.
			// The latest version should be requested first and the cache should be updated.
			((opts.Source.ModSpec != nil && opts.Source.ModSpec.Version != "") || opts.Source.ModSpec == nil) {
			// copy the cache to the local path
			if cacheFullPath != opts.LocalPath {
				err := copy.Copy(cacheFullPath, opts.LocalPath)
				if err != nil {
					return err
				}
			}
			return nil
		} else {
			err := os.MkdirAll(cacheFullPath, 0755)
			if err != nil {
				return err
			}
		}
	}

	// If the dependency package is already exist,
	// Skip the download process.
	if utils.DirExists(localPath) &&
		utils.DirExists(filepath.Join(localPath, constants.KCL_MOD)) {
		return nil
	} else {
		opts.LocalPath = tmpDir
		// Dispatch the download to the specific downloader by package source.
		if opts.Source.Oci != nil {
			if d.OciDownloader == nil {
				d.OciDownloader = &OciDownloader{}
			}
			err := d.OciDownloader.Download(opts)
			if err != nil {
				return err
			}
		}

		if opts.Source.Git != nil {
			if d.GitDownloader == nil {
				d.GitDownloader = &GitDownloader{}
			}
			err := d.GitDownloader.Download(opts)
			if err != nil {
				return err
			}
		}

		// rename the tmp dir to the local path.
		if utils.DirExists(localPath) {
			err := os.RemoveAll(localPath)
			if err != nil {
				return err
			}
		}

		// Move the downloaded package to the local path.
		// On unix, after the move, the tmp dir will be removed.
		err = utils.MoveOrCopy(tmpDir, localPath)
		if err != nil {
			return err
		}

		if ok, err := features.Enabled(features.SupportNewStorage); err == nil && !ok && opts.EnableCache {
			// Enable the cache, update the dependency package to the cache path.
			if cacheFullPath != localPath {
				err := copy.Copy(localPath, cacheFullPath)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Platform option struct.
type Platform struct {
	PlatformSpec string
	Platform     *v1.Platform
}

func (d *OciDownloader) Download(opts *DownloadOptions) error {
	// download the package from the OCI registry
	ociSource := opts.Source.Oci
	if ociSource == nil {
		return errors.New("oci source is nil")
	}

	localPath := opts.LocalPath

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
		oci.WithInsecureSkipTLSverify(opts.InsecureSkipTLSverify),
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

	if ok, err := features.Enabled(features.SupportNewStorage); err == nil && ok {
		if opts.EnableCache {
			cacheFullPath := opts.CachePath
			localFullPath := opts.LocalPath

			if utils.DirExists(localFullPath) &&
				utils.DirExists(filepath.Join(localFullPath, constants.KCL_MOD)) {
				return nil
			} else {
				cacheTarPath, err := utils.FindPkgArchive(cacheFullPath)
				if err != nil && errors.Is(err, utils.PkgArchiveNotFound) {
					reporter.ReportMsgTo(
						fmt.Sprintf(
							"downloading '%s:%s' from '%s/%s:%s'",
							ociSource.Repo, ociSource.Tag, ociSource.Reg, ociSource.Repo, ociSource.Tag,
						),
						opts.LogWriter,
					)

					err = ociCli.Pull(cacheFullPath, ociSource.Tag)
					if err != nil {
						return err
					}
					cacheTarPath, err = utils.FindPkgArchive(cacheFullPath)
					if err != nil {
						return err
					}
				} else if err != nil {
					return err
				}

				if utils.IsTar(cacheTarPath) {
					err = utils.UnTarDir(cacheTarPath, localFullPath)
				} else {
					err = utils.ExtractTarball(cacheTarPath, localFullPath)
				}
				if err != nil {
					return err
				}
			}
		} else {
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
			tarPath, err := utils.FindPkgArchive(localPath)
			if err != nil {
				return err
			}
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
		}
	} else {
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
		tarPath, err := utils.FindPkgArchive(localPath)
		if err != nil {
			return err
		}
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

	}

	return err
}

func (d *GitDownloader) Download(opts *DownloadOptions) error {
	gitSource := opts.Source.Git
	if gitSource == nil {
		return errors.New("git source is nil")
	}
	cloneOpts := []git.CloneOption{
		git.WithCommit(gitSource.Commit),
		git.WithBranch(gitSource.Branch),
		git.WithTag(gitSource.Tag),
	}

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

	if ok, err := features.Enabled(features.SupportNewStorage); err == nil && ok {
		if opts.EnableCache {
			cacheFullPath := opts.CachePath
			localFullPath := opts.LocalPath
			// Check if the package is already downloaded, if so, skip the download.
			if utils.DirExists(localFullPath) &&
				utils.DirExists(filepath.Join(localFullPath, constants.KCL_MOD)) {
				return nil
			} else {
				// Try to clone the bare repository from the cache path.
				_, err := git.CloneWithOpts(
					append(
						cloneOpts,
						git.WithRepoURL(cacheFullPath),
						git.WithLocalPath(localFullPath),
					)...,
				)
				// If failed to clone the bare repository from the cache path,
				// clone the bare repository from the remote git repository, update the cache.
				if err != nil {
					// If the bare repository cache exists, fetch the latest commit from the cache.
					if utils.DirExists(cacheFullPath) && git.IsGitBareRepo(cacheFullPath) {
						err := git.Fetch(cacheFullPath)
						if err != nil {
							return err
						}
					} else {
						reporter.ReportMsgTo(
							fmt.Sprintf("cloning '%s' %s", opts.Source.Git.Url, msg),
							opts.LogWriter,
						)
						// If not, clone the bare repository from the remote git repository, update the cache.
						if utils.DirExists(cacheFullPath) {
							err = os.Remove(cacheFullPath)
							if err != nil {
								return err
							}
						}
						_, err := git.CloneWithOpts(
							append(
								cloneOpts,
								git.WithRepoURL(gitSource.Url),
								git.WithLocalPath(cacheFullPath),
								git.WithBare(true),
							)...,
						)
						if err != nil {
							return err
						}
					}
					// After cloning the bare repository,
					// Clone the repository from the cache path to the local path.
					_, err = git.CloneWithOpts(
						append(
							cloneOpts,
							git.WithRepoURL(cacheFullPath),
							git.WithLocalPath(localFullPath),
						)...,
					)
					if err != nil {
						return err
					}
				}
			}
		} else {
			reporter.ReportMsgTo(
				fmt.Sprintf("cloning '%s' %s", opts.Source.Git.Url, msg),
				opts.LogWriter,
			)
			// If the cache is disabled, clone the repository from the remote git repository.
			_, err := git.CloneWithOpts(
				append(
					cloneOpts,
					git.WithRepoURL(gitSource.Url),
					git.WithLocalPath(opts.LocalPath),
				)...,
			)
			if err != nil {
				return err
			}
		}
	} else {
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
	}
	return nil
}
