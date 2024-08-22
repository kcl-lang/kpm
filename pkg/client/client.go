package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	goerr "errors"

	"github.com/BurntSushi/toml"
	"github.com/dominikbraun/graph"
	"github.com/elliotchance/orderedmap/v2"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/otiai10/copy"
	"golang.org/x/mod/module"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"oras.land/oras-go/pkg/auth"
	"oras.land/oras-go/v2"
	remoteauth "oras.land/oras-go/v2/registry/remote/auth"

	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/git"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/runner"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

// KpmClient is the client of kpm.
type KpmClient struct {
	// The writer of the log.
	logWriter io.Writer
	// The downloader of the dependencies.
	DepDownloader *downloader.DepDownloader
	// credential store
	credsClient *downloader.CredClient
	// The home path of kpm for global configuration file and kcl package storage path.
	homePath string
	// The settings of kpm loaded from the global configuration file.
	settings settings.Settings
	// The flag of whether to check the checksum of the package and update kcl.mod.lock.
	noSumCheck bool
}

// NewKpmClient will create a new kpm client with default settings.
func NewKpmClient() (*KpmClient, error) {
	settings := settings.GetSettings()

	if settings.ErrorEvent != (*reporter.KpmEvent)(nil) {
		return nil, settings.ErrorEvent
	}

	homePath, err := env.GetAbsPkgPath()
	if err != nil {
		return nil, err
	}

	return &KpmClient{
		logWriter:     os.Stdout,
		settings:      *settings,
		homePath:      homePath,
		DepDownloader: &downloader.DepDownloader{},
	}, nil
}

// SetNoSumCheck will set the 'noSumCheck' flag.
func (c *KpmClient) SetNoSumCheck(noSumCheck bool) {
	c.noSumCheck = noSumCheck
}

// GetCredsClient will return the credential client.
func (c *KpmClient) GetCredsClient() (*downloader.CredClient, error) {
	if c.credsClient == nil {
		credCli, err := downloader.LoadCredentialFile(c.settings.CredentialsFile)
		if err != nil {
			return nil, err
		}
		c.credsClient = credCli
	}
	return c.credsClient, nil
}

// GetCredentials will return the credentials of the host.
func (c *KpmClient) GetCredentials(hostName string) (*remoteauth.Credential, error) {
	credCli, err := c.GetCredsClient()
	if err != nil {
		return nil, err
	}

	creds, err := credCli.Credential(hostName)
	if err != nil {
		return nil, err
	}

	return creds, nil
}

// GetNoSumCheck will return the 'noSumCheck' flag.
func (c *KpmClient) GetNoSumCheck() bool {
	return c.noSumCheck
}

func (c *KpmClient) SetLogWriter(writer io.Writer) {
	c.logWriter = writer
}

func (c *KpmClient) GetLogWriter() io.Writer {
	return c.logWriter
}

// SetHomePath will set the home path of kpm.
func (c *KpmClient) SetHomePath(homePath string) {
	c.homePath = homePath
}

// AcquirePackageCacheLock will acquire the lock of the package cache.
func (c *KpmClient) AcquirePackageCacheLock() error {
	return c.settings.AcquirePackageCacheLock(c.logWriter)
}

// ReleasePackageCacheLock will release the lock of the package cache.
func (c *KpmClient) ReleasePackageCacheLock() error {
	return c.settings.ReleasePackageCacheLock()
}

// GetSettings will return the settings of kpm client.
func (c *KpmClient) GetSettings() *settings.Settings {
	return &c.settings
}

func (c *KpmClient) LoadPkgFromPath(pkgPath string) (*pkg.KclPkg, error) {
	modFile, err := c.LoadModFile(pkgPath)
	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.FailedLoadKclMod, err, fmt.Sprintf("could not load 'kcl.mod' in '%s'", pkgPath))
	}

	// Get dependencies from kcl.mod.lock.
	deps, err := c.LoadLockDeps(pkgPath)

	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.FailedLoadKclMod, err, fmt.Sprintf("could not load 'kcl.mod.lock' in '%s'", pkgPath))
	}

	// Align the dependencies between kcl.mod and kcl.mod.lock.
	for _, name := range modFile.Dependencies.Deps.Keys() {
		dep, ok := modFile.Dependencies.Deps.Get(name)
		if !ok {
			break
		}
		if dep.Local != nil {
			if ldep, ok := deps.Deps.Get(name); ok {
				var localFullPath string
				if filepath.IsAbs(dep.Local.Path) {
					localFullPath = dep.Local.Path
				} else {
					localFullPath, err = filepath.Abs(filepath.Join(pkgPath, dep.Local.Path))
					if err != nil {
						return nil, reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, please contact us to fix it.")
					}
				}
				ldep.LocalFullPath = localFullPath
				dep.LocalFullPath = localFullPath
				ldep.Source = dep.Source
				deps.Deps.Set(name, ldep)
				modFile.Dependencies.Deps.Set(name, dep)
			}
		}
	}

	return &pkg.KclPkg{
		ModFile:      *modFile,
		HomePath:     pkgPath,
		Dependencies: *deps,
	}, nil
}

