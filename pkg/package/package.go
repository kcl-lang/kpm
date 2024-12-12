package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v2"
	"github.com/jinzhu/copier"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/downloader"
	errors "kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

var TestPkgDependency = Dependency{
	Name:     "kcl",
	FullName: "kcl",
	Version:  "0.0.0",
	Sum:      "Sum",
}

type KclPkg struct {
	ModFile  ModFile
	HomePath string
	// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
	// not the dependencies in kcl.mod.
	Dependencies
	// The flag 'NoSumCheck' is true if the checksum of the current kcl package is not checked.
	NoSumCheck bool
	// A snapshot of the dependencies in kcl.mod
	// readonly and user can't modify it.
	depUI DependenciesUI
}

func (p *KclPkg) BackupDepUI(name string, dep *Dependency) {
	p.depUI.Deps[name] = *dep
}

type LoadOptions struct {
	// The module path.
	Path string
	// The settings with default oci registry.
	Settings *settings.Settings
}

type LoadOption func(*LoadOptions)

// WithPath sets the module path.
func WithPath(path string) LoadOption {
	return func(opts *LoadOptions) {
		opts.Path = path
	}
}

// WithSettings sets the settings with default oci registry.
func WithSettings(settings *settings.Settings) LoadOption {
	return func(opts *LoadOptions) {
		opts.Settings = settings
	}
}

// LoadKclPkgWithOpts loads a module from the file system with options.
// The options include the module path and the settings with default oci registry.
func LoadKclPkgWithOpts(options ...LoadOption) (*KclPkg, error) {
	opts := &LoadOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pkgPath := opts.Path
	if opts.Settings == nil {
		opts.Settings = settings.GetSettings()
	}

	modFile := new(ModFile)
	err := modFile.LoadModFile(filepath.Join(pkgPath, MOD_FILE))
	if err != nil {
		return nil, fmt.Errorf("could not load 'kcl.mod' in '%s'\n%w", pkgPath, err)
	}

	// load the kcl.mod.lock file.
	// Get dependencies from kcl.mod.lock.
	deps, err := LoadLockDeps(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("could not load 'kcl.mod' in '%s'\n%w", pkgPath, err)
	}

	// Save snapshot of the dependencies in kcl.mod.
	depsUI := DependenciesUI{
		Deps: make(map[string]Dependency),
	}

	for _, name := range modFile.Deps.Keys() {
		dep, ok := modFile.Deps.Get(name)
		if !ok {
			return nil, fmt.Errorf("could not load 'kcl.mod' in '%s'\n%w", pkgPath, err)
		}
		depSnap := Dependency{}
		err := copier.Copy(&depSnap, &dep)
		if err != nil {
			fmt.Printf("failed to copy dependency: %v\n", err)
			continue
		}
		depsUI.Deps[name] = depSnap
	}

	// pre-process the package.
	// 1. Transform the local path to the absolute path.
	err = convertDepsLocalPathToAbsPath(&modFile.Dependencies, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("could not load 'kcl.mod' in '%s'\n%w", pkgPath, err)
	}
	// 2. Fill the default oci registry, the default oci registry is in the settings.
	err = fillDepsInfoWithSettings(&modFile.Dependencies, opts.Settings)
	if err != nil {
		return nil, fmt.Errorf("could not load 'kcl.mod' in '%s'\n%w", pkgPath, err)
	}
	// 3. Sync the dependencies information in kcl.mod.lock with the dependencies in kcl.mod.
	for _, name := range deps.Deps.Keys() {
		lockDep, ok := deps.Deps.Get(name)
		if !ok {
			return nil, fmt.Errorf("could not load 'kcl.mod' in '%s'\n%w", pkgPath, err)
		}
		if modDep, ok := modFile.Dependencies.Deps.Get(name); ok {
			lockDep.Source = modDep.Source
			lockDep.LocalFullPath = modDep.LocalFullPath
		} else {
			// If there is no source in the lock file, fill the default oci registry.
			if lockDep.Source.IsNilSource() {
				lockDep.Source = downloader.Source{
					ModSpec: &downloader.ModSpec{
						Name:    lockDep.Name,
						Version: lockDep.Version,
					},
					Oci: &downloader.Oci{
						Reg:  opts.Settings.DefaultOciRegistry(),
						Repo: utils.JoinPath(opts.Settings.DefaultOciRepo(), lockDep.Name),
						Tag:  lockDep.Version,
					},
				}
			}
		}
		deps.Deps.Set(name, lockDep)
	}

	return &KclPkg{
		ModFile:      *modFile,
		HomePath:     pkgPath,
		Dependencies: *deps,
		depUI:        depsUI,
	}, nil
}

