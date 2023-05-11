package pkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/otiai10/copy"
	"kusionstack.io/kpm/pkg/env"
	errors "kusionstack.io/kpm/pkg/errors"
	modfile "kusionstack.io/kpm/pkg/mod"
	"kusionstack.io/kpm/pkg/opt"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/runner"
	"kusionstack.io/kpm/pkg/utils"
)

type KclPkg struct {
	modFile  modfile.ModFile
	HomePath string
	// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
	// not the dependencies in kcl.mod.
	modfile.Dependencies
}

func NewKclPkg(opts *opt.InitOptions) KclPkg {
	return KclPkg{
		modFile:      *modfile.NewModFile(opts),
		HomePath:     opts.InitPath,
		Dependencies: modfile.Dependencies{Deps: make(map[string]modfile.Dependency)},
	}
}

// Load the kcl package from directory containing kcl.mod and kcl.mod.lock file.
func LoadKclPkg(pkgPath string) (*KclPkg, error) {
	modFile, err := modfile.LoadModFile(pkgPath)
	if err != nil {
		return nil, err
	}

	// Get dependencies from kcl.mod.lock.
	deps, err := modfile.LoadLockDeps(pkgPath)

	if err != nil {
		return nil, err
	}

	return &KclPkg{
		modFile:      *modFile,
		HomePath:     pkgPath,
		Dependencies: *deps,
	}, nil
}

// LoadKclPkgFromTar will load a kcl package from a tar path.
func LoadKclPkgFromTar(pkgTarPath string) (*KclPkg, error) {
	destDir := strings.TrimSuffix(pkgTarPath, filepath.Ext(pkgTarPath))
	err := utils.UnTarDir(pkgTarPath, destDir)
	if err != nil {
		return nil, err
	}
	return LoadKclPkg(destDir)
}

func (kclPkg *KclPkg) IsVendorMode() bool {
	return kclPkg.modFile.VendorMode
}

func (kclPkg *KclPkg) SetVendorMode(vendorMode bool) {
	kclPkg.modFile.VendorMode = vendorMode
}

// Return the full vendor path.
func (kclPkg *KclPkg) LocalVendorPath() string {
	return filepath.Join(kclPkg.HomePath, "vendor")
}

// CompileWithEntryFile will call 'kclvm_cli' to compile the current kcl package and its dependent packages.
func (kclPkg *KclPkg) CompileWithEntryFile(kpmHome string, kclvmCompiler *runner.CompileCmd) (string, error) {

	pkgMap, err := kclPkg.ResolveDeps(kpmHome)
	if err != nil {
		return "", err
	}

	for dName, dPath := range pkgMap {
		kclvmCompiler.AddDepPath(dName, dPath)
	}

	return kclvmCompiler.Run(), nil
}

// ResolveDeps will return a map between dependency name and its local path,
// and analyze the dependencies of the current kcl package,
// look for the package in the $KCL_PKG_PATH or kcl package vendor subdirectory,
// if find it, ResolveDeps will remember the local path of the dependency,
// if can’t find it, re-download the dependency and remember the local path.
func (kclPkg *KclPkg) ResolveDeps(kpmHome string) (map[string]string, error) {

	err := kclPkg.ResolveDepsMetadata(kpmHome, true)
	if err != nil {
		return nil, err
	}

	var pkgMap map[string]string = make(map[string]string)
	for name, d := range kclPkg.Dependencies.Deps {
		pkgMap[name] = d.LocalFullPath
	}

	return pkgMap, nil
}