func (c *KpmClient) LoadModFile(pkgPath string) (*pkg.ModFile, error) {
	modFile := new(pkg.ModFile)
	err := modFile.LoadModFile(filepath.Join(pkgPath, pkg.MOD_FILE))
	if err != nil {
		return nil, err
	}

	modFile.HomePath = pkgPath

	if modFile.Dependencies.Deps == nil {
		modFile.Dependencies.Deps = orderedmap.NewOrderedMap[string, pkg.Dependency]()
	}
	err = c.FillDependenciesInfo(modFile)
	if err != nil {
		return nil, err
	}

	return modFile, nil
}

// Load the kcl.mod.lock and acquire the checksum of the dependencies from OCI registry.
func (c *KpmClient) LoadLockDeps(pkgPath string) (*pkg.Dependencies, error) {
	deps, err := pkg.LoadLockDeps(pkgPath)
	if err != nil {
		return nil, err
	}

	return deps, nil
}

// AcquireDepSum will acquire the checksum of the dependency from the OCI registry.
func (c *KpmClient) AcquireDepSum(dep pkg.Dependency) (string, error) {
	// Only the dependencies from the OCI need can be checked.
	if dep.Source.Oci != nil {
		if len(dep.Source.Oci.Reg) == 0 {
			dep.Source.Oci.Reg = c.GetSettings().DefaultOciRegistry()
		}

		if len(dep.Source.Oci.Repo) == 0 {
			urlpath := utils.JoinPath(c.GetSettings().DefaultOciRepo(), dep.Name)
			dep.Source.Oci.Repo = urlpath
		}
		// Fetch the metadata of the OCI manifest.
		manifest := ocispec.Manifest{}
		jsonDesc, err := c.FetchOciManifestIntoJsonStr(opt.OciFetchOptions{
			FetchBytesOptions: oras.DefaultFetchBytesOptions,
			OciOptions: opt.OciOptions{
				Reg:  dep.Source.Oci.Reg,
				Repo: dep.Source.Oci.Repo,
				Tag:  dep.Source.Oci.Tag,
			},
		})

		if err != nil {
			return "", reporter.NewErrorEvent(reporter.FailedFetchOciManifest, err, fmt.Sprintf("failed to fetch the manifest of '%s'", dep.Name))
		}

		err = json.Unmarshal([]byte(jsonDesc), &manifest)
		if err != nil {
			return "", err
		}

		// Check the dependency checksum.
		if value, ok := manifest.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_SUM]; ok {
			return value, nil
		}
	}

	return "", nil
}

// ResolveDepsIntoMap will calculate the map of kcl package name and local storage path of the external packages.
func (c *KpmClient) ResolveDepsIntoMap(kclPkg *pkg.KclPkg) (map[string]string, error) {
	err := c.ResolvePkgDepsMetadata(kclPkg, true)
	if err != nil {
		return nil, err
	}

	depMetadatas, err := kclPkg.GetDepsMetadata()
	if err != nil {
		return nil, err
	}
	var pkgMap map[string]string = make(map[string]string)
	for _, d := range depMetadatas.Deps {
		pkgMap[d.GetAliasName()] = d.GetLocalFullPath(kclPkg.HomePath)
	}

	return pkgMap, nil
}

const PKG_NAME_PATTERN = "%s_%s"

// Get the local store path for the dependency.
// 1. in the KCL_PKG_PATH: default is $HOME/.kcl/kpm
// 2. in the vendor subdirectory of the current package.
// 3. the dependency is from the local path.
func (c *KpmClient) getDepStorePath(search_path string, d *pkg.Dependency, isVendor bool) string {
	storePkgName := d.GenPathSuffix()
	if d.IsFromLocal() {
		return d.GetLocalFullPath(search_path)
	} else {
		path := ""
		if isVendor {
			path = filepath.Join(search_path, "vendor", storePkgName)
		} else {
			path = filepath.Join(c.homePath, storePkgName)
		}
		return path
	}
}

