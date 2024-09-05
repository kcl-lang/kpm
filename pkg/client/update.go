package client

import (
	"fmt"
	"path/filepath"

	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
)

// UpdateOptions is the option for updating a package.
// Updating a package means iterating all the dependencies of the package
// and updating the dependencies and selecting the version of the dependencies by MVS.
type UpdateOptions struct {
	kpkg *pkg.KclPkg
}

type UpdateOption func(*UpdateOptions) error

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

	modDeps := kpkg.ModFile.Dependencies.Deps
	lockDeps := kpkg.Dependencies.Deps

	// Create a new dependency resolver
	resolver := NewDepsResolver(c)
	// ResolveFunc is the function for resolving each dependency when traversing the dependency graph.
	resolverFunc := func(dep *pkg.Dependency, parentPkg *pkg.KclPkg) error {
		// Check if the dependency exists in the mod file.
		if existDep, exist := modDeps.Get(dep.Name); exist {
			// if the dependency exists in the mod file,
			// check the version and select the greater one.
			if less, err := existDep.VersionLessThan(dep); less && err == nil {
				kpkg.ModFile.Dependencies.Deps.Set(dep.Name, *dep)
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
				kpkg.Dependencies.Deps.Set(dep.Name, *dep)
			}
		} else {
			// if the dependency does not exist in the lock file,
			// the dependency is a new dependency and will be added to the lock file.
			kpkg.Dependencies.Deps.Set(dep.Name, *dep)
		}

		return nil
	}
	resolver.AddResolveFunc(resolverFunc)

	// Iterate all the dependencies of the package in kcl.mod and resolve each dependency.
	for _, depName := range modDeps.Keys() {
		dep, ok := modDeps.Get(depName)
		if !ok {
			return nil, fmt.Errorf("failed to get dependency %s", depName)
		}

		// Check if the dependency is a local path and it is not an absolute path.
		// If it is not an absolute path, transform the path to an absolute path.
		var depSource *downloader.Source
		if dep.Source.IsLocalPath() && !filepath.IsAbs(dep.Source.Local.Path) {
			depSource = &downloader.Source{
				Local: &downloader.Local{
					Path: filepath.Join(kpkg.HomePath, dep.Source.Local.Path),
				},
			}
		} else {
			depSource = &dep.Source
		}

		err := resolver.Resolve(
			WithEnableCache(true),
			WithResolveSource(depSource),
		)
		if err != nil {
			return nil, err
		}
	}

	err := kpkg.UpdateModAndLockFile()
	if err != nil {
		return nil, err
	}

	return kpkg, nil
}
