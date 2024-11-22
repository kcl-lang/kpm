package client

import (
	"fmt"

	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/resolver"
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

	kMod := opts.kpkg
	if kMod == nil {
		return nil, fmt.Errorf("kcl package is nil")
	}

	kMod.NoSumCheck = c.noSumCheck

	modDeps := kMod.ModFile.Dependencies.Deps
	if modDeps == nil {
		return nil, fmt.Errorf("kcl.mod dependencies is nil")
	}
	lockDeps := kMod.Dependencies.Deps
	if lockDeps == nil {
		return nil, fmt.Errorf("kcl.mod.lock dependencies is nil")
	}

	// Create a new dependency resolver
	depResolver := resolver.DepsResolver{
		DefaultCachePath:      c.homePath,
		InsecureSkipTLSverify: c.insecureSkipTLSverify,
		Downloader:            c.DepDownloader,
		Settings:              &c.settings,
		LogWriter:             c.logWriter,
	}
	// ResolveFunc is the function for resolving each dependency when traversing the dependency graph.
	resolverFunc := func(dep *pkg.Dependency, parentPkg *pkg.KclPkg) error {
		selectedModDep := dep
		// Check if the dependency exists in the mod file.
		if existDep, exist := modDeps.Get(dep.Name); exist {
			// if the dependency exists in the mod file,
			// check the version and select the greater one.
			if less, err := dep.VersionLessThan(&existDep); less && err == nil {
				selectedModDep = &existDep
			}
			// if the dependency does not exist in the mod file,
			// the dependency is a indirect dependency.
			// it will be added to the kcl.mod.lock file not the kcl.mod file.
			kMod.ModFile.Dependencies.Deps.Set(dep.Name, *selectedModDep)
		}

		selectedDep := dep
		// Check if the dependency exists in the lock file.
		if existDep, exist := lockDeps.Get(dep.Name); exist {
			// If the dependency exists in the lock file,
			// check the version and select the greater one.
			if less, err := dep.VersionLessThan(&existDep); less && err == nil {
				selectedDep = &existDep
			}
		}
		selectedDep.LocalFullPath = dep.LocalFullPath
		if selectedDep.Sum == "" {
			sum, err := c.AcquireDepSum(*selectedDep)
			if err != nil {
				return err
			}
			if sum != "" {
				selectedDep.Sum = sum
			}
		}
		kMod.Dependencies.Deps.Set(dep.Name, *selectedDep)

		return nil
	}
	depResolver.ResolveFuncs = append(depResolver.ResolveFuncs, resolverFunc)

	err := depResolver.Resolve(
		resolver.WithResolveKclMod(kMod),
		resolver.WithEnableCache(true),
		resolver.WithCachePath(c.homePath),
	)

	if err != nil {
		return nil, err
	}

	err = kMod.UpdateModAndLockFile()
	if err != nil {
		return nil, err
	}

	return kMod, nil
}