// ResolveDepsMetadata will calculate the local storage path of the external package,
// and check whether the package exists locally.
// If the package does not exist, it will re-download to the local.
// Since redownloads are not triggered if local dependencies exists,
// indirect dependencies are also synchronized to the lock file by `lockDeps`.
func (c *KpmClient) ResolvePkgDepsMetadata(kclPkg *pkg.KclPkg, update bool) error {
	if kclPkg.IsVendorMode() {
		// In the vendor mode, the search path is the vendor subdirectory of the current package.
		err := c.VendorDeps(kclPkg)
		if err != nil {
			return err
		}
	} else {
		// In the non-vendor mode, the search path is the KCL_PKG_PATH.
		err := c.resolvePkgDeps(kclPkg, &kclPkg.Dependencies, update)
		if err != nil {
			return err
		}

	}
	return nil
}

func (c *KpmClient) resolvePkgDeps(kclPkg *pkg.KclPkg, lockDeps *pkg.Dependencies, update bool) error {
	var searchPath string
	kclPkg.NoSumCheck = c.noSumCheck

	// If under the mode of '--no_sum_check', the checksum of the package will not be checked.
	// There is no kcl.mod.lock, and the dependencies in kcl.mod and kcl.mod.lock do not need to be aligned.
	if !c.noSumCheck {
		// If not under the mode of '--no_sum_check',
		// all the dependencies in kcl.mod.lock are the dependencies of the current package.
		//
		// alian the dependencies between kcl.mod and kcl.mod.lock
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
func (c *KpmClient) UpdateDeps(kclPkg *pkg.KclPkg) error {
	_, err := c.ResolveDepsMetadataInJsonStr(kclPkg, true)
	if err != nil {
		return err
	}

	// update kcl.mod
	err = kclPkg.ModFile.StoreModFile()
	if err != nil {
		return err
	}

	// Generate file kcl.mod.lock.
	if !kclPkg.NoSumCheck {
		err := kclPkg.LockDepsVersion()
		if err != nil {
			return err
		}
	}
	return nil
}

// ResolveDepsMetadataInJsonStr will calculate the local storage path of the external package,
// and check whether the package exists locally. If the package does not exist, it will re-download to the local.
// Finally, the calculated metadata of the dependent packages is serialized into a json string and returned.
func (c *KpmClient) ResolveDepsMetadataInJsonStr(kclPkg *pkg.KclPkg, update bool) (string, error) {
	// 1. Calculate the dependency path, check whether the dependency exists
	// and re-download the dependency that does not exist.
	err := c.ResolvePkgDepsMetadata(kclPkg, update)
	if err != nil {
		return "", err
	}

	// 2. Serialize to JSON
	depMetadatas, err := kclPkg.GetDepsMetadata()
	if err != nil {
		return "", err
	}
	jsonData, err := json.Marshal(&depMetadatas)
	if err != nil {
		return "", reporter.NewErrorEvent(reporter.Bug, err, "internal bug: failed to marshal the dependencies into json")
	}

	return string(jsonData), nil
}

// Compile will call kcl compiler to compile the current kcl package and its dependent packages.
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

// CompileWithOpts will compile the kcl program with the compile options.
func (c *KpmClient) CompileWithOpts(opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	pkgPath, err := filepath.Abs(opts.PkgPath())
	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, please contact us to fix it.")
	}

	c.noSumCheck = opts.NoSumCheck()
	c.logWriter = opts.LogWriter()

	kclPkg, err := c.LoadPkgFromPath(pkgPath)
	if err != nil {
		return nil, err
	}

	kclPkg.SetVendorMode(opts.IsVendor())

	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return nil, err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != (*reporter.KpmEvent)(nil) {
		return nil, err
	}
	// add all the options from 'kcl.mod'
	opts.Merge(*kclPkg.GetKclOpts())
	if len(opts.Entries()) > 0 {
		// add entry from '--input'
		for _, entry := range opts.Entries() {
			if filepath.IsAbs(entry) {
				opts.Merge(kcl.WithKFilenames(entry))
			} else {
				opts.Merge(kcl.WithKFilenames(filepath.Join(opts.PkgPath(), entry)))
			}
		}
	} else if len(kclPkg.GetEntryKclFilesFromModFile()) == 0 {
		// No entries profile in kcl.mod and no file settings in the settings file
		if !opts.HasSettingsYaml() {
			// No settings file.
			opts.Merge(kcl.WithKFilenames(opts.PkgPath()))
		} else if opts.HasSettingsYaml() && len(opts.KFilenameList) == 0 {
			// Has settings file but no file config in the settings files.
			opts.Merge(kcl.WithKFilenames(opts.PkgPath()))
		}
	}
	opts.Merge(kcl.WithWorkDir(opts.PkgPath()))

	// Calculate the absolute path of entry file described by '--input'.
	compiler := runner.NewCompilerWithOpts(opts)

	// Call the kcl compiler.
	compileResult, err := c.Compile(kclPkg, compiler)

	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.CompileFailed, err, "failed to compile the kcl package")
	}

	return compileResult, nil
}

