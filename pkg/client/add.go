package client

import (
	"errors"
	"fmt"
	"path/filepath"

	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// The AddOptions struct contains the options for adding a package to the dependencies of a kcl package.
type AddOptions struct {
	// NewPkgName is the new package name to be added to the dependencies.
	NewPkgName string
	// Source is the source of the package to be added.
	// Including git, oci, local.
	Source *downloader.Source
	// KclPkg is the kcl package to be added dependencies to.
	KclPkg *pkg.KclPkg
}

type AddOption func(*AddOptions) error

func WithNewPkgName(name string) AddOption {
	return func(opts *AddOptions) error {
		if name == "" {
			return errors.New("package name cannot be empty")
		}
		opts.NewPkgName = name
		return nil
	}
}

func WithAddSource(source *downloader.Source) AddOption {
	return func(opts *AddOptions) error {
		if source == nil {
			return errors.New("source cannot be nil")
		}
		opts.Source = source
		return nil
	}
}

func WithKclPkg(kpkg *pkg.KclPkg) AddOption {
	return func(opts *AddOptions) error {
		if kpkg == nil {
			return errors.New("kpkg cannot be nil")
		}
		opts.KclPkg = kpkg
		return nil
	}
}

func (c *KpmClient) Add(options ...AddOption) (*pkg.KclPkg, error) {
	opts := &AddOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	// acquire the lock of the package cache.
	err := c.AcquirePackageCacheLock()
	if err != nil {
		return nil, err
	}

	defer func() {
		// release the lock of the package cache after the function returns.
		releaseErr := c.ReleasePackageCacheLock()
		if releaseErr != nil && err == nil {
			err = releaseErr
		}
	}()

	// 1. Check if the package is already in local cache.
	searchRoot := c.homePath
	sourceLocalpath, err := opts.Source.ToFilePath()
	if err != nil {
		return nil, err
	}

	var depSearchPath string
	if opts.Source.IsLocalPath() {
		depSearchPath = sourceLocalpath
	} else {
		depSearchPath = filepath.Join(searchRoot, sourceLocalpath)
	}

	var dep pkg.Dependency
	// 2. If not exist, redownload the package.
	if !utils.DirExists(depSearchPath) {
		_, err = c.downloadPkg(
			downloader.WithLocalPath(depSearchPath),
			downloader.WithSource(*opts.Source),
		)
		if err != nil {
			return nil, err
		}
	}

	// 3. Load the dependency package from the local path.
	dpkg, err := c.LoadPkgFromPath(depSearchPath)
	if err != nil {
		return nil, err
	}
	dep.FromKclPkg(dpkg)
	dep.Source = *opts.Source
	depStr, err := dep.ToString()
	if err != nil {
		return nil, err
	}
	reporter.ReportMsgTo(
		fmt.Sprintf("adding %s to dependencies", depStr),
		c.logWriter,
	)

	// 3. Add it to the dependencies of the kcl package.
	err = c.AddDepToPkg(opts.KclPkg, &dep)
	if err != nil {
		return nil, err
	}

	// 4. Rename the package name if NewPkgName is not empty.
	if opts.NewPkgName != "" {
		// update the kcl.mod with NewPkgName
		tempDeps := opts.KclPkg.ModFile.Dependencies.Deps[dep.Name]
		tempDeps.Name = opts.NewPkgName
		opts.KclPkg.ModFile.Dependencies.Deps[dep.Name] = tempDeps

		// update the kcl.mod.lock with NewPkgName
		tempDeps = opts.KclPkg.Dependencies.Deps[dep.Name]
		tempDeps.Name = opts.NewPkgName
		tempDeps.FullName = opts.NewPkgName + "_" + tempDeps.Version
		opts.KclPkg.Dependencies.Deps[dep.Name] = tempDeps

		// update the key of kclPkg.Dependencies.Deps from d.Name to opt.NewPkgName
		opts.KclPkg.Dependencies.Deps[opts.NewPkgName] = opts.KclPkg.Dependencies.Deps[dep.Name]
		delete(opts.KclPkg.Dependencies.Deps, dep.Name)
	}

	// 5. Store the kcl.mod and kcl.mod.lock file.
	err = opts.KclPkg.UpdateModAndLockFile()
	if err != nil {
		return nil, err
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("add %s successfully", depStr),
		c.logWriter,
	)

	return opts.KclPkg, nil
}
