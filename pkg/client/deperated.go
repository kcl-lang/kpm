package client

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/dominikbraun/graph"
	"github.com/elliotchance/orderedmap/v2"
	"golang.org/x/mod/module"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/features"
	"kcl-lang.io/kpm/pkg/git"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/runner"
	"kcl-lang.io/kpm/pkg/utils"
)

// Deprecated: ValidateDependency is deprecated, use `Check` replaced.
func (c *KpmClient) ValidateDependency(dep *pkg.Dependency) error {
	if ok, err := features.Enabled(features.SupportModCheck); err == nil && ok {
		tmpKclPkg := pkg.KclPkg{
			ModFile: pkg.ModFile{
				Pkg: pkg.Package{
					Name:    dep.Name,
					Version: dep.Version,
				},
			},
			HomePath: dep.LocalFullPath,
			Dependencies: pkg.Dependencies{Deps: func() *orderedmap.OrderedMap[string, pkg.Dependency] {
				m := orderedmap.NewOrderedMap[string, pkg.Dependency]()
				m.Set(dep.Name, *dep)
				return m
			}()},
			NoSumCheck: c.GetNoSumCheck(),
		}

		if err := c.ModChecker.Check(tmpKclPkg); err != nil {
			return reporter.NewErrorEvent(reporter.InvalidKclPkg, err, fmt.Sprintf("%s package does not match the original kcl package", dep.FullName))
		}
	}

	return nil
}

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

// Deprecated: Use `ResolvePkgDepsMetadata` instead.
func (c *KpmClient) resolvePkgDeps(kclPkg *pkg.KclPkg, lockDeps *pkg.Dependencies, update bool) error {
	var searchPath string
	kclPkg.NoSumCheck = c.noSumCheck

	// If under the mode of '--no_sum_check', the checksum of the package will not be checked.
	// There is no kcl.mod.lock, and the dependencies in kcl.mod and kcl.mod.lock do not need to be aligned.
	if !c.noSumCheck {
		// If not under the mode of '--no_sum_check',
		// all the dependencies in kcl.mod.lock are the dependencies of the current package.
		//
		// align the dependencies between kcl.mod and kcl.mod.lock
		// clean the dependencies in kcl.mod.lock which not in kcl.mod
		// clean the dependencies in kcl.mod.lock and kcl.mod which have different version
		for _, name := range kclPkg.Dependencies.Deps.Keys() {
			dep, ok := kclPkg.Dependencies.Deps.Get(name)
			if !ok {
				break
			}
			modDep, ok := kclPkg.ModFile.Dependencies.Deps.Get(name)
			if !ok || !dep.Equals(modDep) {
				kclPkg.Dependencies.Deps.Delete(name)
			}
		}
		// add the dependencies in kcl.mod which not in kcl.mod.lock
		for _, name := range kclPkg.ModFile.Dependencies.Deps.Keys() {
			d, ok := kclPkg.ModFile.Dependencies.Deps.Get(name)
			if !ok {
				break
			}
			if _, ok := kclPkg.Dependencies.Deps.Get(name); !ok {
				kclPkg.Dependencies.Deps.Set(name, d)
			}
		}
	} else {
		// If under the mode of '--no_sum_check', the checksum of the package will not be checked.
		// All the dependencies in kcl.mod are the dependencies of the current package.
		kclPkg.Dependencies.Deps = kclPkg.ModFile.Dependencies.Deps
	}

	for _, name := range kclPkg.Dependencies.Deps.Keys() {
		d, ok := kclPkg.Dependencies.Deps.Get(name)
		if !ok {
			break
		}
		searchPath = c.getDepStorePath(kclPkg.HomePath, &d, kclPkg.IsVendorMode())
		depPath := searchPath
		// if the dependency is not exist
		if !utils.DirExists(searchPath) {
			if d.IsFromLocal() {
				// If the dependency is from the local path, and it does not exist locally, raise an error
				return reporter.NewErrorEvent(reporter.DependencyNotFound, fmt.Errorf("dependency '%s' not found in '%s'", d.Name, searchPath))
			} else {
				// redownload the dependency to the local path.
				if update {
					// re-vendor it.
					if kclPkg.IsVendorMode() {
						err := c.vendorDeps(kclPkg, kclPkg.LocalVendorPath())
						if err != nil {
							return err
						}
					} else {
						// re-download it.
						err := c.AddDepToPkg(kclPkg, &d)
						if err != nil {
							return err
						}

						depPath = c.getDepStorePath(kclPkg.HomePath, &d, kclPkg.IsVendorMode())
					}
				} else {
					continue
				}
			}
		}

		if d.GetPackage() != "" {
			depPath, _ = utils.FindPackage(depPath, d.GetPackage())
		}

		// If the dependency exists locally, load the dependency package.
		depPkg, err := c.LoadPkgFromPath(depPath)
		if err != nil {
			return reporter.NewErrorEvent(
				reporter.DependencyNotFound,
				fmt.Errorf("dependency '%s' not found in '%s'", d.Name, searchPath),
				// todo: add command to clean the package cache
			)
		}
		d.FromKclPkg(depPkg)
		err = c.resolvePkgDeps(depPkg, lockDeps, update)
		if err != nil {
			return err
		}
		if d.Source.Git != nil && d.Source.Git.GetPackage() != "" {
			if d.Source.Git != nil && d.Source.Git.GetPackage() != "" {
				name := utils.ParseRepoNameFromGitUrl(d.Source.Git.Url)
				if len(d.Source.Git.Tag) != 0 {
					d.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, d.Source.Git.Tag)
				} else if len(d.Source.Git.Commit) != 0 {
					d.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, d.Source.Git.Commit)
				} else {
					d.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, d.Source.Git.Branch)
				}
			}
		}
		kclPkg.Dependencies.Deps.Set(name, d)
		lockDeps.Deps.Set(name, d)
	}

	// Generate file kcl.mod.lock.
	if kclPkg.ModFile.Dependencies.Deps.Len() > 0 && !kclPkg.NoSumCheck || !update {
		err := kclPkg.LockDepsVersion()
		if err != nil {
			return err
		}
	}

	return nil
}