// ResolveDepsMetadata will calculate the local storage path of the external package,
// and check whether the package exists locally.
// If the package does not exist, it will re-download to the local.
func (kclPkg *KclPkg) ResolveDepsMetadata(kpmHome string, update bool) error {
	var searchPath string
	if kclPkg.IsVendorMode() {
		// In the vendor mode, the search path is the vendor subdirectory of the current package.
		err := kclPkg.VendorDeps(kpmHome)
		if err != nil {
			return err
		}
		searchPath = kclPkg.LocalVendorPath()
	} else {
		// Otherwise, the search path is the $KCL_PKG_PATH.
		searchPath = kpmHome
	}

	for name, d := range kclPkg.Dependencies.Deps {
		searchFullPath := filepath.Join(searchPath, d.FullName)
		if !update {
			if utils.DirExists(searchFullPath) {
				// Find it and update the local path of the dependency.
				d.LocalFullPath = searchFullPath
				kclPkg.Dependencies.Deps[name] = d
			}
		} else {
			if utils.DirExists(searchFullPath) && check(d, searchFullPath) {
				// Find it and update the local path of the dependency.
				d.LocalFullPath = searchFullPath
				kclPkg.Dependencies.Deps[name] = d
			} else {
				// Otherwise, re-vendor it.
				if kclPkg.IsVendorMode() {
					err := kclPkg.VendorDeps(kpmHome)
					if err != nil {
						return err
					}
				} else {
					// Or, re-download it.
					err := kclPkg.DownloadDep(&d, kpmHome)
					if err != nil {
						return err
					}
				}
				// After re-downloading or re-vendoring,
				// re-resolving is required to update the dependent paths.
				err := kclPkg.ResolveDepsMetadata(kpmHome, update)
				if err != nil {
					return err
				}
				return nil
			}
		}
	}

	return nil
}

// ResolveDepsMetadataInJsonStr will calculate the local storage path of the external package,
// and check whether the package exists locally. If the package does not exist, it will re-download to the local.
// Finally, the calculated metadata of the dependent packages is serialized into a json string and returned.
func (kclPkg *KclPkg) ResolveDepsMetadataInJsonStr(kpmHome string, update bool) (string, error) {
	// 1. Calculate the dependency path, check whether the dependency exists
	// and re-download the dependency that does not exist.
	err := kclPkg.ResolveDepsMetadata(kpmHome, update)
	if err != nil {
		return "", err
	}

	// 2. Serialize to JSON
	jsonData, err := json.Marshal(kclPkg.Dependencies)
	if err != nil {
		return "", errors.InternalBug
	}

	return string(jsonData), nil
}

// InitEmptyModule inits an empty kcl module and create a default kcl.modfile.
func (kclPkg KclPkg) InitEmptyPkg() error {
	err := utils.CreateFileIfNotExist(
		kclPkg.modFile.GetModFilePath(),
		kclPkg.modFile.StoreModFile,
	)
	if err != nil {
		return err
	}

	err = utils.CreateFileIfNotExist(
		kclPkg.modFile.GetModLockFilePath(),
		kclPkg.LockDepsVersion,
	)
	if err != nil {
		return err
	}

	return nil
}

// AddDeps will add the dependencies to current kcl package and update kcl.mod and kcl.mod.lock.
func (kclPkg *KclPkg) AddDeps(opt *opt.AddOptions) error {
	// Get the name and version of the repository from the input arguments.
	d := modfile.ParseOpt(&opt.RegistryOpts)
	err := kclPkg.DownloadDep(d, opt.LocalPath)
	if err != nil {
		return err
	}
	err = kclPkg.UpdateModAndLockFile()
	if err != nil {
		return err
	}
	return nil
}

// DownloadDep will download the corresponding dependency.
func (kclPkg *KclPkg) DownloadDep(d *modfile.Dependency, localPath string) error {

	if !reflect.DeepEqual(kclPkg.modFile.Dependencies.Deps[d.Name], *d) {
		// the dep passed on the cli is different from the kcl.mod.
		kclPkg.modFile.Dependencies.Deps[d.Name] = *d
	}

	// download all the dependencies.
	changedDeps, err := getDeps(kclPkg.modFile.Dependencies, kclPkg.Dependencies, localPath)

	if err != nil {
		return errors.FailedDownloadError
	}

	// Update kcl.mod and kcl.mod.lock
	for k, v := range changedDeps.Deps {
		kclPkg.modFile.Dependencies.Deps[k] = v
		kclPkg.Dependencies.Deps[k] = v
	}

	return err
}

