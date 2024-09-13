package loader

import (
	"fmt"
	"path/filepath"

	"github.com/elliotchance/orderedmap/v2"
	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

// Loader is an interface that defines the method to load a package.
type Loader interface {
	Load(pkgPath string) (*pkg.KclPkg, error)
}

// PkgLoader is a struct that contains the settings.
type PkgLoader struct {
	settings *settings.Settings
}

// NewPkgLoader creates a new PkgLoader.
func NewPkgLoader(settings *settings.Settings) *PkgLoader {
	return &PkgLoader{
		settings: settings,
	}
}

// FileLoader is a struct that load a package from the file system.
type FileLoader struct {
	PkgLoader
}

// NewFileLoader creates a new FileLoader.
func NewFileLoader(settings *settings.Settings) *FileLoader {
	return &FileLoader{
		PkgLoader: PkgLoader{
			settings: settings,
		},
	}
}

// loadModFile loads the kcl.mod file from the package path.
func (fl *FileLoader) loadModFile(pkgPath string) (*pkg.ModFile, error) {
	modFile := new(pkg.ModFile)
	err := modFile.LoadModFile(filepath.Join(pkgPath, pkg.MOD_FILE))
	if err != nil {
		return nil, fmt.Errorf("failed to load the mod file: %w", err)
	}
	modFile.HomePath = pkgPath
	if modFile.Dependencies.Deps == nil {
		modFile.Dependencies.Deps = orderedmap.NewOrderedMap[string, pkg.Dependency]()
	}
	return modFile, nil
}

// preProcess pre-processes the package loaded from kcl.mod and kcl.mod.lock
// 1. transform the local path to the absolute path.
// 2. fill the default oci registry.
func (fl *FileLoader) preProcess(kpkg *pkg.KclPkg) error {
	for _, name := range kpkg.ModFile.Dependencies.Deps.Keys() {
		dep, ok := kpkg.ModFile.Dependencies.Deps.Get(name)
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
				localFullPath, err = filepath.Abs(filepath.Join(kpkg.HomePath, dep.Local.Path))
				if err != nil {
					return fmt.Errorf("failed to get the absolute path of the local dependency %s: %w", name, err)
				}
			}
			dep.LocalFullPath = localFullPath
			kpkg.ModFile.Dependencies.Deps.Set(name, dep)
		}
		// Fill the default oci registry.
		if dep.Source.Oci != nil {
			if len(dep.Source.Oci.Reg) == 0 {
				dep.Source.Oci.Reg = fl.settings.DefaultOciRegistry()
			}

			if len(dep.Source.Oci.Repo) == 0 {
				urlpath := utils.JoinPath(fl.settings.DefaultOciRepo(), dep.Name)
				dep.Source.Oci.Repo = urlpath
			}
		}
		if dep.Source.Registry != nil {
			if len(dep.Source.Registry.Reg) == 0 {
				dep.Source.Registry.Reg = fl.settings.DefaultOciRegistry()
			}

			if len(dep.Source.Registry.Repo) == 0 {
				urlpath := utils.JoinPath(fl.settings.DefaultOciRepo(), dep.Name)
				dep.Source.Registry.Repo = urlpath
			}

			dep.Version = dep.Source.Registry.Version
		}
		if dep.Source.IsNilSource() {
			dep.Source.Registry = &downloader.Registry{
				Name:    dep.Name,
				Version: dep.Version,
				Oci: &downloader.Oci{
					Reg:  fl.settings.DefaultOciRegistry(),
					Repo: utils.JoinPath(fl.settings.DefaultOciRepo(), dep.Name),
					Tag:  dep.Version,
				},
			}
		}
		kpkg.ModFile.Dependencies.Deps.Set(name, dep)
		if lockDep, ok := kpkg.Dependencies.Deps.Get(name); ok {
			lockDep.Source = dep.Source
			kpkg.Dependencies.Deps.Set(name, lockDep)
		}
	}

	return nil
}

// Load loads a package from the file system.
func (fl *FileLoader) Load(pkgPath string) (*pkg.KclPkg, error) {
	modFile, err := fl.loadModFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load the package from the path %s: %w", pkgPath, err)
	}

	// load the kcl.mod.lock file.
	// Get dependencies from kcl.mod.lock.
	deps, err := pkg.LoadLockDeps(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load the package from the path %s: %w", pkgPath, err)
	}

	kpkg := &pkg.KclPkg{
		ModFile:      *modFile,
		HomePath:     pkgPath,
		Dependencies: *deps,
	}

	// pre-process the package.
	err = fl.preProcess(kpkg)
	if err != nil {
		return nil, fmt.Errorf("failed to load the package from the path %s: %w", pkgPath, err)
	}

	return kpkg, nil
}
