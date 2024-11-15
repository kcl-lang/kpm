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
	// kMod is the module to be resolved.
	kMod *pkg.KclPkg
	// EnableCache is the flag to enable the cache during the resolving the remote package.
	EnableCache bool
	// CachePath is the path of the cache.
	CachePath string
}

func WithResolveKclMod(kMod *pkg.KclPkg) ResolveOption {
	return func(opts *ResolveOptions) error {
		opts.kMod = kMod
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
				VisitedSpace:          cachePath,
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

	kMod := opts.kMod
	if kMod == nil {
		return fmt.Errorf("kcl module is nil")
	}

	modDeps := kMod.ModFile.Dependencies.Deps
	if modDeps == nil {
		return fmt.Errorf("kcl.mod dependencies is nil")
	}

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
					Path: filepath.Join(kMod.HomePath, dep.Source.Local.Path),
				},
			}
		} else {
			depSource = &dep.Source
		}

		depVisitor, err := visitorSelectorFunc(depSource)
		if err != nil {
			return err
		}

		err = depVisitor.Visit(depSource, func(kclMod *pkg.KclPkg) error {
			dep.LocalFullPath = kclMod.HomePath
			for _, resolveFunc := range dr.ResolveFuncs {
				err := resolveFunc(&dep, kMod)
				if err != nil {
					return err
				}
			}
			err = dr.Resolve(
				WithResolveKclMod(kclMod),
				WithEnableCache(opts.EnableCache),
				WithCachePath(opts.CachePath),
			)
			if err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}
