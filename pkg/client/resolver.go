package client

import (
	"path/filepath"

	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
)

// ResolveOption is the option for resolving dependencies.
type ResolveOption func(*ResolveOptions) error

// resolveFunc is the function for resolving each dependency when traversing the dependency graph.
// currentPkg is the current package to be resolved and parentPkg is the parent package of the current package.
type resolveFunc func(currentPkg, parentPkg *pkg.KclPkg) error

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

// WithPkgSource sets the source of the package to be resolved.
func WithPkgSource(source *downloader.Source) ResolveOption {
	return func(opts *ResolveOptions) error {
		opts.Source = source
		return nil
	}
}

// WithPkgSourceUrl sets the source of the package to be resolved by the source url.
func WithPkgSourceUrl(sourceUrl string) ResolveOption {
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

	// visitorSelectorFunc selects the visitor for the source.
	// For remote source, it will use the RemoteVisitor and enable the cache.
	// For local source, it will use the PkgVisitor.
	visitorSelectorFunc := func(source *downloader.Source) (Visitor, error) {
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
			return NewVisitor(*opts.Source, dr.kpmClient), nil
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
					for _, resolveFunc := range dr.resolveFuncs {
						err := resolveFunc(childPkg, kclPkg)
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
				WithPkgSource(&depSource),
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
