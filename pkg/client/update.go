package client

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"kcl-lang.io/kpm/pkg/checker"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/features"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/resolver"
	"kcl-lang.io/kpm/pkg/utils"
	"oras.land/oras-go/v2"
)

// UpdateOptions is the option for updating a package.
// Updating a package means iterating all the dependencies of the package
// and updating the dependencies and selecting the version of the dependencies by MVS.
type UpdateOptions struct {
	kpkg          *pkg.KclPkg
	offline       bool
	updateModFile bool
}

type UpdateOption func(*UpdateOptions) error

// WithUpdateModFile sets the flag to update the mod file.
func WithUpdateModFile(updateModFile bool) UpdateOption {
	return func(opts *UpdateOptions) error {
		opts.updateModFile = updateModFile
		return nil
	}
}

// WithOffline sets the offline option to update the package.
func WithOffline(offline bool) UpdateOption {
	return func(opts *UpdateOptions) error {
		opts.offline = offline
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
	opts := &UpdateOptions{updateModFile: true}
	for _, option := range options {
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	kMod := opts.kpkg
	if kMod == nil {
		return nil, fmt.Errorf("kcl package is nil")
	}

	if ok, err := features.Enabled(features.SupportModCheck); err == nil && ok && c.noSumCheck {
		c.ModChecker = checker.NewModChecker(
			checker.WithCheckers(
				checker.NewIdentChecker(),
				checker.NewVersionChecker(),
				checker.NewSumChecker(),
			),
		)

		err := c.Check(
			WithCheckKclMod(kMod),
		)
		if err != nil {
			return nil, err
		}
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
			if ok, err := features.Enabled(features.SupportMVS); err == nil && ok {
				// if the dependency exists in the mod file,
				// check the version and select the greater one.
				if less, err := dep.VersionLessThan(&existDep); less && err == nil {
					selectedModDep = &existDep
				}
			}
			// if the dependency does not exist in the mod file,
			// the dependency is a indirect dependency.
			// it will be added to the kcl.mod.lock file not the kcl.mod file.
			kMod.ModFile.Dependencies.Deps.Set(dep.Name, *selectedModDep)
		}

		selectedDep := dep
		// Check if the dependency exists in the lock file.
		if existDep, exist := lockDeps.Get(dep.Name); exist {
			if ok, err := features.Enabled(features.SupportMVS); err == nil && ok {
				// If the dependency exists in the lock file,
				// check the version and select the greater one.
				if less, err := dep.VersionLessThan(&existDep); less && err == nil {
					selectedDep = &existDep
				}
			}
		}

		// Check if the checksum of the dependency exists in the lock file.
		if existDep, exist := lockDeps.Get(dep.Name); exist {
			if equal, err := existDep.VersionEqual(selectedDep); equal && err == nil {
				selectedDep.Sum = existDep.Sum
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
		resolver.WithOffline(opts.offline),
	)

	if err != nil {
		return nil, err
	}

	if opts.updateModFile && utils.DirExists(filepath.Join(kMod.HomePath, constants.KCL_MOD)) {
		err = kMod.UpdateModFile()
		if err != nil {
			return nil, err
		}
	}

	// Generate file kcl.mod.lock.
	if !kMod.NoSumCheck && utils.DirExists(filepath.Join(kMod.HomePath, constants.KCL_MOD)) {
		err := kMod.LockDepsVersion()
		if err != nil {
			return nil, err
		}
	}

	return kMod, nil
}

// AcquireDepSum will acquire the checksum of the dependency from the OCI registry.
func (c *KpmClient) AcquireDepSum(dep pkg.Dependency) (string, error) {
	// Only the dependencies from the OCI need can be checked.
	if dep.Source.Oci != nil {
		if len(dep.Source.Oci.Reg) == 0 {
			dep.Source.Oci.Reg = c.GetSettings().DefaultOciRegistry()
		}

		if len(dep.Source.Oci.Repo) == 0 {
			urlpath := utils.JoinPath(c.GetSettings().DefaultOciRepo(), dep.Name)
			dep.Source.Oci.Repo = urlpath
		}
		// Fetch the metadata of the OCI manifest.
		manifest := ocispec.Manifest{}
		jsonDesc, err := c.FetchOciManifestIntoJsonStr(opt.OciFetchOptions{
			FetchBytesOptions: oras.DefaultFetchBytesOptions,
			OciOptions: opt.OciOptions{
				Reg:  dep.Source.Oci.Reg,
				Repo: dep.Source.Oci.Repo,
				Tag:  dep.Source.Oci.Tag,
			},
		})

		if err != nil {
			return "", reporter.NewErrorEvent(reporter.FailedFetchOciManifest, err, fmt.Sprintf("failed to fetch the manifest of '%s'", dep.Name))
		}

		err = json.Unmarshal([]byte(jsonDesc), &manifest)
		if err != nil {
			return "", err
		}

		// Check the dependency checksum.
		if value, ok := manifest.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_SUM]; ok {
			return value, nil
		}
	}

	return "", nil
}
