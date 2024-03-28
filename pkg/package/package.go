package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/module"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/constants"
	errors "kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

type KclPkg struct {
	ModFile  ModFile
	HomePath string
	// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
	// not the dependencies in kcl.mod.
	Dependencies
	// minimal build list for the current kcl package.
	BuildList []module.Version
	// The flag 'NoSumCheck' is true if the checksum of the current kcl package is not checked.
	NoSumCheck bool
}

func (p *KclPkg) GetDepsMetadata() (*Dependencies, error) {
	return p.Dependencies.ToDepMetadata()
}

func NewKclPkg(opts *opt.InitOptions) KclPkg {
	return KclPkg{
		ModFile:      *NewModFile(opts),
		HomePath:     opts.InitPath,
		Dependencies: Dependencies{Deps: make(map[string]Dependency)},
	}
}

func LoadKclPkg(pkgPath string) (*KclPkg, error) {
	modFile, err := LoadModFile(pkgPath)
	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.FailedLoadKclMod, err, fmt.Sprintf("could not load 'kcl.mod' in '%s'", pkgPath))
	}

	// Get dependencies from kcl.mod.lock.
	deps, err := LoadLockDeps(pkgPath)

	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.FailedLoadKclMod, err, fmt.Sprintf("could not load 'kcl.mod.lock' in '%s'", pkgPath))
	}

	return &KclPkg{
		ModFile:      *modFile,
		HomePath:     pkgPath,
		Dependencies: *deps,
	}, nil
}

func FindFirstKclPkgFrom(path string) (*KclPkg, error) {
	matches, _ := filepath.Glob(filepath.Join(path, "*.tar"))
	if matches == nil || len(matches) != 1 {
		// then try to glob tgz file
		matches, _ = filepath.Glob(filepath.Join(path, "*.tgz"))
		if matches == nil || len(matches) != 1 {
			pkg, err := LoadKclPkg(path)
			if err != nil {
				return nil, reporter.NewErrorEvent(
					reporter.InvalidKclPkg,
					err,
					fmt.Sprintf("failed to find the kcl package tar from '%s'.", path),
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
			fmt.Sprintf("failed to find the kcl package tar from '%s'.", path),
		)
	}

	return pkg, nil
}

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

// updateModAndLockFile will update kcl.mod and kcl.mod.lock
func (kclPkg *KclPkg) UpdateModAndLockFile() error {
	// Generate file kcl.mod.
	err := kclPkg.ModFile.StoreModFile()
	if err != nil {
		return err
	}

	// Generate file kcl.mod.lock.
	if !kclPkg.NoSumCheck {
		err = kclPkg.LockDepsVersion()
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

// CreateDefauleMain will create a default main.k file in the current kcl package.
func (kclPkg *KclPkg) CreateDefauleMain() error {
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

// GenCheckSum generates the checksum of the current kcl package.
func (KclPkg *KclPkg) GenCheckSum() (string, error) {
	return utils.HashDir(KclPkg.HomePath)
}
