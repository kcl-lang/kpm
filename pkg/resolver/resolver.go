package resolver

import (
	"fmt"
	"io"
	"path/filepath"

	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
	"kcl-lang.io/kpm/pkg/visitor"
)

// ResolveOption is the option for resolving dependencies.
type ResolveOption func(*ResolveOptions) error

// resolveFunc is the function for resolving each dependency when traversing the dependency graph.
// currentPkg is the current package to be resolved and parentPkg is the parent package of the current package.
type resolveFunc func(dep *pkg.Dependency, parentPkg *pkg.KclPkg) error

type ResolveOptions struct {
	// Source is the source of the package to be pulled.
	// Including git, oci, local.
	Source *downloader.Source
	// EnableCache is the flag to enable the cache during the resolving the remote package.
	EnableCache bool
	// CachePath is the path of the cache.
	CachePath string
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

// WithSource sets the source of the package to be resolved.
func WithSource(source *downloader.Source) ResolveOption {
	return func(opts *ResolveOptions) error {
		opts.Source = source
		return nil
	}
}

// WithResolveSourceUrl sets the source of the package to be resolved by the source url.
func WithSourceUrl(sourceUrl string) ResolveOption {
	return func(opts *ResolveOptions) error {
		source, err := downloader.NewSourceFromStr(sourceUrl)
		if err != nil {
			return err
		}
		opts.Source = source
		return nil
	}
}

// DepsResolver is the resolver for resolving dependencies.
type DepsResolver struct {
	DefaultCachePath      string
	InsecureSkipTLSverify bool
	Downloader            downloader.Downloader
	Settings              *settings.Settings
	LogWriter             io.Writer
	ResolveFuncs          []resolveFunc
}

// Resolve resolves the dependencies of the package.
func (dr *DepsResolver) Resolve(options ...ResolveOption) error {
	opts := &ResolveOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return err
		}
	}

	// visitorSelectorFunc selects the visitor for the source.
	// For remote source, it will use the RemoteVisitor and enable the cache.
	// For local source, it will use the PkgVisitor.
	visitorSelectorFunc := func(source *downloader.Source) (visitor.Visitor, error) {
		pkgVisitor := &visitor.PkgVisitor{
			Settings:  dr.Settings,
			LogWriter: dr.LogWriter,
		}

		if source.IsRemote() {
			var cachePath string
			if opts.CachePath != "" {
				cachePath = opts.CachePath
			} else {
				cachePath = dr.DefaultCachePath
			}

			return &visitor.RemoteVisitor{
				PkgVisitor:            pkgVisitor,
				Downloader:            dr.Downloader,
				InsecureSkipTLSverify: dr.InsecureSkipTLSverify,
				EnableCache:           opts.EnableCache,
				CachePath:             cachePath,
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

	// visitFunc is the function for visiting the package.
	// It will traverse the dependency graph and visit each dependency by source.
	visitFunc := func(kclPkg *pkg.KclPkg) error {
		// Traverse the all dependencies of the package.
		for _, depKey := range kclPkg.ModFile.Deps.Keys() {
			dep, ok := kclPkg.ModFile.Deps.Get(depKey)
			if !ok {
				break
			}

			// Get the dependency source.
			var depSource downloader.Source
			// If the dependency source is a local path and the path is not absolute, transform the path to absolute path.
			if dep.Source.IsLocalPath() && !filepath.IsAbs(dep.Source.Path) {
				depSource = downloader.Source{
					Local: &downloader.Local{
						Path: filepath.Join(kclPkg.HomePath, dep.Source.Path),
					},
				}
			} else {
				depSource = dep.Source
			}

			// Get the visitor for the dependency source.
			visitor, err := visitorSelectorFunc(&depSource)
			if err != nil {
				return err
			}

			// Visit this dependency and current package as the parent package.
			err = visitor.Visit(&depSource,
				func(childPkg *pkg.KclPkg) error {
					for _, resolveFunc := range dr.ResolveFuncs {
						err := resolveFunc(&dep, kclPkg)
						if err != nil {
							return err
						}
					}
					return nil
				},
			)

			if err != nil {
				return err
			}

			// Recursively resolve the dependencies of the dependency.
			err = dr.Resolve(
				WithSource(&depSource),
				WithEnableCache(opts.EnableCache),
				WithCachePath(opts.CachePath),
			)
			if err != nil {
				return err
			}
		}

		return nil
	}

	visitor, err := visitorSelectorFunc(opts.Source)
	if err != nil {
		return err
	}

	return visitor.Visit(opts.Source, visitFunc)
}
