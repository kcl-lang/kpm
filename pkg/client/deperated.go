package client

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/git"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// CompileWithOpts will compile the kcl program with the compile options.
// Deprecated: Use `Run` instead.
func (c *KpmClient) CompileWithOpts(opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	pkgPath, err := filepath.Abs(opts.PkgPath())
	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, please contact us to fix it.")
	}

	c.noSumCheck = opts.NoSumCheck()
	c.logWriter = opts.LogWriter()

	runOpts := RunOptions{}
	runOpts.Option = opts.Option
	c.noSumCheck = opts.NoSumCheck()
	c.logWriter = opts.LogWriter()
	pathSource := downloader.Source{
		Local: &downloader.Local{
			Path: pkgPath,
		},
	}

	pathSourceUrl, err := pathSource.ToString()
	if err != nil {
		return nil, err
	}

	return c.Run(
		WithRunOptions(&runOpts),
		WithRunSourceUrls(append([]string{pathSourceUrl}, opts.Entries()...)),
		WithVendor(opts.IsVendor()),
	)
}

// RunWithOpts will compile the kcl package with the compile options.
// Deprecated: Use `Run` instead.
func (c *KpmClient) RunWithOpts(opts ...opt.Option) (*kcl.KCLResultList, error) {
	mergedOpts := opt.DefaultCompileOptions()
	for _, opt := range opts {
		opt(mergedOpts)
	}
	return c.CompileWithOpts(mergedOpts)
}

// CompilePkgWithOpts will compile the kcl package with the compile options.
// Deprecated: Use `Run` instead.
func (c *KpmClient) CompilePkgWithOpts(kclPkg *pkg.KclPkg, opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	runOpts := RunOptions{}
	runOpts.Option = opts.Option
	c.noSumCheck = opts.NoSumCheck()
	c.logWriter = opts.LogWriter()
	pathSource := downloader.Source{
		Local: &downloader.Local{
			Path: kclPkg.HomePath,
		},
	}

	pathSourceUrl, err := pathSource.ToString()
	if err != nil {
		return nil, err
	}

	return c.Run(
		WithRunOptions(&runOpts),
		WithRunSourceUrls(append([]string{pathSourceUrl}, opts.Entries()...)),
		WithVendor(opts.IsVendor()),
	)
}

// CompileTarPkg will compile the kcl package from the tar package.
// Deprecated: Use `Run` instead.
func (c *KpmClient) CompileTarPkg(tarPath string, opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	runOpts := RunOptions{}
	runOpts.Option = opts.Option
	c.noSumCheck = opts.NoSumCheck()
	c.logWriter = opts.LogWriter()
	pathSource := downloader.Source{
		Local: &downloader.Local{
			Path: tarPath,
		},
	}

	pathSourceUrl, err := pathSource.ToString()
	if err != nil {
		return nil, err
	}

	return c.Run(
		WithRunOptions(&runOpts),
		WithRunSourceUrls(append([]string{pathSourceUrl}, opts.Entries()...)),
		WithVendor(opts.IsVendor()),
	)
}

// CompileGitPkg will compile the kcl package from the git url.
// Deprecated: Use `Run` instead.
func (c *KpmClient) CompileGitPkg(gitOpts *git.CloneOptions, compileOpts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	runOpts := RunOptions{}
	runOpts.Option = compileOpts.Option
	c.noSumCheck = compileOpts.NoSumCheck()
	c.logWriter = compileOpts.LogWriter()
	gitSource := downloader.Source{
		Git: &downloader.Git{
			Url:    gitOpts.RepoURL,
			Commit: gitOpts.Commit,
			Branch: gitOpts.Branch,
			Tag:    gitOpts.Tag,
		},
	}

	gitSourceUrl, err := gitSource.ToString()
	if err != nil {
		return nil, err
	}

	url, err := url.Parse(gitSourceUrl)
	if err != nil {
		return nil, err
	}

	if url.Scheme != "ssh" && url.Scheme != "git" {
		url.Scheme = "git"
	}

	return c.Run(
		WithRunOptions(&runOpts),
		WithRunSourceUrls(append([]string{url.String()}, compileOpts.Entries()...)),
		WithVendor(compileOpts.IsVendor()),
	)
}

// CompileOciPkg will compile the kcl package from the OCI reference or url.
// Deprecated: Use `Run` instead.
func (c *KpmClient) CompileOciPkg(ociSource, version string, opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	ociOpts, err := c.ParseOciOptionFromString(ociSource, version)

	if err != nil {
		return nil, err
	}

	runOpts := RunOptions{}
	runOpts.Option = opts.Option
	c.noSumCheck = opts.NoSumCheck()
	c.logWriter = opts.LogWriter()
	source := downloader.Source{
		Oci: &downloader.Oci{
			Reg:  ociOpts.Reg,
			Repo: ociOpts.Repo,
			Tag:  ociOpts.Tag,
		},
	}

	ociSourceUrl, err := source.ToString()
	if err != nil {
		return nil, err
	}

	return c.Run(
		WithRunOptions(&runOpts),
		WithRunSourceUrls(append([]string{ociSourceUrl}, opts.Entries()...)),
		WithVendor(opts.IsVendor()),
	)
}

// DownloadFromOci will download the dependency from the oci repository.
// Deprecated: Use the DownloadPkgFromOci instead.
func (c *KpmClient) DownloadFromOci(dep *downloader.Oci, localPath string) (string, error) {
	ociClient, err := oci.NewOciClient(dep.Reg, dep.Repo, &c.settings)
	if err != nil {
		return "", err
	}
	ociClient.SetLogWriter(c.logWriter)
	// Select the latest tag, if the tag, the user inputted, is empty.
	var tagSelected string
	if len(dep.Tag) == 0 {
		tagSelected, err = ociClient.TheLatestTag()
		if err != nil {
			return "", err
		}

		reporter.ReportMsgTo(
			fmt.Sprintf("the latest version '%s' will be added", tagSelected),
			c.logWriter,
		)

		dep.Tag = tagSelected
		localPath = localPath + dep.Tag
	} else {
		tagSelected = dep.Tag
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("downloading '%s:%s' from '%s/%s:%s'", dep.Repo, tagSelected, dep.Reg, dep.Repo, tagSelected),
		c.logWriter,
	)

	// Pull the package with the tag.
	err = ociClient.Pull(localPath, tagSelected)
	if err != nil {
		return "", err
	}

	tarPath, err := utils.FindPkgArchive(localPath)
	if err != nil {
		return "", err
	}
	if utils.IsTar(tarPath) {
		err = utils.UnTarDir(tarPath, localPath)
	} else {
		err = utils.ExtractTarball(tarPath, localPath)
	}
	if err != nil {
		return "", reporter.NewErrorEvent(
			reporter.FailedUntarKclPkg,
			err,
			fmt.Sprintf("failed to untar the kcl package from '%s' into '%s'.", tarPath, localPath),
		)
	}

	// After untar the downloaded kcl package tar file, remove the tar file.
	if utils.DirExists(tarPath) {
		rmErr := os.Remove(tarPath)
		if rmErr != nil {
			return "", reporter.NewErrorEvent(
				reporter.FailedUntarKclPkg,
				err,
				fmt.Sprintf("failed to untar the kcl package tar from '%s' into '%s'.", tarPath, localPath),
			)
		}
	}

	return localPath, nil
}