// updateModAndLockFile will update kcl.mod and kcl.mod.lock
func (kclPkg *KclPkg) UpdateModAndLockFile() error {
	// Generate file kcl.mod.
	err := kclPkg.modFile.StoreModFile()
	if err != nil {
		return err
	}

	// Generate file kcl.mod.lock.
	err = kclPkg.LockDepsVersion()
	if err != nil {
		return err
	}

	return nil
}

// LockDepsVersion locks the dependencies of the current kcl package into kcl.mod.lock.
func (kclPkg *KclPkg) LockDepsVersion() error {
	fullPath := filepath.Join(kclPkg.HomePath, modfile.MOD_LOCK_FILE)
	lockToml, err := kclPkg.Dependencies.MarshalLockTOML()
	if err != nil {
		return err
	}

	return utils.StoreToFile(fullPath, lockToml)
}

// getDeps will recursively download all dependencies to the 'localPath' directory,
// and return the dependencies that need to be updated to kcl.mod and kcl.mod.lock.
func getDeps(deps modfile.Dependencies, lockDeps modfile.Dependencies, localPath string) (*modfile.Dependencies, error) {
	newDeps := modfile.Dependencies{
		Deps: make(map[string]modfile.Dependency),
	}

	// Traverse all dependencies in kcl.mod
	for _, d := range deps.Deps {
		if len(d.Name) == 0 {
			return nil, errors.InvalidDependency
		}

		lockDep, present := lockDeps.Deps[d.Name]

		// Check if the sum of this dependency in kcl.mod.lock has been chanaged.
		if present {
			// If the dependent package does not exist locally, then method 'check' will return false.
			if check(lockDep, filepath.Join(localPath, d.FullName)) {
				newDeps.Deps[d.Name] = lockDep
				continue
			}
		}
		expectedSum := lockDeps.Deps[d.Name].Sum
		// Clean the cache
		if len(localPath) == 0 || len(d.FullName) == 0 {
			return nil, errors.InternalBug
		}
		dir := filepath.Join(localPath, d.FullName)
		os.RemoveAll(dir)

		// download dependencies
		lockedDep, err := d.Download(dir)
		if err != nil {
			return nil, errors.FailedDownloadError
		}
		if expectedSum != "" && lockedDep.Sum != expectedSum && lockDep.FullName == d.FullName {
			return nil, errors.CheckSumMismatchError
		}

		// Update kcl.mod and kcl.mod.lock
		newDeps.Deps[d.Name] = *lockedDep
		lockDeps.Deps[d.Name] = *lockedDep
	}

	// Recursively download the dependencies of the new dependencies.
	for _, d := range newDeps.Deps {
		// Load kcl.mod file of the new downloaded dependencies.
		modfile, err := modfile.LoadModFile(filepath.Join(localPath, d.FullName))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		// Download the dependencies.
		nested, err := getDeps(modfile.Dependencies, lockDeps, localPath)
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

// check sum for a Dependency.
func check(dep modfile.Dependency, newDepPath string) bool {
	if dep.Sum == "" {
		return false
	}

	sum, err := utils.HashDir(newDepPath)

	if err != nil {
		return false
	}

	return dep.Sum == sum
}

// PackageCurrentPkg will package the current kcl package into the current path and return the tar path.
// And the tar will be named "<package_name>-<package_version>.tar"
// <package_name> is the package name specified in kcl.mod.
// <package_version> is the package version specified in kcl.mod.
func (kclPkg *KclPkg) PackageCurrentPkgPath() (string, error) {
	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return "", err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != nil {
		return "", err
	}

	err = kclPkg.PackageKclPkg(globalPkgPath, kclPkg.DefaultTarPath())

	if err != nil {
		reporter.ExitWithReport("kpm: failed to package pkg " + kclPkg.GetPkgName() + ".")
		return "", err
	}
	return kclPkg.DefaultTarPath(), nil
}

const TAR_SUFFIX = ".tar"
const VERSION_PREFFIX = "v"

// DefaultTarPath will return "<kcl_package_path>/<package_name>-<package_version>.tar"
func (kclPkg *KclPkg) DefaultTarPath() string {
	return filepath.Join(kclPkg.HomePath, kclPkg.GetPkgTarName())
}

// PkgCurrentPackageIntoTarPath will package the current kcl package into 'tarPath'.
func (kclPkg *KclPkg) PackageToTarball(tarPath string) error {

	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != nil {
		return err
	}

	err = kclPkg.PackageKclPkg(globalPkgPath, tarPath)

	if err != nil {
		reporter.ExitWithReport("kpm: failed to package pkg " + kclPkg.GetPkgName() + ".")
		return err
	}
	return nil
}

// PackageKclPkg will save all dependencies to the 'vendor' in current pacakge
// and package the current package into tar
func (kclPkg *KclPkg) PackageKclPkg(kpmHome string, tarPath string) error {
	// Vendor all the dependencies into the current kcl package.
	err := kclPkg.VendorDeps(kpmHome)
	if err != nil {
		return errors.FailedToVendorDependency
	}

	// Tar the current kcl package into a "*.tar" file.
	err = utils.TarDir(kclPkg.HomePath, tarPath)
	if err != nil {
		return errors.FailedToPackage
	}
	return nil
}

// Vendor all dependencies to the 'vendor' in current pacakge.
func (kclPkg *KclPkg) VendorDeps(cachePath string) error {
	// Mkdir the dir "vendor".
	vendorPath := kclPkg.LocalVendorPath()
	err := os.MkdirAll(vendorPath, 0755)
	if err != nil {
		return errors.InternalBug
	}

	lockDeps := make([]modfile.Dependency, 0, len(kclPkg.Dependencies.Deps))

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
			cacheFullPath := filepath.Join(cachePath, d.FullName)
			if utils.DirExists(cacheFullPath) && check(d, cacheFullPath) {
				// If there is, copy it into the 'vendor' directory.
				err := copy.Copy(cacheFullPath, vendorFullPath)
				if err != nil {
					return errors.FailedToVendorDependency
				}
			} else {
				// re-download if not.
				err = kclPkg.DownloadDep(&d, cachePath)
				if err != nil {
					return errors.FailedToVendorDependency
				}
				// re-vendor again with new kcl.mod and kcl.mod.lock
				err = kclPkg.VendorDeps(cachePath)
				if err != nil {
					return errors.FailedToVendorDependency
				}
				return nil
			}
		}
	}

	return nil
}

