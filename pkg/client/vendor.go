package client

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/hashicorp/go-version"
	"github.com/otiai10/copy"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/features"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/utils"
	"kcl-lang.io/kpm/pkg/visitor"
)

// VendorDeps will vendor all the dependencies of the current kcl package.
func (c *KpmClient) VendorDeps(kclPkg *pkg.KclPkg) error {
	// Mkdir the dir "vendor".
	vendorPath := kclPkg.LocalVendorPath()
	err := os.MkdirAll(vendorPath, 0755)
	if err != nil {
		return err
	}

	return c.vendorDeps(kclPkg, vendorPath)
}

func (c *KpmClient) vendorDeps(kclPkg *pkg.KclPkg, vendorPath string) error {
	if ok, err := features.Enabled(features.SupportMVS); err == nil && ok {
		// Select all the vendored dependencies
		// and fill the vendored dependencies into kclPkg.Dependencies.Deps
		err := c.selectVendoredDeps(kclPkg, vendorPath, kclPkg.Dependencies.Deps)
		if err != nil {
			return err
		}

		// Move all the selected vendored dependencies to the vendor directory.
		for _, depName := range kclPkg.Dependencies.Deps.Keys() {
			dep, ok := kclPkg.Dependencies.Deps.Get(depName)
			if !ok {
				return fmt.Errorf("failed to get dependency %s", depName)
			}

			// Check if the dependency is already vendored in the vendor directory.
			existLocalDep, err := c.dependencyExistsLocal(filepath.Dir(vendorPath), &dep, true)
			if err != nil {
				return err
			}

			if existLocalDep == nil {
				vendorFullPath := filepath.Join(vendorPath, dep.GenDepFullName())
				cacheFullPath := filepath.Join(c.homePath, dep.GenDepFullName())
				if !utils.DirExists(vendorFullPath) {
					err := copy.Copy(cacheFullPath, vendorFullPath)
					if err != nil {
						return err
					}
				}
				// Load the vendored dependency
				existLocalDep, err = c.dependencyExistsLocal(filepath.Dir(vendorPath), &dep, true)
				if err != nil {
					return err
				}

				if existLocalDep == nil {
					return fmt.Errorf("failed to find the vendored dependency %s", depName)
				}
			}
			kclPkg.Dependencies.Deps.Set(depName, *existLocalDep)
		}
	} else {
		lockDeps := make([]pkg.Dependency, 0, kclPkg.Dependencies.Deps.Len())
		for _, k := range kclPkg.Dependencies.Deps.Keys() {
			d, _ := kclPkg.Dependencies.Deps.Get(k)
			lockDeps = append(lockDeps, d)
		}

		// Traverse all dependencies in kcl.mod.lock.
		for i := 0; i < len(lockDeps); i++ {
			d := lockDeps[i]
			if len(d.Name) == 0 {
				return errors.InvalidDependency
			}
			// If the dependency is from the local path, do not vendor it, vendor its dependencies.
			if d.IsFromLocal() {
				dpkg, err := c.LoadPkgFromPath(d.GetLocalFullPath(kclPkg.HomePath))
				if err != nil {
					return err
				}
				err = c.vendorDeps(dpkg, vendorPath)
				if err != nil {
					return err
				}
				continue
			} else {
				vendorFullPath := filepath.Join(vendorPath, d.GenPathSuffix())

				// If the package already exists in the 'vendor', do nothing.
				if utils.DirExists(vendorFullPath) {
					d.LocalFullPath = vendorFullPath
					lockDeps[i] = d
					continue
				} else {
					// If not in the 'vendor', check the global cache.
					cacheFullPath := c.getDepStorePath(c.homePath, &d, false)
					if utils.DirExists(cacheFullPath) {
						// If there is, copy it into the 'vendor' directory.
						err := copy.Copy(cacheFullPath, vendorFullPath)
						if err != nil {
							return err
						}
					} else {
						// re-download if not.
						err := c.AddDepToPkg(kclPkg, &d)
						if err != nil {
							return err
						}
						// re-vendor again with new kcl.mod and kcl.mod.lock
						err = c.vendorDeps(kclPkg, vendorPath)
						if err != nil {
							return err
						}
						return nil
					}
				}

				if d.GetPackage() != "" {
					tempVendorFullPath, err := utils.FindPackage(vendorFullPath, d.GetPackage())
					if err != nil {
						return err
					}
					vendorFullPath = tempVendorFullPath
				}

				dpkg, err := c.LoadPkgFromPath(vendorFullPath)
				if err != nil {
					return err
				}

				// Vendor the dependencies of the current dependency.
				err = c.vendorDeps(dpkg, vendorPath)
				if err != nil {
					return err
				}
				d.LocalFullPath = vendorFullPath
				lockDeps[i] = d
			}
		}

		// Update the dependencies in kcl.mod.lock.
		for _, d := range lockDeps {
			kclPkg.Dependencies.Deps.Set(d.Name, d)
		}

		return nil
	}
	return nil
}