// Deprecated: this function is unstable and will be removed soon.
func GetReleasesFromSource(sourceType, uri string) ([]string, error) {
	var releases []string
	var err error

	switch sourceType {
	case pkg.GIT:
		releases, err = git.GetAllGithubReleases(uri)
	case pkg.OCI:
		releases, err = oci.GetAllImageTags(uri)
	}
	if err != nil {
		return nil, err
	}

	return releases, nil
}

// UpdateDeps will update the dependencies.
// Deprecated: Use `Update` instead.
func (c *KpmClient) UpdateDeps(kclPkg *pkg.KclPkg) error {
	_, err := c.ResolveDepsMetadataInJsonStr(kclPkg, true)
	if err != nil {
		return err
	}

	if ok, err := features.Enabled(features.SupportMVS); err != nil && ok {
		_, err = c.Update(
			WithUpdatedKclPkg(kclPkg),
			WithOffline(false),
		)
		if err != nil {
			return err
		}
	} else {
		// update kcl.mod
		err = kclPkg.UpdateModAndLockFile()
		if err != nil {
			return err
		}
	}

	return nil
}

// Compile will call kcl compiler to compile the current kcl package and its dependent packages.
// Deprecated: Use `Run` instead.
func (c *KpmClient) Compile(kclPkg *pkg.KclPkg, kclvmCompiler *runner.Compiler) (*kcl.KCLResultList, error) {
	pkgMap, err := c.ResolveDepsIntoMap(kclPkg)
	if err != nil {
		return nil, err
	}

	// Fill the dependency path.
	for dName, dPath := range pkgMap {
		if !filepath.IsAbs(dPath) {
			dPath = filepath.Join(c.homePath, dPath)
		}
		kclvmCompiler.AddDepPath(dName, dPath)
	}

	return kclvmCompiler.Run()
}

// AddDepWithOpts will add a dependency to the current kcl package.
// Deperated: Use Add instead.
func (c *KpmClient) AddDepWithOpts(kclPkg *pkg.KclPkg, opt *opt.AddOptions) (*pkg.KclPkg, error) {
	c.noSumCheck = opt.NoSumCheck
	kclPkg.NoSumCheck = opt.NoSumCheck

	// 1. get the name and version of the repository/package from the input arguments.
	d, err := pkg.ParseOpt(&opt.RegistryOpts)
	if err != nil {
		return nil, err
	}

	// Backup the dependency used in kcl.mod
	if opt.RegistryOpts.Registry != nil {
		kclPkg.BackupDepUI(d.Name, &pkg.Dependency{
			Name:    d.Name,
			Version: d.Version,
			Source: downloader.Source{
				ModSpec: &downloader.ModSpec{
					Name:    d.Name,
					Version: d.Version,
				},
			},
		})
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("adding dependency '%s'", d.Name),
		c.logWriter,
	)

	// 2. download the dependency to the local path.
	err = c.AddDepToPkg(kclPkg, d)
	if err != nil {
		return nil, err
	}

	// 3. update the kcl.mod and kcl.mod.lock.
	if opt.NewPkgName != "" {
		// update the kcl.mod with NewPkgName
		tempDeps, ok := kclPkg.ModFile.Dependencies.Deps.Get(d.Name)
		if !ok {
			return nil, fmt.Errorf("dependency '%s' not found in 'kcl.mod'", d.Name)
		}
		tempDeps.Name = opt.NewPkgName
		kclPkg.ModFile.Dependencies.Deps.Set(d.Name, tempDeps)

		// update the kcl.mod.lock with NewPkgName
		tempDeps, ok = kclPkg.Dependencies.Deps.Get(d.Name)
		if !ok {
			return nil, fmt.Errorf("dependency '%s' not found in 'kcl.mod.lock'", d.Name)
		}
		tempDeps.Name = opt.NewPkgName
		tempDeps.FullName = opt.NewPkgName + "_" + tempDeps.Version
		kclPkg.Dependencies.Deps.Set(d.Name, tempDeps)

		// update the key of kclPkg.Dependencies.Deps from d.Name to opt.NewPkgName
		kclPkg.Dependencies.Deps.Set(opt.NewPkgName, kclPkg.Dependencies.Deps.GetOrDefault(d.Name, pkg.TestPkgDependency))
		kclPkg.Dependencies.Deps.Delete(d.Name)
	}

	if ok, err := features.Enabled(features.SupportMVS); err != nil && ok {
		// After adding the new dependency,
		// Iterate through all the dependencies and select the version by mvs
		_, err = c.Update(
			WithUpdatedKclPkg(kclPkg),
		)

		if err != nil {
			return nil, err
		}
	} else {
		err = kclPkg.UpdateModAndLockFile()
		if err != nil {
			return nil, err
		}
	}

	succeedMsgInfo := d.Name
	if len(d.Version) != 0 {
		succeedMsgInfo = fmt.Sprintf("%s:%s", d.Name, d.Version)
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("add dependency '%s' successfully", succeedMsgInfo),
		c.logWriter,
	)
	return kclPkg, nil
}