// LoadAndFillModFileWithOpts loads a mod file from the file system with options.
// It will load the mod file, convert the local path to the absolute path, and fill the default oci registry.
func LoadAndFillModFileWithOpts(options ...LoadOption) (*ModFile, error) {
	opts := &LoadOptions{}
	for _, opt := range options {
		opt(opts)
	}

	path := opts.Path

	// Load the mod file.
	// The content of the `ModFile` is the same as the content in kcl.mod
	// The `ModFile` lacks some information of the dependencies.
	modFile := new(ModFile)
	err := modFile.LoadModFile(filepath.Join(path, MOD_FILE))
	if err != nil {
		return nil, fmt.Errorf("could not load 'kcl.mod' in '%s'\n%w", path, err)
	}

	// pre-process the package.
	// 1. Transform the local path to the absolute path.
	err = convertDepsLocalPathToAbsPath(&modFile.Dependencies, path)
	if err != nil {
		return nil, fmt.Errorf("could not load 'kcl.mod' in '%s'\n%w", path, err)
	}
	// 2. Fill the default oci registry, the default oci registry is in the settings.
	err = fillDepsInfoWithSettings(&modFile.Dependencies, opts.Settings)
	if err != nil {
		return nil, fmt.Errorf("could not load 'kcl.mod' in '%s'\n%w", path, err)
	}

	return modFile, nil
}

// `convertDepsLocalPathToAbsPath` will transform the local path to the absolute path from `rootPath` in dependencies.
func convertDepsLocalPathToAbsPath(deps *Dependencies, rootPath string) error {
	for _, name := range deps.Deps.Keys() {
		dep, ok := deps.Deps.Get(name)
		if !ok {
			break
		}
		// Transform the local path to the absolute path.
		if dep.Local != nil {
			var localFullPath string
			var err error
			if filepath.IsAbs(dep.Local.Path) {
				localFullPath = dep.Local.Path
			} else {
				localFullPath, err = filepath.Abs(filepath.Join(rootPath, dep.Local.Path))
				if err != nil {
					return fmt.Errorf("failed to get the absolute path of the local dependency %s: %w", name, err)
				}
			}
			dep.LocalFullPath = localFullPath
		}
		deps.Deps.Set(name, dep)
	}

	return nil
}

// `fillDepsInfoWithSettings` will fill the default oci registry info in dependencies.
func fillDepsInfoWithSettings(deps *Dependencies, settings *settings.Settings) error {
	for _, name := range deps.Deps.Keys() {
		dep, ok := deps.Deps.Get(name)
		if !ok {
			break
		}
		// Fill the default oci registry.
		if dep.Source.Oci != nil {
			if len(dep.Source.Oci.Reg) == 0 {
				dep.Source.Oci.Reg = settings.DefaultOciRegistry()
			}

			if len(dep.Source.Oci.Repo) == 0 {
				urlpath := utils.JoinPath(settings.DefaultOciRepo(), dep.Name)
				dep.Source.Oci.Repo = urlpath
			}
		}
		if dep.Source.SpecOnly() {
			dep.Source.Oci = &downloader.Oci{
				Reg:  settings.DefaultOciRegistry(),
				Repo: utils.JoinPath(settings.DefaultOciRepo(), dep.ModSpec.Name),
				Tag:  dep.ModSpec.Version,
			}
		}
		dep.FullName = dep.GenDepFullName()
		deps.Deps.Set(name, dep)
	}

	return nil
}

// GenOciManifestFromPkg will generate the oci manifest from the kcl package.
func (kclPkg *KclPkg) GenOciManifestFromPkg() (map[string]string, error) {
	res := make(map[string]string)
	res[constants.DEFAULT_KCL_OCI_MANIFEST_NAME] = kclPkg.GetPkgName()
	res[constants.DEFAULT_KCL_OCI_MANIFEST_VERSION] = kclPkg.GetPkgVersion()
	res[constants.DEFAULT_KCL_OCI_MANIFEST_DESCRIPTION] = kclPkg.GetPkgDescription()
	sum, err := kclPkg.GenCheckSum()
	if err != nil {
		return nil, err
	}
	res[constants.DEFAULT_KCL_OCI_MANIFEST_SUM] = sum
	return res, nil
}

func (p *KclPkg) GetDepsMetadata() (*DependenciesUI, error) {
	return p.Dependencies.ToDepMetadata()
}

func NewKclPkg(opts *opt.InitOptions) KclPkg {
	return KclPkg{
		ModFile:      *NewModFile(opts),
		HomePath:     opts.InitPath,
		Dependencies: Dependencies{Deps: orderedmap.NewOrderedMap[string, Dependency]()},
	}
}

