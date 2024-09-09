package client

import (
	"fmt"
	"path/filepath"

	"github.com/elliotchance/orderedmap/v2"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/utils"
)

// UpdateOptions is the option for updating a package.
// Updating a package means iterating all the dependencies of the package
// and updating the dependencies and selecting the version of the dependencies by MVS.
type UpdateOptions struct {
	kpkg         *pkg.KclPkg
	enableVendor bool
	vendorPath   string
	offline      bool
}

type UpdateOption func(*UpdateOptions) error

// WithOffline sets the flag to enable the offline mode.
func WithUpdateOffline(offline bool) UpdateOption {
	return func(opts *UpdateOptions) error {
		opts.offline = offline
		return nil
	}
}

// WithEnableVendor sets the flag to enable the vendor.
func WithUpdateEnableVendor(enableVendor bool) UpdateOption {
	return func(opts *UpdateOptions) error {
		opts.enableVendor = enableVendor
		return nil
	}
}

// WithVendorPath sets the path of the vendor.
func WithUpdateVendorPath(vendorPath string) UpdateOption {
	return func(opts *UpdateOptions) error {
		opts.vendorPath = vendorPath
		return nil
	}
}

// WithUpdatedKclPkg sets the kcl package to be updated.
func WithUpdatedKclPkg(kpkg *pkg.KclPkg) UpdateOption {
	return func(opts *UpdateOptions) error {
		opts.kpkg = kpkg
		return nil
	}
}

func (c *KpmClient) Update(options ...UpdateOption) (*pkg.KclPkg, error) {
	opts := &UpdateOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	kpkg := opts.kpkg
	if kpkg == nil {
		return nil, fmt.Errorf("kcl package is nil")
	}

	modDeps := kpkg.ModFile.Dependencies.Deps
	if modDeps == nil {
		return nil, fmt.Errorf("kcl.mod dependencies is nil")
	}
	lockDeps := orderedmap.NewOrderedMap[string, pkg.Dependency]()

	// Create a new dependency resolver
	resolver := NewDepsResolver(c)
	// TODO: Update the local path of the dependency.
	//  After the new local storage structure is complete,
	// this section should be replaced with the new storage structure instead of the cache path according to the <Cache Path>/<Package Name>.
	// https://github.com/kcl-lang/kpm/issues/384
	resolvePathFunc := func(dep *pkg.Dependency, kpkg *pkg.KclPkg) error {
		if opts.enableVendor {
			dep.LocalFullPath = filepath.Join(opts.vendorPath, dep.GenPathSuffix())
		} else {
			dep.LocalFullPath = filepath.Join(c.homePath, dep.GenPathSuffix())
		}

		var err error
		if dep.Source.Git != nil && len(dep.Source.Git.Package) > 0 {
			dep.LocalFullPath, err = utils.FindPackage(dep.LocalFullPath, dep.Source.Git.Package)
			if err != nil {
				return err
			}
		}

		if dep.Source.IsLocalPath() {
			if !filepath.IsAbs(dep.Source.Local.Path) {
				dep.LocalFullPath = filepath.Join(kpkg.HomePath, dep.Source.Local.Path)
			} else {
				dep.LocalFullPath = dep.Source.Local.Path
			}
		}

		return nil
	}

	// ResolveFunc is the function for resolving each dependency when traversing the dependency graph.
	resolverFunc := func(dep *pkg.Dependency, parentPkg *pkg.KclPkg) error {
		// Check if the dependency exists in the mod file.
		if existDep, exist := modDeps.Get(dep.Name); exist {
			// if the dependency exists in the mod file,
			// check the version and select the greater one.
			if less, err := existDep.VersionLessThan(dep); less && err == nil {
				err = resolvePathFunc(dep, parentPkg)
				if err != nil {
					return err
				}
				modDeps.Set(dep.Name, *dep)
			} else {
				err = resolvePathFunc(&existDep, parentPkg)
				if err != nil {
					return err
				}
				modDeps.Set(dep.Name, existDep)
			}
			// if the dependency does not exist in the mod file,
			// the dependency is a indirect dependency.
			// it will be added to the kcl.mod.lock file not the kcl.mod file.
		}
		// Check if the dependency exists in the lock file.
		if existDep, exist := lockDeps.Get(dep.Name); exist {
			// If the dependency exists in the lock file,
			// check the version and select the greater one.
			if less, err := existDep.VersionLessThan(dep); less && err == nil {
				err = resolvePathFunc(dep, parentPkg)
				if err != nil {
					return err
				}
				lockDeps.Set(dep.Name, *dep)
			} else {
				err = resolvePathFunc(&existDep, parentPkg)
				if err != nil {
					return err
				}
				lockDeps.Set(dep.Name, existDep)
			}
		} else {
			// if the dependency does not exist in the lock file,
			// the dependency is a new dependency and will be added to the lock file.
			err := resolvePathFunc(dep, parentPkg)
			if err != nil {
				return err
			}
			lockDeps.Set(dep.Name, *dep)
		}

		return nil
	}

	resolver.AddResolveFunc(resolverFunc)

	err := resolver.Resolve(
		WithEnableCache(true),
		WithCachePath(c.homePath),
		WithResolveKclPkg(kpkg),
		WithEnableVendor(opts.enableVendor),
		WithVendorPath(opts.vendorPath),
	)
	if err != nil {
		return nil, err
	}

	kpkg.Dependencies.Deps = lockDeps
	kpkg.ModFile.Dependencies.Deps = modDeps

	err = kpkg.UpdateModAndLockFile()
	if err != nil {
		return nil, err
	}

	return kpkg, nil
}