// Verify that the environment variable KPM HOME is set correctly
func (kclPkg *KclPkg) ValidateKpmHome(kpmHome string) error {
	if kclPkg.HomePath == kpmHome {
		return errors.InvalidKpmHomeInCurrentPkg
	}
	return nil
}

// GetPkgFullName returns the full name of package.
// The full name is "<pkg_name>-<pkg_version>",
// <pkg_name> is the name of package.
// <pkg_version> is the version of package
func (kclPkg *KclPkg) GetPkgFullName() string {
	return kclPkg.modFile.Pkg.Name + "-" + VERSION_PREFFIX + kclPkg.modFile.Pkg.Version
}

// GetPkgName returns name of package.
func (kclPkg *KclPkg) GetPkgName() string {
	return kclPkg.modFile.Pkg.Name
}

// GetPkgTag returns version of package.
func (kclPkg *KclPkg) GetPkgTag() string {
	return kclPkg.modFile.Pkg.Version
}

// GetPkgTagForOci returns version of package in OCI format.
// The version of a package is "0.0.1".
// The version of a package in OCI format is "v0.0.1"
func (kclPkg *KclPkg) GetOciPkgTag() string {
	if !strings.HasPrefix(kclPkg.GetPkgTag(), VERSION_PREFFIX) {
		return VERSION_PREFFIX + kclPkg.GetPkgTag()
	} else {
		return kclPkg.GetPkgTag()
	}
}

// GetPkgTarName returns the kcl package tar name "<package_name>-v<package_version>.tar"
func (kclPkg *KclPkg) GetPkgTarName() string {
	return kclPkg.GetPkgFullName() + TAR_SUFFIX
}