func (c *KpmClient) selectVendoredDeps(kpkg *pkg.KclPkg, vendorPath string, vendoredDeps *orderedmap.OrderedMap[string, pkg.Dependency]) error {
	// visitorSelectorFunc selects the visitor for the source.
	// For remote source, it will use the RemoteVisitor and enable the cache.
	// For local source, it will use the PkgVisitor.
	visitorSelectorFunc := func(source *downloader.Source) (visitor.Visitor, error) {
		pkgVisitor := &visitor.PkgVisitor{
			Settings:  &c.settings,
			LogWriter: c.logWriter,
		}

		if source.IsRemote() {
			return &visitor.RemoteVisitor{
				PkgVisitor:            pkgVisitor,
				Downloader:            c.DepDownloader,
				InsecureSkipTLSverify: c.insecureSkipTLSverify,
				EnableCache:           true,
				CachePath:             c.homePath,
			}, nil
		} else if source.IsLocalTarPath() || source.IsLocalTgzPath() {
			return visitor.NewArchiveVisitor(pkgVisitor), nil
		} else if source.IsLocalPath() {
			rootPath, err := source.FindRootPath()
			if err != nil {
				return nil, err
			}
			kclmodpath := filepath.Join(rootPath, constants.KCL_MOD)
			if utils.DirExists(kclmodpath) {
				return pkgVisitor, nil
			} else {
				return visitor.NewVirtualPkgVisitor(pkgVisitor), nil
			}
		} else {
			return nil, fmt.Errorf("unsupported source")
		}
	}

	// Iterate all the dependencies of the package in kcl.mod.
	for _, depName := range kpkg.ModFile.Dependencies.Deps.Keys() {
		dep, ok := kpkg.ModFile.Dependencies.Deps.Get(depName)
		if !ok {
			return fmt.Errorf("failed to get dependency %s", depName)
		}

		// Select the dependency with the MVS
		// Keep the greater version in dependencies graph
		selectedDep := &dep
		if existsDep, exists := vendoredDeps.Get(depName); exists && len(existsDep.Version) > 0 &&
			len(dep.Version) > 0 &&
			// TODO: Skip the git dependencies for now and get the version from the cache when the new local storage structure is complete
			// the new local storage structure: https://github.com/kcl-lang/kpm/issues/384
			dep.Source.Git == nil &&
			existsDep.Source.Git == nil {
			existsVersion, err := version.NewVersion(existsDep.Version)
			if err != nil {
				return err
			}
			depVersion, err := version.NewVersion(dep.Version)
			if err != nil {
				return err
			}
			// Select the greater version
			if existsVersion.GreaterThan(depVersion) {
				selectedDep = &existsDep
			}
		}

		// Check if the dependency is already vendored in the vendor directory.
		existLocalDep, err := c.dependencyExistsLocal(filepath.Dir(vendorPath), selectedDep, true)
		if err != nil {
			return err
		}

		// If the dependency is already vendored, just update the dependency path.
		if existLocalDep != nil {
			// Collect the vendored dependency
			vendoredDeps.Set(depName, *existLocalDep)
			// Load the vendored dependency
			dpkg, err := pkg.LoadKclPkgWithOpts(
				pkg.WithPath(existLocalDep.LocalFullPath),
				pkg.WithSettings(&c.settings),
			)
			if err != nil {
				return err
			}
			// Vendor the indirected dependencies of the vendored dependency
			err = c.selectVendoredDeps(dpkg, vendorPath, vendoredDeps)
			if err != nil {
				return err
			}
		} else {
			// If the dependency is not vendored in the vendor directory
			selectDepSource := &selectedDep.Source
			// Check if the dependency is a local path and it is not an absolute path.
			// If it is not an absolute path, transform the path to an absolute path.
			if selectDepSource.IsLocalPath() && !filepath.IsAbs(selectDepSource.Local.Path) {
				selectDepSource = &downloader.Source{
					Local: &downloader.Local{
						Path: filepath.Join(kpkg.HomePath, selectDepSource.Local.Path),
					},
				}
			}

			// By visitor, if the dependency is a remote source, it will download and load the dependency
			// if the dependency is a local source, it will load the dependency.
			// if the dependency is cached, it will load the dependency from the cache.
			pkgVisitor, err := visitorSelectorFunc(selectDepSource)
			if err != nil {
				return err
			}
			err = pkgVisitor.Visit(selectDepSource,
				func(kclPkg *pkg.KclPkg) error {
					existLocalDep, err := c.dependencyExistsLocal(c.homePath, selectedDep, false)
					if err != nil {
						return err
					}

					if existLocalDep == nil {
						return fmt.Errorf("failed to find the vendored dependency %s", depName)
					}
					// Collect the vendored dependency
					vendoredDeps.Set(depName, *existLocalDep)
					return nil
				},
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