// AddDepToPkg will add a dependency to the kcl package.
// Deprecated: Use `Add` instead.
func (c *KpmClient) AddDepToPkg(kclPkg *pkg.KclPkg, d *pkg.Dependency) error {

	// If the dependency is from the local path, do nothing.
	if d.IsFromLocal() {
		kclPkg.ModFile.Dependencies.Deps.Set(d.Name, *d)
		kclPkg.Dependencies.Deps.Set(d.Name, *d)
		return nil
	}

	// Some field will be empty when the dependency is add from CLI.
	// For avoiding re-download the dependency, just complete part of the fields not all of them.
	if !kclPkg.ModFile.Dependencies.Deps.GetOrDefault(d.Name, pkg.TestPkgDependency).Equals(*d) {
		// the dep passed on the cli is different from the kcl.mod.
		kclPkg.ModFile.Dependencies.Deps.Set(d.Name, *d)
	}

	// download all the dependencies.
	_, _, err := c.InitGraphAndDownloadDeps(kclPkg)

	if err != nil {
		return err
	}

	return err
}

// FillDepInfo will fill registry information for a dependency.
// Deprecated: this function is not used anymore.
func (c *KpmClient) FillDepInfo(dep *pkg.Dependency, homepath string) error {
	// Homepath for a dependency is the homepath of the kcl package.
	if dep.Source.Local != nil {
		dep.LocalFullPath = dep.Source.Local.Path
		return nil
	}
	if dep.Source.Oci != nil {
		if len(dep.Source.Oci.Reg) == 0 {
			dep.Source.Oci.Reg = c.GetSettings().DefaultOciRegistry()
		}

		if len(dep.Source.Oci.Repo) == 0 {
			urlpath := utils.JoinPath(c.GetSettings().DefaultOciRepo(), dep.Name)
			dep.Source.Oci.Repo = urlpath
		}
	}
	if dep.Source.Git != nil && dep.Source.Git.GetPackage() != "" {
		name := utils.ParseRepoNameFromGitUrl(dep.Source.Git.Url)
		if len(dep.Source.Git.Tag) != 0 {
			dep.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, dep.Source.Git.Tag)
		} else if len(dep.Source.Git.Commit) != 0 {
			dep.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, dep.Source.Git.Commit)
		} else {
			dep.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, dep.Source.Git.Branch)
		}
	}
	return nil
}

// FillDependenciesInfo will fill registry information for all dependencies in a kcl.mod.
// Deprecated: this function is not used anymore.
func (c *KpmClient) FillDependenciesInfo(modFile *pkg.ModFile) error {
	for _, k := range modFile.Deps.Keys() {
		v, ok := modFile.Deps.Get(k)
		if !ok {
			break
		}
		err := c.FillDepInfo(&v, modFile.HomePath)
		if err != nil {
			return err
		}
		modFile.Deps.Set(k, v)
	}
	return nil
}

