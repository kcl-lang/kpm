package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/otiai10/copy"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/utils"
)

// ResolveOption is the option for resolving dependencies.
type ResolveOption func(*ResolveOptions) error

// resolveFunc is the function for resolving each dependency when traversing the dependency graph.
// currentPkg is the current package to be resolved and parentPkg is the parent package of the current package.
type resolveFunc func(dep *pkg.Dependency, parentPkg *pkg.KclPkg) error

type ResolveOptions struct {
	// Source is the source of the package to be resolved.
	kpkg *pkg.KclPkg
	// EnableCache is the flag to enable the cache during the resolving the remote package.
	EnableCache bool
	// EnableVendor is the flag to enable the vendor.
	EnableVendor bool
	// CachePath is the path of the cache.
	CachePath string
	// vendorPath is the path of the vendor.
	VendorPath string
	// Offline is the flag to enable the offline mode.
	Offline bool
}

// WithOffline sets the flag to enable the offline mode.
func WithOffline(offline bool) ResolveOption {
	return func(opts *ResolveOptions) error {
		opts.Offline = offline
		return nil
	}
}

// WithEnableVendor sets the flag to enable the vendor.
func WithEnableVendor(enableVendor bool) ResolveOption {
	return func(opts *ResolveOptions) error {
		opts.EnableVendor = enableVendor
		return nil
	}
}

// WithVendorPath sets the path of the vendor.
func WithVendorPath(vendorPath string) ResolveOption {
	return func(opts *ResolveOptions) error {
		opts.VendorPath = vendorPath
		return nil
	}
}

// WithEnableCache sets the flag to enable the cache during the resolving the remote package.
func WithEnableCache(enableCache bool) ResolveOption {
	return func(opts *ResolveOptions) error {
		opts.EnableCache = enableCache
		return nil
	}
}

// WithCachePath sets the path of the cache.
func WithCachePath(cachePath string) ResolveOption {
	return func(opts *ResolveOptions) error {
		opts.CachePath = cachePath
		return nil
	}
}

// WithKclPkg sets the kcl package to be resolved.
func WithResolveKclPkg(kpkg *pkg.KclPkg) ResolveOption {
	return func(opts *ResolveOptions) error {
		opts.kpkg = kpkg
		return nil
	}
}

// DepsResolver is the resolver for resolving dependencies.
type DepsResolver struct {
	kpmClient    *KpmClient
	resolveFuncs []resolveFunc
}

// NewDepsResolver creates a new DepsResolver.
func NewDepsResolver(kpmClient *KpmClient) *DepsResolver {
	return &DepsResolver{
		kpmClient:    kpmClient,
		resolveFuncs: []resolveFunc{},
	}
}

// AddResolveFunc adds a resolve function to the DepsResolver.
func (dr *DepsResolver) AddResolveFunc(rf resolveFunc) {
	if dr.resolveFuncs == nil {
		dr.resolveFuncs = []resolveFunc{}
	}

	dr.resolveFuncs = append(dr.resolveFuncs, rf)
}

