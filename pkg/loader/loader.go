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

// loadModFile loads the kcl.mod file from the package path.
func loadModFile(pkgPath string) (*pkg.ModFile, error) {
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
func preProcess(kpkg *pkg.KclPkg, settings *settings.Settings) error {
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
		if dep.Source.Registry != nil {
			if len(dep.Source.Registry.Reg) == 0 {
				dep.Source.Registry.Reg = settings.DefaultOciRegistry()
			}

			if len(dep.Source.Registry.Repo) == 0 {
				urlpath := utils.JoinPath(settings.DefaultOciRepo(), dep.Name)
				dep.Source.Registry.Repo = urlpath
			}

			dep.Version = dep.Source.Registry.Version
		}
		if dep.Source.IsNilSource() {
			dep.Source.Registry = &downloader.Registry{
				Name:    dep.Name,
				Version: dep.Version,
				Oci: &downloader.Oci{
					Reg:  settings.DefaultOciRegistry(),
					Repo: utils.JoinPath(settings.DefaultOciRepo(), dep.Name),
					Tag:  dep.Version,
				},
			}
		}
		kpkg.ModFile.Dependencies.Deps.Set(name, dep)
		if lockDep, ok := kpkg.Dependencies.Deps.Get(name); ok {
			lockDep.Source = dep.Source
			lockDep.LocalFullPath = dep.LocalFullPath
			kpkg.Dependencies.Deps.Set(name, lockDep)
		}
	}

	return nil
}

type LoadOptions struct {
	// The package path.
	PkgPath string
	// The settings with default oci registry.
	Settings *settings.Settings
}

type Option func(*LoadOptions)

// WithPkgPath sets the package path.
func WithPkgPath(pkgPath string) Option {
	return func(opts *LoadOptions) {
		opts.PkgPath = pkgPath
	}
}

// WithSettings sets the settings with default oci registry.
func WithSettings(settings *settings.Settings) Option {
	return func(opts *LoadOptions) {
		opts.Settings = settings
	}
}

// Load loads a package from the file system.
func Load(options ...Option) (*pkg.KclPkg, error) {
	opts := &LoadOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pkgPath := opts.PkgPath

	modFile, err := loadModFile(pkgPath)
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
	err = preProcess(kpkg, opts.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to load the package from the path %s: %w", pkgPath, err)
	}

	return kpkg, nil
}