// Deprecated: this function is not used anymore.
func (c *KpmClient) downloadPkg(options ...downloader.Option) (*pkg.KclPkg, error) {
	opts := downloader.DownloadOptions{}
	for _, option := range options {
		option(&opts)
	}

	localPath := opts.LocalPath
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, err
	}
	tmpDir = filepath.Join(tmpDir, constants.GitScheme)
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	credCli, err := c.GetCredsClient()
	if err != nil {
		return nil, err
	}

	err = c.DepDownloader.Download(downloader.NewDownloadOptions(
		downloader.WithLocalPath(tmpDir),
		downloader.WithSource(opts.Source),
		downloader.WithLogWriter(c.GetLogWriter()),
		downloader.WithSettings(*c.GetSettings()),
		downloader.WithCredsClient(credCli),
		downloader.WithInsecureSkipTLSverify(opts.InsecureSkipTLSverify),
	))

	if err != nil {
		return nil, err
	}

	if utils.DirExists(localPath) {
		err := os.RemoveAll(localPath)
		if err != nil {
			return nil, err
		}
	}

	destDir := filepath.Dir(localPath)
	if !utils.DirExists(destDir) {
		err = os.MkdirAll(destDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	err = utils.MoveFile(tmpDir, localPath)
	if err != nil {
		return nil, err
	}

	localPath, err = filepath.Abs(localPath)
	if err != nil {
		return nil, err
	}

	pkg, err := c.LoadPkgFromPath(localPath)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

// Download will download the dependency to the local path.
// Deprecated: the function is not used anymore, use `downloader.Download` instead.
func (c *KpmClient) Download(dep *pkg.Dependency, homePath, localPath string) (*pkg.Dependency, error) {
	if dep.Source.Git != nil {
		err := c.DepDownloader.Download(downloader.NewDownloadOptions(
			downloader.WithLocalPath(localPath),
			downloader.WithSource(dep.Source),
			downloader.WithLogWriter(c.logWriter),
			downloader.WithSettings(c.settings),
		))
		if err != nil {
			return nil, err
		}

		dep.FullName = dep.GenDepFullName()

		if dep.GetPackage() != "" {
			localFullPath, err := utils.FindPackage(localPath, dep.GetPackage())
			if err != nil {
				return nil, err
			}
			dep.LocalFullPath = localFullPath
			dep.Name = dep.GetPackage()
		} else {
			dep.LocalFullPath = localPath
		}

		modFile, err := c.LoadModFile(dep.LocalFullPath)
		if err != nil {
			return nil, err
		}
		dep.Version = modFile.Pkg.Version
	}

	if dep.Source.Oci != nil {
		var ociSource *downloader.Oci
		if dep.Source.Oci != nil {
			ociSource = dep.Source.Oci
		}
		// Select the latest tag, if the tag, the user inputted, is empty.
		if ociSource.Tag == "" || ociSource.Tag == constants.LATEST {
			latestTag, err := c.AcquireTheLatestOciVersion(*ociSource)
			if err != nil {
				return nil, err
			}
			ociSource.Tag = latestTag
			if dep.Source.ModSpec != nil {
				dep.Source.ModSpec.Version = latestTag
			}

			// Complete some information that the local three dependencies depend on.
			// The invalid path such as '$HOME/.kcl/kpm/k8s_' is placed because the version field is missing.
			dep.Version = latestTag
			dep.FullName = dep.GenDepFullName()
			dep.LocalFullPath = filepath.Join(filepath.Dir(localPath), dep.FullName)
			localPath = dep.LocalFullPath

			if utils.DirExists(dep.LocalFullPath) {
				dpkg, err := c.LoadPkgFromPath(localPath)
				if err != nil {
					// If the package is invalid, delete it and re-download it.
					err := os.RemoveAll(dep.LocalFullPath)
					if err != nil {
						return nil, err
					}
				} else {
					dep.FromKclPkg(dpkg)
					return dep, nil
				}
			}
		}

		credCli, err := c.GetCredsClient()
		if err != nil {
			return nil, err
		}
		err = c.DepDownloader.Download(downloader.NewDownloadOptions(
			downloader.WithLocalPath(localPath),
			downloader.WithSource(dep.Source),
			downloader.WithLogWriter(c.logWriter),
			downloader.WithSettings(c.settings),
			downloader.WithCredsClient(credCli),
			downloader.WithInsecureSkipTLSverify(c.insecureSkipTLSverify),
		))
		if err != nil {
			return nil, err
		}

		// load the package from the local path.
		dpkg, err := c.LoadPkgFromPath(localPath)
		if err != nil {
			return nil, err
		}

		dep.FromKclPkg(dpkg)

		dep.Sum, err = utils.HashDir(localPath)
		if err != nil {
			return nil, err
		}

		if dep.LocalFullPath == "" {
			dep.LocalFullPath = localPath
		}

		if localPath != dep.LocalFullPath {
			err = os.Rename(localPath, dep.LocalFullPath)
			if err != nil {
				return nil, err
			}
		}

		// Creating symbolic links in a global cache is not an optimal solution.
		// This allows kclvm to locate the package by default.
		// This feature is unstable and will be removed soon.
		// err = createDepRef(dep.LocalFullPath, filepath.Join(filepath.Dir(localPath), dep.Name))
		// if err != nil {
		//     return nil, err
		// }
		if err := c.ValidateDependency(dep); err != nil {
			return nil, err
		}
	}

	if dep.Source.Local != nil {
		kpkg, err := pkg.FindFirstKclPkgFrom(c.getDepStorePath(homePath, dep, false))
		if err != nil {
			return nil, err
		}
		dep.FromKclPkg(kpkg)
	}

	return dep, nil
}

// DownloadFromGit will download the dependency from the git repository.
// Deprecated: use 'DownloadFromGit' instead.
func (c *KpmClient) DownloadFromGit(dep *downloader.Git, localPath string) (string, error) {
	var msg string
	if len(dep.Tag) != 0 {
		msg = fmt.Sprintf("with tag '%s'", dep.Tag)
	}

	if len(dep.Commit) != 0 {
		msg = fmt.Sprintf("with commit '%s'", dep.Commit)
	}

	if len(dep.Branch) != 0 {
		msg = fmt.Sprintf("with branch '%s'", dep.Branch)
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("cloning '%s' %s", dep.Url, msg),
		c.logWriter,
	)

	_, err := git.CloneWithOpts(
		git.WithCommit(dep.Commit),
		git.WithTag(dep.Tag),
		git.WithRepoURL(dep.Url),
		git.WithLocalPath(localPath),
		git.WithWriter(c.logWriter),
	)

	if err != nil {
		return localPath, reporter.NewErrorEvent(
			reporter.FailedCloneFromGit,
			err,
			fmt.Sprintf("failed to clone from '%s' into '%s'.", dep.Url, localPath),
		)
	}

	return localPath, err
}

// Deprecated: This function is deprecated and will be removed in a future release.
func (c *KpmClient) ParseKclModFile(kclPkg *pkg.KclPkg) (map[string]map[string]string, error) {
	// Get path to kcl.mod file
	modFilePath := kclPkg.ModFile.GetModFilePath()

	// Read the content of the kcl.mod file
	modFileBytes, err := os.ReadFile(modFilePath)
	if err != nil {
		return nil, err
	}

	// Normalize line endings for Windows systems
	modFileContent := strings.ReplaceAll(string(modFileBytes), "\r\n", "\n")

	// Parse the TOML content
	var modFileData map[string]interface{}
	if err := toml.Unmarshal([]byte(modFileContent), &modFileData); err != nil {
		return nil, err
	}

	// Extract dependency information
	dependencies := make(map[string]map[string]string)
	if deps, ok := modFileData["dependencies"].(map[string]interface{}); ok {
		for dep, details := range deps {
			dependency := make(map[string]string)
			switch d := details.(type) {
			case string:
				// For simple version strings
				dependency["version"] = d
			case map[string]interface{}:
				// For dependencies with attributes
				for key, value := range d {
					dependency[key] = fmt.Sprintf("%v", value)
				}
			default:
				return nil, fmt.Errorf("unsupported dependency format")
			}
			dependencies[dep] = dependency
		}
	}

	return dependencies, nil
}

// LoadPkgFromOci will download the kcl package from the oci repository and return an `KclPkg`.
// Deprecated: this function is deprecated and will be removed in a future release.
func (c *KpmClient) DownloadPkgFromOci(dep *downloader.Oci, localPath string) (*pkg.KclPkg, error) {
	repoPath := utils.JoinPath(dep.Reg, dep.Repo)
	cred, err := c.GetCredentials(dep.Reg)
	if err != nil {
		return nil, err
	}

	ociClient, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
	)

	if err != nil {
		return nil, err
	}

	ociClient.SetLogWriter(c.logWriter)
	// Select the latest tag, if the tag, the user inputted, is empty.
	var tagSelected string
	if len(dep.Tag) == 0 {
		tagSelected, err = ociClient.TheLatestTag()
		if err != nil {
			return nil, err
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
		return nil, err
	}

	pkg, err := pkg.FindFirstKclPkgFrom(localPath)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

// PullFromOci will pull a kcl package from oci registry and unpack it.
// Deprecated: use `Pull` instead.
func (c *KpmClient) PullFromOci(localPath, source, tag string) error {
	localPath, err := filepath.Abs(localPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err)
	}
	if len(source) == 0 {
		return reporter.NewErrorEvent(
			reporter.UnKnownPullWhat,
			errors.FailedPull,
			"oci url or package name must be specified",
		)
	}

	if len(tag) == 0 {
		reporter.ReportMsgTo(
			fmt.Sprintf("start to pull '%s'", source),
			c.logWriter,
		)
	} else {
		reporter.ReportMsgTo(
			fmt.Sprintf("start to pull '%s' with tag '%s'", source, tag),
			c.logWriter,
		)
	}

	ociOpts, err := c.ParseOciOptionFromString(source, tag)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err, fmt.Sprintf("failed to create temp dir '%s'.", tmpDir))
	}
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	storepath := ociOpts.SanitizePathWithSuffix(tmpDir)
	err = c.pullTarFromOci(storepath, ociOpts)
	if err != nil {
		return err
	}

	// Get the (*.tar) file path.
	tarPath := filepath.Join(storepath, constants.KCL_PKG_TAR)
	matches, err := filepath.Glob(tarPath)
	if err != nil || len(matches) != 1 {
		if err == nil {
			err = errors.InvalidPkg
		}

		return reporter.NewErrorEvent(
			reporter.InvalidKclPkg,
			err,
			fmt.Sprintf("failed to find the kcl package tar from '%s'.", tarPath),
		)
	}

	// Untar the tar file.
	storagePath := ociOpts.SanitizePathWithSuffix(localPath)
	err = utils.UnTarDir(matches[0], storagePath)
	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedUntarKclPkg,
			err,
			fmt.Sprintf("failed to untar the kcl package tar from '%s' into '%s'.", matches[0], storagePath),
		)
	}

	if err := c.ValidatePkgPullFromOci(ociOpts, storagePath); err != nil {
		return reporter.NewErrorEvent(
			reporter.InvalidKclPkg,
			err,
			fmt.Sprintf("failed to validate kclPkg at %s", storagePath),
		)
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("pulled '%s' in '%s' successfully", source, storagePath),
		c.logWriter,
	)
	return nil
}

