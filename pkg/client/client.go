package client

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/dominikbraun/graph"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/otiai10/copy"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"oras.land/oras-go/v2"

	"kcl-lang.io/kpm/pkg/constants"
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
		logWriter: os.Stdout,
		settings:  *settings,
		homePath:  homePath,
	}, nil
}

// SetNoSumCheck will set the 'noSumCheck' flag.
func (c *KpmClient) SetNoSumCheck(noSumCheck bool) {
	c.noSumCheck = noSumCheck
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
		modFile.Dependencies.Deps = make(map[string]pkg.Dependency)
	}
	err = c.FillDependenciesInfo(modFile)
	if err != nil {
		return nil, err
	}

	return modFile, nil
}

func (c *KpmClient) LoadLockDeps(pkgPath string) (*pkg.Dependencies, error) {
	return pkg.LoadLockDeps(pkgPath)
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

// ResolveDepsMetadata will calculate the local storage path of the external package,
// and check whether the package exists locally.
// If the package does not exist, it will re-download to the local.
func (c *KpmClient) ResolvePkgDepsMetadata(kclPkg *pkg.KclPkg, update bool) error {
	var searchPath string
	kclPkg.NoSumCheck = c.noSumCheck

	if kclPkg.IsVendorMode() {
		// In the vendor mode, the search path is the vendor subdirectory of the current package.
		err := c.VendorDeps(kclPkg)
		if err != nil {
			return err
		}
		searchPath = kclPkg.LocalVendorPath()
	} else {
		// Otherwise, the search path is the $KCL_PKG_PATH.
		searchPath = c.homePath
	}

	// If under the mode of '--no_sum_check', the checksum of the package will not be checked.
	// There is no kcl.mod.lock, and the dependencies in kcl.mod and kcl.mod.lock do not need to be aligned.
	if !c.noSumCheck {
		// If not under the mode of '--no_sum_check',
		// all the dependencies in kcl.mod.lock are the dependencies of the current package.
		//
		// alian the dependencies between kcl.mod and kcl.mod.lock
		// clean the dependencies in kcl.mod.lock which not in kcl.mod
		// clean the dependencies in kcl.mod.lock and kcl.mod which have different version
		for name, dep := range kclPkg.Dependencies.Deps {
			modDep, ok := kclPkg.ModFile.Dependencies.Deps[name]
			if !ok || !dep.WithTheSameVersion(modDep) {
				reporter.ReportMsgTo(
					fmt.Sprintf("removing '%s' with version '%s'", name, dep.Version),
					c.logWriter,
				)
				delete(kclPkg.Dependencies.Deps, name)
			}
		}
		// add the dependencies in kcl.mod which not in kcl.mod.lock
		for name, d := range kclPkg.ModFile.Dependencies.Deps {
			if _, ok := kclPkg.Dependencies.Deps[name]; !ok {
				reporter.ReportMsgTo(
					fmt.Sprintf("adding '%s' with version '%s'", name, d.Version),
					c.logWriter,
				)
				kclPkg.Dependencies.Deps[name] = d
			}
		}
	} else {
		// If under the mode of '--no_sum_check', the checksum of the package will not be checked.
		// All the dependencies in kcl.mod are the dependencies of the current package.
		kclPkg.Dependencies.Deps = kclPkg.ModFile.Dependencies.Deps
	}

	for name, d := range kclPkg.Dependencies.Deps {
		searchFullPath := filepath.Join(searchPath, d.FullName)
		if !update {
			if d.IsFromLocal() {
				searchFullPath = d.GetLocalFullPath(kclPkg.HomePath)
			}

			// Find it and update the local path of the dependency.
			d.LocalFullPath = searchFullPath
			kclPkg.Dependencies.Deps[name] = d

		} else {
			if utils.DirExists(searchFullPath) && (c.GetNoSumCheck() || utils.CheckPackageSum(d.Sum, searchFullPath)) {
				// Find it and update the local path of the dependency.
				d.LocalFullPath = searchFullPath
				kclPkg.Dependencies.Deps[name] = d
			} else if d.IsFromLocal() && !utils.DirExists(d.GetLocalFullPath(kclPkg.HomePath)) {
				return reporter.NewErrorEvent(reporter.DependencyNotFound, fmt.Errorf("dependency '%s' not found in '%s'", d.Name, searchFullPath))
			} else if d.IsFromLocal() && utils.DirExists(d.GetLocalFullPath(kclPkg.HomePath)) {
				sum, err := utils.HashDir(d.GetLocalFullPath(kclPkg.HomePath))
				if err != nil {
					return reporter.NewErrorEvent(reporter.CalSumFailed, err, fmt.Sprintf("failed to calculate checksum for '%s' in '%s'", d.Name, searchFullPath))
				}
				d.Sum = sum
				kclPkg.Dependencies.Deps[name] = d
			} else {
				// Otherwise, re-vendor it.
				if kclPkg.IsVendorMode() {
					err := c.VendorDeps(kclPkg)
					if err != nil {
						return err
					}
				} else {
					// Or, re-download it.
					err := c.AddDepToPkg(kclPkg, &d)
					if err != nil {
						return err
					}
				}
				// After re-downloading or re-vendoring,
				// re-resolving is required to update the dependent paths.
				err := c.ResolvePkgDepsMetadata(kclPkg, update)
				if err != nil {
					return err
				}
				return nil
			}
		}
	}
	if update {
		// update the kcl.mod and kcl.mod.lock.
		err := kclPkg.UpdateModAndLockFile()
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateDeps will update the dependencies.
func (c *KpmClient) UpdateDeps(kclPkg *pkg.KclPkg) error {
	_, err := c.ResolveDepsMetadataInJsonStr(kclPkg, true)
	if err != nil {
		return err
	}

	err = kclPkg.UpdateModAndLockFile()
	if err != nil {
		return err
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

	kclPkg, err := pkg.LoadKclPkg(pkgPath)
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
	} else if len(kclPkg.GetEntryKclFilesFromModFile()) == 0 && !opts.HasSettingsYaml() {
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

	// 1. get the name and version of the repository from the input arguments.
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

	if !reflect.DeepEqual(kclPkg.ModFile.Dependencies.Deps[d.Name], *d) {
		// the dep passed on the cli is different from the kcl.mod.
		kclPkg.ModFile.Dependencies.Deps[d.Name] = *d
	}

	// download all the dependencies.
	changedDeps, _, err := c.InitGraphAndDownloadDeps(kclPkg)

	if err != nil {
		return err
	}

	// Update kcl.mod and kcl.mod.lock
	for k, v := range changedDeps.Deps {
		kclPkg.ModFile.Dependencies.Deps[k] = v
		kclPkg.Dependencies.Deps[k] = v
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
	err := utils.TarDir(kclPkg.HomePath, tarPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedPackage, err, "failed to package the kcl module")
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

	lockDeps := make([]pkg.Dependency, 0, len(kclPkg.Dependencies.Deps))

	for _, d := range kclPkg.Dependencies.Deps {
		lockDeps = append(lockDeps, d)
	}

	// Traverse all dependencies in kcl.mod.lock.
	for i := 0; i < len(lockDeps); i++ {
		d := lockDeps[i]
		if len(d.Name) == 0 {
			return errors.InvalidDependency
		}
		vendorFullPath := filepath.Join(vendorPath, d.FullName)
		// If the package already exists in the 'vendor', do nothing.
		if utils.DirExists(vendorFullPath) && check(d, vendorFullPath) {
			continue
		} else {
			// If not in the 'vendor', check the global cache.
			cacheFullPath := filepath.Join(c.homePath, d.FullName)
			if utils.DirExists(cacheFullPath) && check(d, cacheFullPath) {
				// If there is, copy it into the 'vendor' directory.
				err := copy.Copy(cacheFullPath, vendorFullPath)
				if err != nil {
					return err
				}
			} else if utils.DirExists(d.GetLocalFullPath(kclPkg.HomePath)) && check(d, d.GetLocalFullPath(kclPkg.HomePath)) {
				// If there is, copy it into the 'vendor' directory.
				err := copy.Copy(d.GetLocalFullPath(kclPkg.HomePath), vendorFullPath)
				if err != nil {
					return err
				}
			} else {
				// re-download if not.
				err = c.AddDepToPkg(kclPkg, &d)
				if err != nil {
					return err
				}
				// re-vendor again with new kcl.mod and kcl.mod.lock
				err = c.VendorDeps(kclPkg)
				if err != nil {
					return err
				}
				return nil
			}
		}
	}

	return nil
}

// FillDepInfo will fill registry information for a dependency.
func (c *KpmClient) FillDepInfo(dep *pkg.Dependency, homepath string) error {
	if dep.Source.Local != nil {
		dep.LocalFullPath = dep.Source.Local.Path
		return nil
	}
	if dep.Source.Oci != nil {
		dep.Source.Oci.Reg = c.GetSettings().DefaultOciRegistry()
		urlpath := utils.JoinPath(c.GetSettings().DefaultOciRepo(), dep.Name)
		dep.Source.Oci.Repo = urlpath
		manifest := ocispec.Manifest{}
		jsonDesc, err := c.FetchOciManifestIntoJsonStr(opt.OciFetchOptions{
			FetchBytesOptions: oras.DefaultFetchBytesOptions,
			OciOptions: opt.OciOptions{
				Reg:  c.GetSettings().DefaultOciRegistry(),
				Repo: fmt.Sprintf("%s/%s", c.GetSettings().DefaultOciRepo(), dep.Name),
				Tag:  dep.Version,
			},
		})

		if err != nil {
			return err
		}

		err = json.Unmarshal([]byte(jsonDesc), &manifest)
		if err != nil {
			return err
		}

		if value, ok := manifest.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_SUM]; ok {
			dep.Sum = value
		}
		return nil
	}
	return nil
}

// FillDependenciesInfo will fill registry information for all dependencies in a kcl.mod.
func (c *KpmClient) FillDependenciesInfo(modFile *pkg.ModFile) error {
	for k, v := range modFile.Deps {
		err := c.FillDepInfo(&v, modFile.HomePath)
		if err != nil {
			return err
		}
		modFile.Deps[k] = v
	}
	return nil
}

// Download will download the dependency to the local path.
func (c *KpmClient) Download(dep *pkg.Dependency, homePath, localPath string) (*pkg.Dependency, error) {
	if dep.Source.Git != nil {
		_, err := c.DownloadFromGit(dep.Source.Git, localPath)
		if err != nil {
			return nil, err
		}

		dep.LocalFullPath = localPath
		// Creating symbolic links in a global cache is not an optimal solution.
		// This allows kclvm to locate the package by default.
		// This feature is unstable and will be removed soon.
		err = createDepRef(dep.LocalFullPath, filepath.Join(filepath.Dir(localPath), dep.Name))
		if err != nil {
			return nil, err
		}
		dep.FullName = dep.GenDepFullName()
		// If the dependency is from git commit, the version is the commit id.
		// If the dependency is from git tag, the version is the tag.
		dep.Version, err = dep.Source.Git.GetValidGitReference()
		if err != nil {
			return nil, err
		}
	}

	if dep.Source.Oci != nil {
		kpkg, err := c.DownloadPkgFromOci(dep.Source.Oci, localPath)
		if err != nil {
			return nil, err
		}
		dep.FromKclPkg(kpkg)
		// Creating symbolic links in a global cache is not an optimal solution.
		// This allows kclvm to locate the package by default.
		// This feature is unstable and will be removed soon.
		err = createDepRef(dep.LocalFullPath, filepath.Join(filepath.Dir(localPath), dep.Name))
		if err != nil {
			return nil, err
		}
	}

	if dep.Source.Local != nil {
		kpkg, err := pkg.FindFirstKclPkgFrom(dep.GetLocalFullPath(homePath))
		if err != nil {
			return nil, err
		}
		dep.FromKclPkg(kpkg)
	}

	var err error
	dep.Sum, err = utils.HashDir(dep.LocalFullPath)
	if err != nil {
		return nil, reporter.NewErrorEvent(
			reporter.FailedHashPkg,
			err,
			fmt.Sprintf("failed to hash the kcl package '%s' in '%s'.", dep.Name, dep.LocalFullPath),
		)
	}

	return dep, nil
}

// DownloadFromGit will download the dependency from the git repository.
func (c *KpmClient) DownloadFromGit(dep *pkg.Git, localPath string) (string, error) {
	var msg string
	if len(dep.Tag) != 0 {
		msg = fmt.Sprintf("with tag '%s'", dep.Tag)
	}

	if len(dep.Commit) != 0 {
		msg = fmt.Sprintf("with commit '%s'", dep.Commit)
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
func (c *KpmClient) DownloadPkgFromOci(dep *pkg.Oci, localPath string) (*pkg.KclPkg, error) {
	ociClient, err := oci.NewOciClient(dep.Reg, dep.Repo, &c.settings)
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
func (c *KpmClient) DownloadFromOci(dep *pkg.Oci, localPath string) (string, error) {
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
				fmt.Sprintf("failed to find the kcl package tar from '%s'.", localPath),
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
			fmt.Sprintf("failed to untar the kcl package tar from '%s' into '%s'.", tarPath, localPath),
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
	ociCli, err := oci.NewOciClient(ociOpts.Reg, ociOpts.Repo, &c.settings)
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
	return oci.Login(hostname, username, password, &c.settings)
}

// LogoutOci will logout from the oci registry.
func (c *KpmClient) LogoutOci(hostname string) error {
	return oci.Logout(hostname, &c.settings)
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
func (c *KpmClient) InitGraphAndDownloadDeps(kclPkg *pkg.KclPkg) (*pkg.Dependencies, graph.Graph[string, string], error) {

	depGraph := graph.New(graph.StringHash, graph.Directed(), graph.PreventCycles())

	// add the root vertex(package name) to the dependency graph.
	root := fmt.Sprintf("%s@%s", kclPkg.GetPkgName(), kclPkg.GetPkgVersion())
	err := depGraph.AddVertex(root)
	if err != nil {
		return nil, nil, err
	}

	changedDeps, err := c.downloadDeps(kclPkg.ModFile.Dependencies, kclPkg.Dependencies, depGraph, kclPkg.HomePath, root)
	if err != nil {
		return nil, nil, err
	}

	return changedDeps, depGraph, nil
}

// dependencyExists will check whether the dependency exists in the local filesystem.
func (c *KpmClient) dependencyExists(dep *pkg.Dependency, lockDeps *pkg.Dependencies) *pkg.Dependency {

	// If the flag '--no_sum_check' is set, skip the checksum check.
	if c.noSumCheck {
		// If the dependent package does exist locally
		if utils.DirExists(filepath.Join(c.homePath, dep.FullName)) {
			return dep
		}
	}

	lockDep, present := lockDeps.Deps[dep.Name]
	// Check if the sum of this dependency in kcl.mod.lock has been changed.
	if !c.noSumCheck && present {
		// If the dependent package does not exist locally, then method 'check' will return false.
		if c.noSumCheck || check(lockDep, filepath.Join(c.homePath, dep.FullName)) {
			return &lockDep
		}
	}

	return nil
}

// downloadDeps will download all the dependencies of the current kcl package.
func (c *KpmClient) downloadDeps(deps pkg.Dependencies, lockDeps pkg.Dependencies, depGraph graph.Graph[string, string], pkghome, parent string) (*pkg.Dependencies, error) {

	newDeps := pkg.Dependencies{
		Deps: make(map[string]pkg.Dependency),
	}

	// Traverse all dependencies in kcl.mod
	for _, d := range deps.Deps {
		if len(d.Name) == 0 {
			return nil, errors.InvalidDependency
		}

		existDep := c.dependencyExists(&d, &lockDeps)
		if existDep != nil {
			newDeps.Deps[d.Name] = *existDep
			continue
		}

		expectedSum := lockDeps.Deps[d.Name].Sum
		// Clean the cache
		if len(c.homePath) == 0 || len(d.FullName) == 0 {
			return nil, errors.InternalBug
		}
		dir := filepath.Join(c.homePath, d.FullName)
		os.RemoveAll(dir)

		// download dependencies
		lockedDep, err := c.Download(&d, pkghome, dir)
		if err != nil {
			return nil, err
		}

		if !lockedDep.IsFromLocal() {
			if !c.noSumCheck && expectedSum != "" &&
				lockedDep.Sum != expectedSum &&
				existDep != nil &&
				existDep.FullName == d.FullName {
				return nil, reporter.NewErrorEvent(
					reporter.CheckSumMismatch,
					errors.CheckSumMismatchError,
					fmt.Sprintf("checksum for '%s' changed in lock file", lockedDep.Name),
				)
			}
		}

		// Update kcl.mod and kcl.mod.lock
		newDeps.Deps[d.Name] = *lockedDep
		lockDeps.Deps[d.Name] = *lockedDep
	}

	// necessary to make a copy as when we are updating kcl.mod in below for loop
	// then newDeps.Deps gets updated and range gets an extra value to iterate through
	// this messes up the dependency graph
	newDepsCopy := make(map[string]pkg.Dependency)
	for k, v := range newDeps.Deps {
		newDepsCopy[k] = v
	}

	// Recursively download the dependencies of the new dependencies.
	for _, d := range newDepsCopy {
		// Load kcl.mod file of the new downloaded dependencies.
		deppkg, err := pkg.LoadKclPkg(filepath.Join(c.homePath, d.FullName))
		if len(d.LocalFullPath) != 0 {
			deppkg, err = pkg.LoadKclPkg(d.LocalFullPath)
		}

		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		source := fmt.Sprintf("%s@%s", d.Name, d.Version)
		source = strings.TrimRight(source, "@")
		err = depGraph.AddVertex(source)
		if err != nil && err != graph.ErrVertexAlreadyExists {
			return nil, err
		}

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

		// Download the dependencies.
		nested, err := c.downloadDeps(deppkg.ModFile.Dependencies, lockDeps, depGraph, deppkg.HomePath, source)
		if err != nil {
			return nil, err
		}

		// Update kcl.mod.
		for _, d := range nested.Deps {
			if _, ok := newDeps.Deps[d.Name]; !ok {
				newDeps.Deps[d.Name] = d
			}
		}
	}

	return &newDeps, nil
}

// pullTarFromOci will pull a kcl package tar file from oci registry.
func (c *KpmClient) pullTarFromOci(localPath string, ociOpts *opt.OciOptions) error {
	absPullPath, err := filepath.Abs(localPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err)
	}

	ociCli, err := oci.NewOciClient(ociOpts.Reg, ociOpts.Repo, &c.settings)
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
	ociCli, err := oci.NewOciClient(opts.Reg, opts.Repo, &c.settings)
	if err != nil {
		return "", err
	}

	manifestJson, err := ociCli.FetchManifestIntoJsonStr(opts)
	if err != nil {
		return "", err
	}
	return manifestJson, nil
}

// check sum for a Dependency.
func check(dep pkg.Dependency, newDepPath string) bool {
	if dep.Sum == "" {
		return false
	}

	sum, err := utils.HashDir(newDepPath)

	if err != nil {
		return false
	}

	return dep.Sum == sum
}

// createDepRef will create a dependency reference for the dependency saved on the local filesystem.
// On the unix-like system, it will create a symbolic link.
// On the windows system, it will create a junction.
func createDepRef(depName, refName string) error {
	if runtime.GOOS == "windows" {
		// 'go-getter' continuously occupies files in '.git', causing the copy operation to fail
		opt := copy.Options{
			Skip: func(srcinfo os.FileInfo, src, dest string) (bool, error) {
				return filepath.Base(src) == constants.GitPathSuffix, nil
			},
		}
		return copy.Copy(depName, refName, opt)
	} else {
		return utils.CreateSymlink(depName, refName)
	}
}