// Resolve resolves the dependencies of the package.
func (dr *DepsResolver) Resolve(options ...ResolveOption) error {
	opts := &ResolveOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return err
		}
	}

	var accessFunc accessFunc

	// A custom function to download the package consider the offline mode.
	downloadPkgTo := func(source *downloader.Source, localPath string) error {
		credCli, err := dr.kpmClient.GetCredsClient()
		if err != nil {
			return err
		}

		var pkgPath string

		if !opts.Offline {
			tmpDir, err := os.MkdirTemp("", "")
			if err != nil {
				return err
			}

			if source.Git != nil {
				tmpDir = filepath.Join(tmpDir, constants.GitScheme)
			}
			defer os.RemoveAll(tmpDir)
			err = dr.kpmClient.DepDownloader.Download(*downloader.NewDownloadOptions(
				downloader.WithLocalPath(tmpDir),
				downloader.WithSource(*source),
				downloader.WithLogWriter(dr.kpmClient.GetLogWriter()),
				downloader.WithSettings(*dr.kpmClient.GetSettings()),
				downloader.WithCredsClient(credCli),
				downloader.WithCachePath(opts.CachePath),
				downloader.WithEnableCache(opts.EnableCache),
				downloader.WithInsecureSkipTLSverify(dr.kpmClient.insecureSkipTLSverify),
			))

			if err != nil {
				return err
			}
			pkgPath = tmpDir
			if source.Git != nil && len(source.Git.Package) > 0 {
				pkgPath, err = utils.FindPackage(tmpDir, source.Git.Package)
				if err != nil {
					return err
				}
			}
		} else {
			pkgPath = filepath.Join(opts.CachePath, cacheFileNameFromSource(source))
			if !utils.DirExists(pkgPath) {
				return fmt.Errorf("package not found in the %s", pkgPath)
			} else {
				if source.Git != nil && len(source.Git.Package) > 0 {
					pkgPath, err = utils.FindPackage(pkgPath, source.Git.Package)
					if err != nil {
						return err
					}
				}
			}
		}

		if !utils.DirExists(localPath) {
			err = copy.Copy(pkgPath, localPath)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// A custom function to access the package consider the vendor path.
	accessFunc = func(source *downloader.Source) (*pkg.KclPkg, error) {
		if opts.EnableVendor {
			vendorFullPath := filepath.Join(opts.VendorPath, cacheFileNameFromSource(source))
			if utils.DirExists(vendorFullPath) {
				if source.GetPackage() != "" {
					tempVendorFullPath, err := utils.FindPackage(vendorFullPath, source.GetPackage())
					if err != nil {
						return nil, err
					}
					vendorFullPath = tempVendorFullPath
				}
				return dr.kpmClient.LoadPkgFromPath(vendorFullPath)
			} else {
				downloadPkgTo(source, vendorFullPath)
				if source.GetPackage() != "" {
					tempVendorFullPath, err := utils.FindPackage(vendorFullPath, source.GetPackage())
					if err != nil {
						return nil, err
					}
					vendorFullPath = tempVendorFullPath
				}
				kclPkg, err := dr.kpmClient.LoadPkgFromPath(vendorFullPath)
				if err != nil {
					return nil, err
				}
				return kclPkg, nil
			}
		} else {
			cacheFullPath := filepath.Join(opts.CachePath, cacheFileNameFromSource(source))
			if utils.DirExists(cacheFullPath) && utils.DirExists(filepath.Join(cacheFullPath, constants.KCL_MOD)) {
				if source.GetPackage() != "" {
					tempCacheFullPath, err := utils.FindPackage(cacheFullPath, source.GetPackage())
					if err != nil {
						return nil, err
					}
					cacheFullPath = tempCacheFullPath
				}
				return dr.kpmClient.LoadPkgFromPath(cacheFullPath)
			} else {
				err := os.MkdirAll(cacheFullPath, 0755)
				if err != nil {
					return nil, err
				}
				err = downloadPkgTo(source, cacheFullPath)
				if err != nil {
					return nil, err
				}
				if source.GetPackage() != "" {
					tempCacheFullPath, err := utils.FindPackage(cacheFullPath, source.GetPackage())
					if err != nil {
						return nil, err
					}
					cacheFullPath = tempCacheFullPath
				}
				return dr.kpmClient.LoadPkgFromPath(cacheFullPath)
			}
		}
	}

	// visitorSelectorFunc selects the visitor for the source.
	// For remote source, it will use the RemoteVisitor and enable the cache.
	// For local source, it will use the PkgVisitor.
	visitorSelectorFunc := func(source *downloader.Source) (Visitor, error) {
		if source.IsNilSource() {
			return nil, fmt.Errorf("the dependency source is nil")
		}

		if source.IsRemote() {
			PkgVisitor := NewRemoteVisitor(NewPkgVisitor(dr.kpmClient))
			PkgVisitor.EnableCache = opts.EnableCache
			if opts.CachePath == "" {
				PkgVisitor.CachePath = dr.kpmClient.homePath
			} else {
				PkgVisitor.CachePath = opts.CachePath
			}
			return PkgVisitor, nil
		} else {
			accessFunc = nil
			return NewVisitor(*source, dr.kpmClient), nil
		}
	}

	kpkg := opts.kpkg
	if kpkg == nil {
		return fmt.Errorf("kcl package is nil")
	}

	modDeps := kpkg.ModFile.Dependencies.Deps
	if modDeps == nil {
		return fmt.Errorf("kcl.mod dependencies is nil")
	}

	// Iterate all the dependencies of the package in kcl.mod and resolve each dependency.
	for _, depName := range modDeps.Keys() {
		dep, ok := modDeps.Get(depName)
		if !ok {
			return fmt.Errorf("failed to get dependency %s", depName)
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

		visitor, err := visitorSelectorFunc(depSource)
		if err != nil {
			return err
		}

		// visitFunc is the function for visiting the package.
		// It will traverse the dependency graph and visit each dependency by source.
		visitFunc := func(kclPkg *pkg.KclPkg) error {
			dep.FromKclPkg(kclPkg)
			for _, resolveFunc := range dr.resolveFuncs {
				err := resolveFunc(&dep, kpkg)
				if err != nil {
					return err
				}
			}

			// Recursively resolve the dependencies of the dependency.
			err = dr.Resolve(
				WithResolveKclPkg(kclPkg),
				WithEnableCache(opts.EnableCache),
				WithCachePath(opts.CachePath),
			)
			if err != nil {
				return err
			}
			// }

			return nil
		}

		err = visitor.Visit(
			WithVisitSource(depSource),
			WithVisitFunc(visitFunc),
			WithAccessFunc(accessFunc),
		)

		if err != nil {
			return err
		}
	}

	return nil
}

// TODO: After the new local storage structure is complete,
//
// this section should be replaced with the new storage structure instead of the cache path according to the <Cache Path>/<Package Name>.
//
//	https://github.com/kcl-lang/kpm/issues/384
func cacheFileNameFromSource(source *downloader.Source) string {
	var pkgFullName string
	if source.Registry != nil && len(source.Registry.Version) != 0 {
		pkgFullName = fmt.Sprintf("%s_%s", filepath.Base(source.Registry.Oci.Repo), source.Registry.Version)
	}
	if source.Oci != nil && len(source.Oci.Tag) != 0 {
		pkgFullName = fmt.Sprintf("%s_%s", filepath.Base(source.Oci.Repo), source.Oci.Tag)
	}

	if source.Git != nil && len(source.Git.Tag) != 0 {
		gitUrl := strings.TrimSuffix(source.Git.Url, filepath.Ext(source.Git.Url))
		pkgFullName = fmt.Sprintf("%s_%s", filepath.Base(gitUrl), source.Git.Tag)
	}
	if source.Git != nil && len(source.Git.Branch) != 0 {
		gitUrl := strings.TrimSuffix(source.Git.Url, filepath.Ext(source.Git.Url))
		pkgFullName = fmt.Sprintf("%s_%s", filepath.Base(gitUrl), source.Git.Branch)
	}
	if source.Git != nil && len(source.Git.Commit) != 0 {
		gitUrl := strings.TrimSuffix(source.Git.Url, filepath.Ext(source.Git.Url))
		pkgFullName = fmt.Sprintf("%s_%s", filepath.Base(gitUrl), source.Git.Commit)
	}

	return pkgFullName
}