// Deprecated: This function is deprecated and will be removed in a future release.
func (c *KpmClient) ValidatePkgPullFromOci(ociOpts *opt.OciOptions, storagePath string) error {
	kclPkg, err := c.LoadPkgFromPath(storagePath)
	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedGetPkg,
			err,
			fmt.Sprintf("failed to load kclPkg at %v", storagePath),
		)
	}

	dep := &pkg.Dependency{
		Name: kclPkg.GetPkgName(),
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  ociOpts.Reg,
				Repo: ociOpts.Repo,
				Tag:  ociOpts.Tag,
			},
		},
	}

	dep.FromKclPkg(kclPkg)
	dep.Sum, err = utils.HashDir(storagePath)
	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedHashPkg,
			err,
			fmt.Sprintf("failed to hash kclPkg - %s", dep.Name),
		)
	}
	if err := c.ValidateDependency(dep); err != nil {
		return err
	}
	return nil
}

// ParseOciOptionFromString will parse '<repo_name>:<repo_tag>' into an 'OciOptions' with an OCI registry.
// the default OCI registry is 'docker.io'.
// if the 'ociUrl' is only '<repo_name>', ParseOciOptionFromString will take 'latest' as the default tag.
// Deprecated: this function is deprecated and will be removed in a future release.
func (c *KpmClient) ParseOciOptionFromString(oci string, tag string) (*opt.OciOptions, error) {
	ociOpt, event := opt.ParseOciUrl(oci)
	if event != nil && (event.Type() == reporter.IsNotUrl || event.Type() == reporter.UrlSchemeNotOci) {
		ociOpt, err := c.ParseOciRef(oci)
		if err != nil {
			return nil, err
		}
		if len(tag) != 0 {
			reporter.ReportEventTo(
				reporter.NewEvent(
					reporter.InvalidFlag,
					"kpm get version from oci reference '<repo_name>:<repo_tag>'",
				),
				c.logWriter,
			)
			reporter.ReportEventTo(
				reporter.NewEvent(
					reporter.InvalidFlag,
					"arg '--tag' is invalid for oci reference",
				),
				c.logWriter,
			)
		}
		return ociOpt, nil
	}

	ociOpt.Tag = tag

	return ociOpt, nil
}