// RunWithOpts will compile the kcl package with the compile options.
func (c *KpmClient) RunWithOpts(opts ...opt.Option) (*kcl.KCLResultList, error) {
	mergedOpts := opt.DefaultCompileOptions()
	for _, opt := range opts {
		opt(mergedOpts)
	}
	return c.CompileWithOpts(mergedOpts)
}

// CompilePkgWithOpts will compile the kcl package with the compile options.
func (c *KpmClient) CompilePkgWithOpts(kclPkg *pkg.KclPkg, opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	opts.SetPkgPath(kclPkg.HomePath)
	if len(opts.Entries()) > 0 {
		// add entry from '--input'
		for _, entry := range opts.Entries() {
			if filepath.IsAbs(entry) {
				opts.Merge(kcl.WithKFilenames(entry))
			} else {
				opts.Merge(kcl.WithKFilenames(filepath.Join(opts.PkgPath(), entry)))
			}
		}
		// add entry from 'kcl.mod'
	} else if len(kclPkg.GetEntryKclFilesFromModFile()) > 0 {
		opts.Merge(*kclPkg.GetKclOpts())
	} else if !opts.HasSettingsYaml() {
		// no entry
		opts.Merge(kcl.WithKFilenames(opts.PkgPath()))
	}
	opts.Merge(kcl.WithWorkDir(opts.PkgPath()))
	// Calculate the absolute path of entry file described by '--input'.
	compiler := runner.NewCompilerWithOpts(opts)
	// Call the kcl compiler.
	compileResult, err := c.Compile(kclPkg, compiler)

	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.CompileFailed, err, "failed to compile the kcl package")
	}

	return compileResult, nil
}

// CompileTarPkg will compile the kcl package from the tar package.
func (c *KpmClient) CompileTarPkg(tarPath string, opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	absTarPath, err := utils.AbsTarPath(tarPath)
	if err != nil {
		return nil, err
	}
	// Extract the tar package to a directory with the same name.
	// e.g.
	// 'xxx/xxx/xxx/test.tar' will be extracted to the directory 'xxx/xxx/xxx/test'.
	destDir := strings.TrimSuffix(absTarPath, filepath.Ext(absTarPath))
	err = utils.UnTarDir(absTarPath, destDir)
	if err != nil {
		return nil, err
	}

	opts.SetPkgPath(destDir)
	// The directory after extracting the tar package is taken as the root directory of the package,
	// and kclvm is called to compile the kcl program under the 'destDir'.
	// e.g.
	// if the tar path is 'xxx/xxx/xxx/test.tar',
	// the 'xxx/xxx/xxx/test' will be taken as the root path of the kcl package to compile.
	return c.CompileWithOpts(opts)
}

// CompileGitPkg will compile the kcl package from the git url.
func (c *KpmClient) CompileGitPkg(gitOpts *git.CloneOptions, compileOpts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	// 1. Create the temporary directory to pull the tar.
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, please contact us to fix it.")
	}
	tmpDir = filepath.Join(tmpDir, constants.GitEntry)

	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	// 2. clone the git repo
	_, err = git.CloneWithOpts(
		git.WithCommit(gitOpts.Commit),
		git.WithBranch(gitOpts.Branch),
		git.WithTag(gitOpts.Tag),
		git.WithRepoURL(gitOpts.RepoURL),
		git.WithLocalPath(tmpDir),
	)
	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.FailedGetPkg, err, "failed to get the git repository")
	}

	compileOpts.SetPkgPath(tmpDir)

	return c.CompileWithOpts(compileOpts)
}

