package downloader

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/otiai10/copy"
	"kcl-lang.io/kpm/pkg/constants"
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
	if opts.EnableCache {
		// TODO: After the new local storage structure is complete,
		// this section should be replaced with the new storage structure instead of the cache path according to the <Cache Path>/<Package Name>.
		//  https://github.com/kcl-lang/kpm/issues/384
		var pkgFullName string
		if opts.Source.Registry != nil && len(opts.Source.Registry.Version) != 0 {
			pkgFullName = fmt.Sprintf("%s_%s", filepath.Base(opts.Source.Registry.Oci.Repo), opts.Source.Registry.Version)
		}
		if opts.Source.Oci != nil && len(opts.Source.Oci.Tag) != 0 {
			pkgFullName = fmt.Sprintf("%s_%s", filepath.Base(opts.Source.Oci.Repo), opts.Source.Oci.Tag)
		}

		if opts.Source.Git != nil && len(opts.Source.Git.Tag) != 0 {
			gitUrl := strings.TrimSuffix(opts.Source.Git.Url, filepath.Ext(opts.Source.Git.Url))
			pkgFullName = fmt.Sprintf("%s_%s", filepath.Base(gitUrl), opts.Source.Git.Tag)
		}
		if opts.Source.Git != nil && len(opts.Source.Git.Branch) != 0 {
			gitUrl := strings.TrimSuffix(opts.Source.Git.Url, filepath.Ext(opts.Source.Git.Url))
			pkgFullName = fmt.Sprintf("%s_%s", filepath.Base(gitUrl), opts.Source.Git.Branch)
		}
		if opts.Source.Git != nil && len(opts.Source.Git.Commit) != 0 {
			gitUrl := strings.TrimSuffix(opts.Source.Git.Url, filepath.Ext(opts.Source.Git.Url))
			pkgFullName = fmt.Sprintf("%s_%s", filepath.Base(gitUrl), opts.Source.Git.Commit)
		}

		cacheFullPath := filepath.Join(opts.CachePath, pkgFullName)

		if utils.DirExists(cacheFullPath) && utils.DirExists(filepath.Join(cacheFullPath, constants.KCL_MOD)) {
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

	opts.LocalPath = tmpDir
	// Dispatch the download to the specific downloader by package source.
	if opts.Source.Oci != nil || opts.Source.Registry != nil {
		if opts.Source.Registry != nil {
			opts.Source.Oci = opts.Source.Registry.Oci
		}
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

	err = utils.MoveOrCopy(tmpDir, localPath)
	if err != nil {
		return err
	}

	if opts.EnableCache {
		// Enable the cache, update the dependency package to the cache path.
		err := copy.Copy(tmpDir, cacheFullPath)
		if err != nil {
			return err
		}
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

	if len(opts.Source.Git.Branch) != 0 {
		msg = fmt.Sprintf("with branch '%s'", opts.Source.Git.Branch)
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