// InitGraphAndDownloadDeps initializes a dependency graph and call downloadDeps function.
// Deprecated: this function is not used anymore and will be removed in the future.
func (c *KpmClient) InitGraphAndDownloadDeps(kclPkg *pkg.KclPkg) (*pkg.Dependencies, graph.Graph[module.Version, module.Version], error) {

	moduleHash := func(m module.Version) module.Version {
		return m
	}
	depGraph := graph.New(moduleHash, graph.Directed(), graph.PreventCycles())

	// add the root vertex(package name) to the dependency graph.
	root := module.Version{Path: kclPkg.GetPkgName(), Version: kclPkg.GetPkgVersion()}
	err := depGraph.AddVertex(root)
	if err != nil {
		return nil, nil, err
	}

	changedDeps, err := c.DownloadDeps(&kclPkg.ModFile.Dependencies, &kclPkg.Dependencies, depGraph, kclPkg.HomePath, root)
	if err != nil {
		return nil, nil, err
	}

	return changedDeps, depGraph, nil
}

// downloadDeps will download all the dependencies of the current kcl package.
// Deprecated: this function is not used anymore, it will be removed in the future.
func (c *KpmClient) DownloadDeps(deps *pkg.Dependencies, lockDeps *pkg.Dependencies, depGraph graph.Graph[module.Version, module.Version], pkghome string, parent module.Version) (*pkg.Dependencies, error) {

	newDeps := pkg.Dependencies{
		Deps: orderedmap.NewOrderedMap[string, pkg.Dependency](),
	}

	// Traverse all dependencies in kcl.mod
	for _, k := range deps.Deps.Keys() {
		d, _ := deps.Deps.Get(k)
		if len(d.Name) == 0 {
			return nil, errors.InvalidDependency
		}

		existDep, err := c.dependencyExistsLocal(pkghome, &d, false)
		if existDep != nil && err == nil {
			newDeps.Deps.Set(d.Name, *existDep)
			continue
		}

		expectedSum := lockDeps.Deps.GetOrDefault(d.Name, pkg.TestPkgDependency).Sum
		// Clean the cache
		if len(c.homePath) == 0 || len(d.FullName) == 0 {
			return nil, errors.InternalBug
		}

		dir := c.getDepStorePath(c.homePath, &d, false)
		err = os.RemoveAll(dir)
		if err != nil {
			return nil, err
		}

		// download dependencies
		lockedDep, err := c.Download(&d, pkghome, dir)
		if err != nil {
			return nil, err
		}

		if lockedDep.Oci != nil && lockedDep.Equals(lockDeps.Deps.GetOrDefault(d.Name, pkg.TestPkgDependency)) {
			if !c.noSumCheck && expectedSum != "" &&
				lockedDep.Sum != "" &&
				lockedDep.Sum != expectedSum {
				return nil, reporter.NewErrorEvent(
					reporter.CheckSumMismatch,
					errors.CheckSumMismatchError,
					fmt.Sprintf("checksum for '%s' changed in lock file '%s' and '%s'", lockedDep.Name, expectedSum, lockedDep.Sum),
				)
			} else {
				lockedDep.Sum = lockDeps.Deps.GetOrDefault(d.Name, pkg.Dependency{}).Sum
			}
		}

		newDeps.Deps.Set(d.Name, *lockedDep)
		// After downloading the dependency in kcl.mod, update the dep into to the kcl.mod
		// Only the direct dependencies are updated to kcl.mod.
		deps.Deps.Set(d.Name, *lockedDep)
	}

	// necessary to make a copy as when we are updating kcl.mod in below for loop
	// then newDeps.Deps gets updated and range gets an extra value to iterate through
	// this messes up the dependency graph
	newDepsCopy := orderedmap.NewOrderedMap[string, pkg.Dependency]()
	for _, k := range newDeps.Deps.Keys() {
		v, ok := newDeps.Deps.Get(k)
		if !ok {
			break
		}
		newDepsCopy.Set(k, v)
	}

	// Recursively download the dependencies of the new dependencies.
	for _, k := range newDepsCopy.Keys() {
		d, ok := newDepsCopy.Get(k)
		if !ok {
			break
		}
		var err error
		var deppkg *pkg.KclPkg
		if len(d.LocalFullPath) != 0 {
			if d.GetPackage() != "" {
				d.LocalFullPath, _ = utils.FindPackage(d.LocalFullPath, d.GetPackage())
			}
		} else {
			// Load kcl.mod file of the new downloaded dependencies.
			if d.GetPackage() != "" {
				d.LocalFullPath, _ = utils.FindPackage(filepath.Join(c.homePath, d.FullName), d.GetPackage())
			}
		}
		deppkg, err = c.LoadPkgFromPath(d.LocalFullPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		source := module.Version{Path: d.Name, Version: d.Version}

		err = depGraph.AddVertex(source, graph.VertexAttribute(d.GetSourceType(), d.GetDownloadPath()))
		if err != nil && err != graph.ErrVertexAlreadyExists {
			return nil, err
		}

		if parent != (module.Version{}) {
			err = depGraph.AddEdge(parent, source)
			if err != nil {
				if err == graph.ErrEdgeCreatesCycle {
					return nil, reporter.NewErrorEvent(
						reporter.CircularDependencyExist,
						nil,
						fmt.Sprintf("adding %s as a dependency results in a cycle", source),
					)
				}
				return nil, err
			}
		}

		// Download the indirect dependencies.
		nested, err := c.DownloadDeps(&deppkg.ModFile.Dependencies, lockDeps, depGraph, deppkg.HomePath, source)
		if err != nil {
			return nil, err
		}

		for _, k := range nested.Deps.Keys() {
			d, ok := nested.Deps.Get(k)
			if !ok {
				break
			}
			if _, ok := newDeps.Deps.Get(d.Name); !ok {
				newDeps.Deps.Set(d.Name, d)
			}
		}
	}

	// After each dependency is downloaded, update all the new deps to kcl.mod.lock.
	// No matter whether the dependency is directly or indirectly.
	for _, k := range newDeps.Deps.Keys() {
		v, ok := newDeps.Deps.Get(k)
		if !ok {
			break
		}
		lockDeps.Deps.Set(k, v)
	}

	return &newDeps, nil
}

// ParseOciRef will parse '<repo_name>:<repo_tag>' into an 'OciOptions'.
// Deprecated: ParseOciRef is deprecated and will be removed in a future release.
func (c *KpmClient) ParseOciRef(ociRef string) (*opt.OciOptions, error) {
	oci_address := strings.Split(ociRef, constants.OCI_SEPARATOR)
	if len(oci_address) == 1 {
		return &opt.OciOptions{
			Reg:  c.GetSettings().DefaultOciRegistry(),
			Repo: utils.JoinPath(c.GetSettings().DefaultOciRepo(), oci_address[0]),
		}, nil
	} else if len(oci_address) == 2 {
		return &opt.OciOptions{
			Reg:  c.GetSettings().DefaultOciRegistry(),
			Repo: utils.JoinPath(c.GetSettings().DefaultOciRepo(), oci_address[0]),
			Tag:  oci_address[1],
		}, nil
	} else {
		return nil, reporter.NewEvent(reporter.IsNotRef)
	}
}

// pullTarFromOci will pull a kcl package tar file from oci registry.
// Deprecated: use 'Pull' instead.
func (c *KpmClient) pullTarFromOci(localPath string, ociOpts *opt.OciOptions) error {
	absPullPath, err := filepath.Abs(localPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err)
	}

	repoPath := utils.JoinPath(ociOpts.Reg, ociOpts.Repo)
	cred, err := c.GetCredentials(ociOpts.Reg)
	if err != nil {
		return err
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
		oci.WithInsecureSkipTLSverify(ociOpts.InsecureSkipTLSverify),
	)

	if err != nil {
		return err
	}

	ociCli.SetLogWriter(c.logWriter)

	var tagSelected string
	if len(ociOpts.Tag) == 0 {
		tagSelected, err = ociCli.TheLatestTag()
		if err != nil {
			return err
		}
		reporter.ReportMsgTo(
			fmt.Sprintf("the latest version '%s' will be pulled", tagSelected),
			c.logWriter,
		)
	} else {
		tagSelected = ociOpts.Tag
	}

	ociOpts.Tag = tagSelected

	full_repo := utils.JoinPath(ociOpts.Reg, ociOpts.Repo)
	reporter.ReportMsgTo(
		fmt.Sprintf("pulling '%s:%s' from '%s'", ociOpts.Repo, tagSelected, full_repo),
		c.logWriter,
	)

	err = ociCli.Pull(absPullPath, tagSelected)
	if err != nil {
		return err
	}

	return nil
}