// CompileOciPkg will compile the kcl package from the OCI reference or url.
func (c *KpmClient) CompileOciPkg(ociSource, version string, opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	ociOpts, err := c.ParseOciOptionFromString(ociSource, version)

	if err != nil {
		return nil, err
	}

	// 1. Create the temporary directory to pull the tar.
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, please contact us to fix it.")
	}
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	localPath := ociOpts.SanitizePathWithSuffix(tmpDir)

	// 2. Pull the tar.
	err = c.pullTarFromOci(localPath, ociOpts)

	if err != nil {
		return nil, err
	}

	// 3.Get the (*.tar) file path.
	matches, err := filepath.Glob(filepath.Join(localPath, constants.KCL_PKG_TAR))
	if err != nil || len(matches) != 1 {
		if err != nil {
			return nil, reporter.NewErrorEvent(reporter.FailedGetPkg, err, "failed to pull kcl package")
		} else {
			return nil, errors.FailedPull
		}
	}

	return c.CompileTarPkg(matches[0], opts)
}

// createIfNotExist will create a file if it does not exist.
func (c *KpmClient) createIfNotExist(filepath string, storeFunc func() error) error {
	reporter.ReportMsgTo(fmt.Sprintf("creating new :%s", filepath), c.GetLogWriter())
	err := utils.CreateFileIfNotExist(
		filepath,
		storeFunc,
	)
	if err != nil {
		if errEvent, ok := err.(*reporter.KpmEvent); ok {
			if errEvent.Type() != reporter.FileExists {
				return err
			} else {
				reporter.ReportMsgTo(fmt.Sprintf("'%s' already exists", filepath), c.GetLogWriter())
			}
		} else {
			return err
		}
	}

	return nil
}

// InitEmptyPkg will initialize an empty kcl package.
func (c *KpmClient) InitEmptyPkg(kclPkg *pkg.KclPkg) error {
	err := c.createIfNotExist(kclPkg.ModFile.GetModFilePath(), kclPkg.ModFile.StoreModFile)
	if err != nil {
		return err
	}

	err = c.createIfNotExist(kclPkg.ModFile.GetModLockFilePath(), kclPkg.LockDepsVersion)
	if err != nil {
		return err
	}

	err = c.createIfNotExist(filepath.Join(kclPkg.ModFile.HomePath, constants.DEFAULT_KCL_FILE_NAME), kclPkg.CreateDefauleMain)
	if err != nil {
		return err
	}

	return nil
}