// LoadKclPkg will load a package from the 'pkgPath'
// The default oci registry in '$KCL_PKG_PATH/.kpm/config/kpm.json' will be used.
func LoadKclPkg(path string) (*KclPkg, error) {
	return LoadKclPkgWithOpts(WithPath(path), WithSettings(settings.GetSettings()))
}

// FindFirstKclPkgFrom will find the first kcl package from the 'path'
// The default oci registry in '$KCL_PKG_PATH/.kpm/config/kpm.json' will be used.
func FindFirstKclPkgFrom(pkgpath string) (*KclPkg, error) {
	matches, _ := filepath.Glob(filepath.Join(pkgpath, "*.tar"))
	if matches == nil || len(matches) != 1 {
		// then try to glob tgz file
		matches, _ = filepath.Glob(filepath.Join(pkgpath, "*.tgz"))
		if matches == nil || len(matches) != 1 {
			pkg, err := LoadKclPkg(pkgpath)
			if err != nil {
				return nil, reporter.NewErrorEvent(
					reporter.InvalidKclPkg,
					err,
					fmt.Sprintf("failed to find the kcl package tar from '%s'.", pkgpath),
				)
			}

			return pkg, nil
		}
	}

	tarPath := matches[0]
	unTarPath := filepath.Dir(tarPath)
	var err error
	if utils.IsTar(tarPath) {
		err = utils.UnTarDir(tarPath, unTarPath)
	} else {
		err = utils.ExtractTarball(tarPath, unTarPath)
	}
	if err != nil {
		return nil, reporter.NewErrorEvent(
			reporter.FailedUntarKclPkg,
			err,
			fmt.Sprintf("failed to untar the kcl package tar from '%s' into '%s'.", tarPath, unTarPath),
		)
	}

	// After untar the downloaded kcl package tar file, remove the tar file.
	if utils.DirExists(tarPath) {
		rmErr := os.Remove(tarPath)
		if rmErr != nil {
			return nil, reporter.NewErrorEvent(
				reporter.FailedUntarKclPkg,
				err,
				fmt.Sprintf("failed to untar the kcl package tar from '%s' into '%s'.", tarPath, unTarPath),
			)
		}
	}

	pkg, err := LoadKclPkg(unTarPath)
	if err != nil {
		return nil, reporter.NewErrorEvent(
			reporter.InvalidKclPkg,
			err,
			fmt.Sprintf("failed to find the kcl package tar from '%s'.", pkgpath),
		)
	}

	return pkg, nil
}

// LoadKclPkgFromTar loads a package *.tar file from the 'pkgTarPath'
// The default oci registry in '$KCL_PKG_PATH/.kpm/config/kpm.json' will be used.
func LoadKclPkgFromTar(pkgTarPath string) (*KclPkg, error) {
	destDir := strings.TrimSuffix(pkgTarPath, filepath.Ext(pkgTarPath))
	err := utils.UnTarDir(pkgTarPath, destDir)
	if err != nil {
		return nil, err
	}
	return LoadKclPkg(destDir)
}

// GetKclOpts will return the kcl options from kcl.mod.
func (kclPkg *KclPkg) GetKclOpts() *kcl.Option {
	if kclPkg.ModFile.Profiles == nil {
		return kcl.NewOption()
	}
	return kclPkg.ModFile.Profiles.IntoKclOptions()
}

// GetEntryKclFilesFromModFile will return the entry kcl files from kcl.mod.
func (kclPkg *KclPkg) GetEntryKclFilesFromModFile() []string {
	return kclPkg.ModFile.GetEntries()
}

// HasProfile will return true if the current kcl package has the profile.
func (kclPkg *KclPkg) HasProfile() bool {
	return kclPkg.ModFile.Profiles != nil
}

func (kclPkg *KclPkg) IsVendorMode() bool {
	return kclPkg.ModFile.VendorMode
}

func (kclPkg *KclPkg) SetVendorMode(vendorMode bool) {
	kclPkg.ModFile.VendorMode = vendorMode
}

// Return the full vendor path.
func (kclPkg *KclPkg) LocalVendorPath() string {
	return filepath.Join(kclPkg.HomePath, "vendor")
}

// UpdateModFile will update the kcl.mod file.
func (kclPkg *KclPkg) UpdateModFile() error {
	// Load kcl.mod SnapShot.
	depSnapShot := kclPkg.depUI

	for _, name := range kclPkg.ModFile.Deps.Keys() {
		modDep, ok := kclPkg.ModFile.Deps.Get(name)
		if !ok {
			return fmt.Errorf("failed to get dependency %s", name)
		}

		if existDep, ok := depSnapShot.Deps[name]; ok {
			existDep.Source.ModSpec = modDep.ModSpec
			if !existDep.Source.SpecOnly() {
				existDep.Source = modDep.Source
			}
			kclPkg.ModFile.Deps.Set(name, existDep)
		}
	}

	// Generate file kcl.mod.
	err := kclPkg.ModFile.StoreModFile()
	if err != nil {
		return err
	}

	return nil
}

