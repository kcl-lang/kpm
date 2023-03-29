package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	modfile "kusionstack.io/kpm/pkg/mod"
	"kusionstack.io/kpm/pkg/opt"
	"kusionstack.io/kpm/pkg/reporter"
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

// InitEmptyModule inits an empty kcl module and create a default kcl.modfile.
func (kclPkg KclPkg) InitEmptyPkg() error {
	err := createFileIfNotExist(kclPkg.modFile.GetModFilePath(), "kcl.mod", kclPkg.modFile.Store)
	if err != nil {
		return err
	}

	err = createFileIfNotExist(kclPkg.modFile.GetModLockFilePath(), "kcl.mod.lock", kclPkg.LockDepsVersion)
	if err != nil {
		return err
	}

	return nil
}

func createFileIfNotExist(filePath string, fileName string, storeFunc func() error) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		reporter.Report("kpm: creating new "+fileName+":", filePath)
		err := storeFunc()
		if err != nil {
			reporter.Report("kpm: failed to create "+fileName+",", err)
			return err
		}
	} else {
		reporter.Report("kpm: '" + filePath + "' already exists")
		return err
	}
	return nil
}

// InitEmptyModule inits an empty kcl module and create a default kcl.modfile.
func (kclPkg KclPkg) AddDeps(opt *opt.AddOptions) error {

	d := modfile.ParseOpt(&opt.RegistryOpts)

	if !reflect.DeepEqual(kclPkg.modFile.Dependencies.Deps[d.Name], *d) {
		// the dep passed on the cli is different from the jsonnetFile
		kclPkg.modFile.Dependencies.Deps[d.Name] = *d
		// we want to install the passed version (ignore the lock)
		delete(kclPkg.Dependencies.Deps, d.Name)
	}

	changedDeps, err := getDeps(kclPkg.modFile.Dependencies, kclPkg.Dependencies, opt.LocalPath)

	if err != nil {
		reporter.ExitWithReport("kpm: failed to download dependancies.")
	}

	for k, v := range changedDeps.Deps {
		kclPkg.modFile.Dependencies.Deps[k] = v
		kclPkg.Dependencies.Deps[k] = v
	}

	err = kclPkg.modFile.Store()
	if err != nil {
		return err
	}

	err = kclPkg.LockDepsVersion()
	if err != nil {
		return err
	}

	return nil
}

// LockDepsVersion locks the dependencies of the current kcl package into kcl.mod.lock.
func (kclPkg KclPkg) LockDepsVersion() error {
	fullPath := filepath.Join(kclPkg.HomePath, modfile.MOD_LOCK_FILE)
	return modfile.StoreToFile(fullPath, kclPkg.Dependencies)
}

func getDeps(deps modfile.Dependencies, lockDeps modfile.Dependencies, localPath string) (*modfile.Dependencies, error) {
	newDeps := modfile.Dependencies{
		Deps: make(map[string]modfile.Dependency),
	}

	for _, d := range deps.Deps {
		if len(d.Name) == 0 {
			reporter.ExitWithReport("kpm: invalid dependencies.")
			return nil, fmt.Errorf("kpm: invalid dependencies.")
		}
		lockDep, present := lockDeps.Deps[d.Name]

		// already locked and the integrity is intact
		if present {
			d.Version = lockDeps.Deps[d.Name].Version

			if check(lockDep, localPath) {
				newDeps.Deps[d.Name] = lockDep
				continue
			}
		}
		expectedSum := lockDeps.Deps[d.Name].Sum

		dir := filepath.Join(localPath, d.Name)
		os.RemoveAll(dir)

		lockedDep, err := d.Download(dir)
		if err != nil {
			return nil, fmt.Errorf("checksum mismatch")
		}
		if expectedSum != "" && lockedDep.Sum != expectedSum {
			return nil, fmt.Errorf("checksum mismatch")
		}
		newDeps.Deps[d.Name] = *lockedDep
		lockDeps.Deps[d.Name] = *lockedDep
	}

	for _, d := range newDeps.Deps {
		modfile, err := modfile.LoadModFile(filepath.Join(localPath, d.Name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		nested, err := getDeps(modfile.Dependencies, lockDeps, localPath)
		if err != nil {
			return nil, err
		}

		for _, d := range nested.Deps {
			if _, ok := newDeps.Deps[d.Name]; !ok {
				newDeps.Deps[d.Name] = d
			}
		}
	}

	return &newDeps, nil
}

// check sum for a Dependency.
func check(dep modfile.Dependency, vendorDir string) bool {
	if dep.Sum == "" {
		return false
	}

	dir := filepath.Join(vendorDir, dep.Name)
	sum := utils.HashDir(dir)
	return dep.Sum == sum
}