// AddDepWithOpts will add a dependency to the current kcl package.
func (c *KpmClient) AddDepWithOpts(kclPkg *pkg.KclPkg, opt *opt.AddOptions) (*pkg.KclPkg, error) {
	c.noSumCheck = opt.NoSumCheck
	kclPkg.NoSumCheck = opt.NoSumCheck

	// 1. get the name and version of the repository/package from the input arguments.
	d, err := pkg.ParseOpt(&opt.RegistryOpts)
	if err != nil {
		return nil, err
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

	err = kclPkg.UpdateModAndLockFile()
	if err != nil {
		return nil, err
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

// PackagePkg will package the current kcl package into a "*.tar" file in under the package path.
func (c *KpmClient) PackagePkg(kclPkg *pkg.KclPkg, vendorMode bool) (string, error) {
	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return "", err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != (*reporter.KpmEvent)(nil) {
		return "", err
	}

	err = c.Package(kclPkg, kclPkg.DefaultTarPath(), vendorMode)

	if err != nil {
		reporter.ExitWithReport("failed to package pkg " + kclPkg.GetPkgName() + ".")
		return "", err
	}
	return kclPkg.DefaultTarPath(), nil
}

// Package will package the current kcl package into a "*.tar" file into 'tarPath'.
func (c *KpmClient) Package(kclPkg *pkg.KclPkg, tarPath string, vendorMode bool) error {
	// Vendor all the dependencies into the current kcl package.
	if vendorMode {
		err := c.VendorDeps(kclPkg)
		if err != nil {
			return reporter.NewErrorEvent(reporter.FailedVendor, err, "failed to vendor dependencies")
		}
	}

	// Tar the current kcl package into a "*.tar" file.
	err := utils.TarDir(kclPkg.HomePath, tarPath, kclPkg.GetPkgInclude(), kclPkg.GetPkgExclude())
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedPackage, err, "failed to package the kcl module")
	}
	return nil
}

func (c *KpmClient) vendorDeps(kclPkg *pkg.KclPkg, vendorPath string) error {
	lockDeps := make([]pkg.Dependency, 0, kclPkg.Dependencies.Deps.Len())
	for _, k := range kclPkg.Dependencies.Deps.Keys() {
		d, _ := kclPkg.Dependencies.Deps.Get(k)
		lockDeps = append(lockDeps, d)
	}

	// Traverse all dependencies in kcl.mod.lock.
	for i := 0; i < len(lockDeps); i++ {
		d := lockDeps[i]
		if len(d.Name) == 0 {
			return errors.InvalidDependency
		}
		// If the dependency is from the local path, do not vendor it, vendor its dependencies.
		if d.IsFromLocal() {
			dpkg, err := c.LoadPkgFromPath(d.GetLocalFullPath(kclPkg.HomePath))
			if err != nil {
				return err
			}
			err = c.vendorDeps(dpkg, vendorPath)
			if err != nil {
				return err
			}
			continue
		} else {
			vendorFullPath := filepath.Join(vendorPath, d.GenPathSuffix())

			// If the package already exists in the 'vendor', do nothing.
			if utils.DirExists(vendorFullPath) {
				d.LocalFullPath = vendorFullPath
				lockDeps[i] = d
				continue
			} else {
				// If not in the 'vendor', check the global cache.
				cacheFullPath := c.getDepStorePath(c.homePath, &d, false)
				if utils.DirExists(cacheFullPath) {
					// If there is, copy it into the 'vendor' directory.
					err := copy.Copy(cacheFullPath, vendorFullPath)
					if err != nil {
						return err
					}
				} else {
					// re-download if not.
					err := c.AddDepToPkg(kclPkg, &d)
					if err != nil {
						return err
					}
					// re-vendor again with new kcl.mod and kcl.mod.lock
					err = c.vendorDeps(kclPkg, vendorPath)
					if err != nil {
						return err
					}
					return nil
				}
			}

			if d.GetPackage() != "" {
				tempVendorFullPath, err := utils.FindPackage(vendorFullPath, d.GetPackage())
				if err != nil {
					return err
				}
				vendorFullPath = tempVendorFullPath
			}

			dpkg, err := c.LoadPkgFromPath(vendorFullPath)
			if err != nil {
				return err
			}

			// Vendor the dependencies of the current dependency.
			err = c.vendorDeps(dpkg, vendorPath)
			if err != nil {
				return err
			}
			d.LocalFullPath = vendorFullPath
			lockDeps[i] = d
		}
	}

	// Update the dependencies in kcl.mod.lock.
	for _, d := range lockDeps {
		kclPkg.Dependencies.Deps.Set(d.Name, d)
	}

	return nil
}

// VendorDeps will vendor all the dependencies of the current kcl package.
func (c *KpmClient) VendorDeps(kclPkg *pkg.KclPkg) error {
	// Mkdir the dir "vendor".
	vendorPath := kclPkg.LocalVendorPath()
	err := os.MkdirAll(vendorPath, 0755)
	if err != nil {
		return err
	}

	return c.vendorDeps(kclPkg, vendorPath)
}

// FillDepInfo will fill registry information for a dependency.
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
	if dep.Source.Registry != nil {
		if len(dep.Source.Registry.Reg) == 0 {
			dep.Source.Registry.Reg = c.GetSettings().DefaultOciRegistry()
		}

		if len(dep.Source.Registry.Repo) == 0 {
			urlpath := utils.JoinPath(c.GetSettings().DefaultOciRepo(), dep.Name)
			dep.Source.Registry.Repo = urlpath
		}

		dep.Version = dep.Source.Registry.Version
	}
	return nil
}

// FillDependenciesInfo will fill registry information for all dependencies in a kcl.mod.
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