// updateModAndLockFile will update kcl.mod and kcl.mod.lock
func (kclPkg *KclPkg) UpdateModAndLockFile() error {
	err := kclPkg.UpdateModFile()
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

// LockDepsVersion locks the dependencies of the current kcl package into kcl.mod.lock.
func (kclPkg *KclPkg) LockDepsVersion() error {
	fullPath := filepath.Join(kclPkg.HomePath, MOD_LOCK_FILE)
	lockToml, err := kclPkg.Dependencies.MarshalLockTOML()
	if err != nil {
		return err
	}

	return utils.StoreToFile(fullPath, lockToml)
}

// CreateDefaultMain will create a default main.k file in the current kcl package.
func (kclPkg *KclPkg) CreateDefaultMain() error {
	mainKPath := filepath.Join(kclPkg.HomePath, constants.DEFAULT_KCL_FILE_NAME)
	return utils.StoreToFile(mainKPath, constants.DEFAULT_KCL_FILE_CONTENT)
}

// check sum for a Dependency.
func check(dep Dependency, newDepPath string) bool {
	if dep.Sum == "" {
		return false
	}

	sum, err := utils.HashDir(newDepPath)

	if err != nil {
		return false
	}

	return dep.Sum == sum
}

const TAR_SUFFIX = ".tar"

// DefaultTarPath will return "<kcl_package_path>/<package_name>-<package_version>.tar"
func (kclPkg *KclPkg) DefaultTarPath() string {
	return filepath.Join(kclPkg.HomePath, kclPkg.GetPkgTarName())
}

// Verify that the environment variable KPM HOME is set correctly
func (kclPkg *KclPkg) ValidateKpmHome(kpmHome string) *reporter.KpmEvent {
	if kclPkg.HomePath == kpmHome {
		return reporter.NewErrorEvent(reporter.InvalidKpmHomeInCurrentPkg, errors.InvalidKpmHomeInCurrentPkg)
	}
	return nil
}

// GetPkgFullName returns the full name of package.
// The full name is "<pkg_name>-<pkg_version>",
// <pkg_name> is the name of package.
// <pkg_version> is the version of package
func (kclPkg *KclPkg) GetPkgFullName() string {
	return fmt.Sprintf(PKG_NAME_PATTERN, kclPkg.ModFile.Pkg.Name, kclPkg.ModFile.Pkg.Version)
}

// GetPkgName returns name of package.
func (kclPkg *KclPkg) GetPkgName() string {
	return kclPkg.ModFile.Pkg.Name
}

// GetPkgTag returns version of package.
func (kclPkg *KclPkg) GetPkgTag() string {
	return kclPkg.ModFile.Pkg.Version
}

// GetPkgEdition returns compile edition of package.
func (kclPkg *KclPkg) GetPkgEdition() string {
	return kclPkg.ModFile.Pkg.Edition
}

// GetPkgProfile returns the profile of package.
func (kclPkg *KclPkg) GetPkgProfile() *Profile {
	return kclPkg.ModFile.Profiles
}

// GetPkgTarName returns the kcl package tar name "<package_name>-v<package_version>.tar"
func (kclPkg *KclPkg) GetPkgTarName() string {
	return kclPkg.GetPkgFullName() + TAR_SUFFIX
}

const LOCK_FILE_NAME = "kcl.mod.lock"

// GetLockFilePath returns the abs path of kcl.mod.lock.
func (kclPkg *KclPkg) GetLockFilePath() string {
	return filepath.Join(kclPkg.HomePath, LOCK_FILE_NAME)
}

// GetPkgVersion returns the version of package.
func (KclPkg *KclPkg) GetPkgVersion() string {
	return KclPkg.ModFile.Pkg.Version
}

// GetPkgDescription returns the description of package.
func (KclPkg *KclPkg) GetPkgDescription() string {
	return KclPkg.ModFile.Pkg.Description
}

// GetPkgInclude returns the include of package.
func (KclPkg *KclPkg) GetPkgInclude() []string {
	return KclPkg.ModFile.Pkg.Include
}

// GetPkgExclude returns the exclude of package.
func (KclPkg *KclPkg) GetPkgExclude() []string {
	return KclPkg.ModFile.Pkg.Exclude
}

// GenCheckSum generates the checksum of the current kcl package.
func (KclPkg *KclPkg) GenCheckSum() (string, error) {
	return utils.HashDir(KclPkg.HomePath)
}