// FetchOciManifestConfIntoJsonStr will fetch the oci manifest config of the kcl package from the oci registry and return it into json string.
// Deprecated: use `SumChecker.FetchOciManifestIntoJsonStr` instead.
func (c *KpmClient) FetchOciManifestIntoJsonStr(opts opt.OciFetchOptions) (string, error) {

	repoPath := utils.JoinPath(opts.Reg, opts.Repo)
	cred, err := c.GetCredentials(opts.Reg)
	if err != nil {
		return "", err
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
	)

	if err != nil {
		return "", err
	}

	manifestJson, err := ociCli.FetchManifestIntoJsonStr(opts)
	if err != nil {
		return "", err
	}
	return manifestJson, nil
}

// AcquireTheLatestOciVersion will acquire the latest version of the OCI reference.
// Deprecated: use the 'downloader.LatestVersion' instead.
func (c *KpmClient) AcquireTheLatestOciVersion(ociSource downloader.Oci) (string, error) {
	repoPath := utils.JoinPath(ociSource.Reg, ociSource.Repo)
	cred, err := c.GetCredentials(ociSource.Reg)
	if err != nil {
		return "", err
	}

	ociClient, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
		oci.WithInsecureSkipTLSverify(c.insecureSkipTLSverify),
	)

	if err != nil {
		return "", err
	}

	return ociClient.TheLatestTag()
}

// Load the kcl.mod.lock and acquire the checksum of the dependencies from OCI registry.
// Deprecated: use `pkg.LoadLockDeps` instead.
func (c *KpmClient) LoadLockDeps(pkgPath string) (*pkg.Dependencies, error) {
	deps, err := pkg.LoadLockDeps(pkgPath)
	if err != nil {
		return nil, err
	}

	return deps, nil
}

// Deprecated: Use `pkg.LoadAndFillModFileWithOpts` replace this function.
func (c *KpmClient) LoadModFile(path string) (*pkg.ModFile, error) {
	return pkg.LoadAndFillModFileWithOpts(
		pkg.WithPath(path),
		pkg.WithSettings(&c.settings),
	)
}

// Deprecated: use `pkg.LoadKclPkgWithOpts` instead.
func (c *KpmClient) LoadPkgFromPath(path string) (*pkg.KclPkg, error) {
	return pkg.LoadKclPkgWithOpts(
		pkg.WithPath(path),
		pkg.WithSettings(&c.settings),
	)
}