// AcquireTheLatestOciVersion will acquire the latest version of the OCI reference.
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
	)

	if err != nil {
		return "", err
	}

	return ociClient.TheLatestTag()
}

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
	err = c.DepDownloader.Download(*downloader.NewDownloadOptions(
		downloader.WithLocalPath(tmpDir),
		downloader.WithSource(opts.Source),
		downloader.WithLogWriter(c.GetLogWriter()),
		downloader.WithSettings(*c.GetSettings()),
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
func (c *KpmClient) Download(dep *pkg.Dependency, homePath, localPath string) (*pkg.Dependency, error) {
	if dep.Source.Git != nil {
		err := c.DepDownloader.Download(*downloader.NewDownloadOptions(
			downloader.WithLocalPath(localPath),
			downloader.WithSource(dep.Source),
			downloader.WithLogWriter(c.logWriter),
			downloader.WithSettings(c.settings),
		))
		if err != nil {
			return nil, err
		}

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
		// Creating symbolic links in a global cache is not an optimal solution.
		// This allows kclvm to locate the package by default.
		// This feature is unstable and will be removed soon.
		// err = createDepRef(dep.LocalFullPath, filepath.Join(filepath.Dir(localPath), dep.Name))
		// if err != nil {
		//     return nil, err
		// }
		dep.FullName = dep.GenDepFullName()

		modFile, err := c.LoadModFile(dep.LocalFullPath)
		if err != nil {
			return nil, err
		}
		dep.Version = modFile.Pkg.Version
	}

	if dep.Source.Oci != nil || dep.Source.Registry != nil {
		var ociSource *downloader.Oci
		if dep.Source.Oci != nil {
			ociSource = dep.Source.Oci
		} else if dep.Source.Registry != nil {
			ociSource = dep.Source.Registry.Oci
		}
		// Select the latest tag, if the tag, the user inputed, is empty.
		if ociSource.Tag == "" || ociSource.Tag == constants.LATEST {
			latestTag, err := c.AcquireTheLatestOciVersion(*ociSource)
			if err != nil {
				return nil, err
			}
			ociSource.Tag = latestTag

			if dep.Source.Registry != nil {
				dep.Source.Registry.Tag = latestTag
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

		// create a tmp dir to download the oci package.
		tmpDir, err := os.MkdirTemp("", "")
		if err != nil {
			return nil, reporter.NewErrorEvent(reporter.Bug, err, fmt.Sprintf("failed to create temp dir '%s'.", tmpDir))
		}
		// clean the temp dir.
		defer os.RemoveAll(tmpDir)

		credCli, err := c.GetCredsClient()
		if err != nil {
			return nil, err
		}
		err = c.DepDownloader.Download(*downloader.NewDownloadOptions(
			downloader.WithLocalPath(tmpDir),
			downloader.WithSource(dep.Source),
			downloader.WithLogWriter(c.logWriter),
			downloader.WithSettings(c.settings),
			downloader.WithCredsClient(credCli),
		))
		if err != nil {
			return nil, err
		}

		// check the package in tmp dir is a valid kcl package.
		_, err = pkg.FindFirstKclPkgFrom(tmpDir)
		if err != nil {
			return nil, err
		}

		// rename the tmp dir to the local path.
		if utils.DirExists(localPath) {
			err := os.RemoveAll(localPath)
			if err != nil {
				return nil, err
			}
		}

		if runtime.GOOS != "windows" {
			err = os.Rename(tmpDir, localPath)
			if err != nil {
				// check the error is caused by moving the file across file systems.
				if goerr.Is(err, syscall.EXDEV) {
					// If it is, use copy as a fallback.
					err = copy.Copy(tmpDir, localPath)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			}
		} else {
			err = copy.Copy(tmpDir, localPath)
			if err != nil {
				return nil, err
			}
		}

		// load the package from the local path.
		dpkg, err := c.LoadPkgFromPath(localPath)
		if err != nil {
			return nil, err
		}

		dep.FromKclPkg(dpkg)
		dep.Sum, err = c.AcquireDepSum(*dep)
		if err != nil {
			return nil, err
		}
		if dep.Sum == "" {
			dep.Sum, err = utils.HashDir(localPath)
			if err != nil {
				return nil, err
			}
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
	// Select the latest tag, if the tag, the user inputed, is empty.
	var tagSelected string
	if len(dep.Tag) == 0 {
		tagSelected, err = ociClient.TheLatestTag()
		if err != nil {
			return nil, err
		}

		reporter.ReportMsgTo(
			fmt.Sprintf("the lastest version '%s' will be added", tagSelected),
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

// DownloadFromOci will download the dependency from the oci repository.
// Deprecated: Use the DownloadPkgFromOci instead.
func (c *KpmClient) DownloadFromOci(dep *downloader.Oci, localPath string) (string, error) {
	ociClient, err := oci.NewOciClient(dep.Reg, dep.Repo, &c.settings)
	if err != nil {
		return "", err
	}
	ociClient.SetLogWriter(c.logWriter)
	// Select the latest tag, if the tag, the user inputed, is empty.
	var tagSelected string
	if len(dep.Tag) == 0 {
		tagSelected, err = ociClient.TheLatestTag()
		if err != nil {
			return "", err
		}

		reporter.ReportMsgTo(
			fmt.Sprintf("the lastest version '%s' will be added", tagSelected),
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

	matches, _ := filepath.Glob(filepath.Join(localPath, "*.tar"))
	if matches == nil || len(matches) != 1 {
		// then try to glob tgz file
		matches, _ = filepath.Glob(filepath.Join(localPath, "*.tgz"))
		if matches == nil || len(matches) != 1 {
			return "", reporter.NewErrorEvent(
				reporter.InvalidKclPkg,
				err,
				fmt.Sprintf("failed to find the kcl package from '%s'.", localPath),
			)
		}
	}

	tarPath := matches[0]
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

// PullFromOci will pull a kcl package from oci registry and unpack it.
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

	reporter.ReportMsgTo(
		fmt.Sprintf("pulled '%s' in '%s' successfully", source, storagePath),
		c.logWriter,
	)
	return nil
}

// PushToOci will push a kcl package to oci registry.
func (c *KpmClient) PushToOci(localPath string, ociOpts *opt.OciOptions) error {
	repoPath := utils.JoinPath(ociOpts.Reg, ociOpts.Repo)
	cred, err := c.GetCredentials(ociOpts.Reg)
	if err != nil {
		return err
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
	)

	if err != nil {
		return err
	}

	ociCli.SetLogWriter(c.logWriter)

	exist, err := ociCli.ContainsTag(ociOpts.Tag)
	if err != (*reporter.KpmEvent)(nil) {
		return err
	}

	if exist {
		return reporter.NewErrorEvent(
			reporter.PkgTagExists,
			fmt.Errorf("package version '%s' already exists", ociOpts.Tag),
		)
	}

	return ociCli.PushWithOciManifest(localPath, ociOpts.Tag, &opt.OciManifestOptions{
		Annotations: ociOpts.Annotations,
	})
}

// LoginOci will login to the oci registry.
func (c *KpmClient) LoginOci(hostname, username, password string) error {

	credCli, err := c.GetCredsClient()
	if err != nil {
		return err
	}

	err = credCli.GetAuthClient().LoginWithOpts(
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

// LogoutOci will logout from the oci registry.
func (c *KpmClient) LogoutOci(hostname string) error {

	credCli, err := c.GetCredsClient()
	if err != nil {
		return err
	}

	err = credCli.GetAuthClient().Logout(context.Background(), hostname)

	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedLogout, err, fmt.Sprintf("failed to logout '%s'", hostname))
	}

	return nil
}

// ParseOciRef will parser '<repo_name>:<repo_tag>' into an 'OciOptions'.
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

// ParseOciOptionFromString will parser '<repo_name>:<repo_tag>' into an 'OciOptions' with an OCI registry.
// the default OCI registry is 'docker.io'.
// if the 'ociUrl' is only '<repo_name>', ParseOciOptionFromString will take 'latest' as the default tag.
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

// dependencyExists will check whether the dependency exists in the local filesystem.
func (c *KpmClient) dependencyExistsLocal(searchPath string, dep *pkg.Dependency) (*pkg.Dependency, error) {
	// If the flag '--no_sum_check' is set, skip the checksum check.
	deppath := c.getDepStorePath(searchPath, dep, false)
	if utils.DirExists(deppath) {
		depPkg, err := c.LoadPkgFromPath(deppath)
		if err != nil {
			return nil, err
		}
		dep.FromKclPkg(depPkg)
		return dep, nil
	}
	return nil, nil
}

// downloadDeps will download all the dependencies of the current kcl package.
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

		existDep, err := c.dependencyExistsLocal(pkghome, &d)
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
			deppkg, err = c.LoadPkgFromPath(d.LocalFullPath)
		} else {
			// Load kcl.mod file of the new downloaded dependencies.
			deppkg, err = c.LoadPkgFromPath(filepath.Join(c.homePath, d.FullName))

		}
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

// pullTarFromOci will pull a kcl package tar file from oci registry.
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
			fmt.Sprintf("the lastest version '%s' will be pulled", tagSelected),
			c.logWriter,
		)
	} else {
		tagSelected = ociOpts.Tag
	}

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

// createDepRef will create a dependency reference for the dependency saved on the local filesystem.
// On the unix-like system, it will create a symbolic link.
// On the windows system, it will create a junction.
// func createDepRef(depName, refName string) error {
// 	if runtime.GOOS == "windows" {
// 		// 'go-getter' continuously occupies files in '.git', causing the copy operation to fail
// 		opt := copy.Options{
// 			Skip: func(srcinfo os.FileInfo, src, dest string) (bool, error) {
// 				return filepath.Base(src) == constants.GitPathSuffix, nil
// 			},
// 		}
// 		return copy.Copy(depName, refName, opt)
// 	} else {
// 		return utils.CreateSymlink(depName, refName)
// 	}
// }
