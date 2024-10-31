package client

import (
	"fmt"
	"path/filepath"

	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/utils"
	"kcl-lang.io/kpm/pkg/visitor"
)

type AddOptions struct {
	// Source is the source of the package to be pulled.
	// Including git, oci, local.
	Sources []*downloader.Source
	KclPkg  *pkg.KclPkg
}

type AddOption func(*AddOptions) error

func WithAddSource(source *downloader.Source) AddOption {
	return func(opts *AddOptions) error {
		if opts.Sources == nil {
			opts.Sources = make([]*downloader.Source, 0)
		}
		opts.Sources = append(opts.Sources, source)
		return nil
	}
}

func WithAddSources(sources []*downloader.Source) AddOption {
	return func(ro *AddOptions) error {
		ro.Sources = sources
		return nil
	}
}

func WithAddSourceUrl(sourceUrl string) AddOption {
	return func(opts *AddOptions) error {
		if opts.Sources == nil {
			opts.Sources = make([]*downloader.Source, 0)
		}
		source, err := downloader.NewSourceFromStr(sourceUrl)
		if err != nil {
			return err
		}
		opts.Sources = append(opts.Sources, source)
		return nil
	}
}

func WithAddSourceUrls(sourceUrls []string) AddOption {
	return func(opts *AddOptions) error {
		var sources []*downloader.Source
		for _, sourceUrl := range sourceUrls {
			source, err := downloader.NewSourceFromStr(sourceUrl)
			if err != nil {
				return err
			}
			sources = append(sources, source)
		}
		opts.Sources = sources
		return nil
	}
}

func WithAddKclPkg(kclPkg *pkg.KclPkg) AddOption {
	return func(opts *AddOptions) error {
		opts.KclPkg = kclPkg
		return nil
	}
}

func NewAddOptions(opts ...AddOption) *AddOptions {
	ao := &AddOptions{}
	for _, opt := range opts {
		opt(ao)
	}
	return ao
}

func (c *KpmClient) Add(options ...AddOption) error {
	opts := &AddOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return err
		}
	}
	addedPkg := opts.KclPkg

	visitorSelector := func(source *downloader.Source) (visitor.Visitor, error) {
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
				VisitedSpace:          c.homePath,
			}, nil
		} else if source.IsLocalTarPath() || source.IsLocalTgzPath() {
			return visitor.NewArchiveVisitor(pkgVisitor), nil
		} else if source.IsLocalPath() {
			return pkgVisitor, nil
		} else {
			return nil, fmt.Errorf("unsupported source")
		}
	}

	for _, depSource := range opts.Sources {
		// Set the default OCI registry and repo if the source is nil and the package spec is not nil.
		if depSource.IsNilSource() && !depSource.ModSpec.IsNil() {
			depSource.Oci = &downloader.Oci{
				Reg:  c.GetSettings().Conf.DefaultOciRegistry,
				Repo: utils.JoinPath(c.GetSettings().Conf.DefaultOciRepo, depSource.ModSpec.Name),
				Tag:  depSource.ModSpec.Version,
			}
		}

		var fullSouce *downloader.Source
		// Transform the relative path to the full path.
		if depSource.IsLocalPath() && !filepath.IsAbs(depSource.Path) {
			fullSouce = &downloader.Source{
				ModSpec: depSource.ModSpec,
				Local: &downloader.Local{
					Path: filepath.Join(addedPkg.HomePath, depSource.Path),
				},
			}
		} else {
			fullSouce = depSource
		}

		visitor, err := visitorSelector(fullSouce)
		if err != nil {
			return err
		}

		// Visit the dependency source
		// If the dependency is remote, the visitor will download it to the local.
		// If the dependency is already in local cache, the visitor will not download it again.
		err = visitor.Visit(fullSouce, func(depPkg *pkg.KclPkg) error {
			dep := pkg.Dependency{
				Name:          depPkg.ModFile.Pkg.Name,
				FullName:      depPkg.GetPkgFullName(),
				Version:       depPkg.ModFile.Pkg.Version,
				LocalFullPath: depPkg.HomePath,
				Source:        *depSource,
			}

			// Add the dependency to the kcl.mod file.
			if modExistDep, ok := addedPkg.ModFile.Dependencies.Deps.Get(dep.Name); ok {
				if less, err := modExistDep.VersionLessThan(&dep); less && err == nil {
					addedPkg.ModFile.Dependencies.Deps.Set(dep.Name, dep)
				}
			} else {
				addedPkg.ModFile.Dependencies.Deps.Set(dep.Name, dep)
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	// Iterate the dependencies and update the kcl.mod and kcl.mod.lock respectively.
	_, err := c.Update(
		WithUpdatedKclPkg(addedPkg),
	)

	if err != nil {
		return err
	}
	return nil
}
