package client

import (
	"fmt"
	"path/filepath"

	orderedmap "github.com/elliotchance/orderedmap/v2"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/features"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
	"kcl-lang.io/kpm/pkg/visitor"
)

type AddOptions struct {
	// Source is the source of the package to be pulled.
	// Including git, oci, local.
	Source *downloader.Source
	KclPkg *pkg.KclPkg
	// IsDevDep, when true, routes the new dependency into the
	// `[dev_dependencies]` section of kcl.mod / kcl.mod.lock instead of the
	// regular `[dependencies]` table. Dev deps are only used by `kcl test`
	// and are excluded from `kcl run`.
	IsDevDep bool
}

type AddOption func(*AddOptions) error

func WithAlias(alias string) AddOption {
	return func(opts *AddOptions) error {
		if opts.Source == nil {
			return fmt.Errorf("source cannot be nil")
		}
		if opts.Source.ModSpec.IsNil() {
			return fmt.Errorf("modSpec cannot be nil")
		}
		opts.Source.ModSpec.Alias = alias
		return nil
	}
}

func WithAddModSpec(modSpec *downloader.ModSpec) AddOption {
	return func(opts *AddOptions) error {
		if modSpec == nil {
			return fmt.Errorf("modSpec cannot be nil")
		}
		if opts.Source == nil {
			opts.Source = &downloader.Source{
				ModSpec: modSpec,
			}
		} else {
			opts.Source.ModSpec = modSpec
		}

		return nil
	}
}

func WithAddSource(source *downloader.Source) AddOption {
	return func(opts *AddOptions) error {
		if source == nil {
			return fmt.Errorf("source cannot be nil")
		}
		opts.Source = source
		return nil
	}
}

func WithAddSourceUrl(sourceUrl string) AddOption {
	return func(opts *AddOptions) error {
		source, err := downloader.NewSourceFromStr(sourceUrl)
		if err != nil {
			return err
		}
		opts.Source = source
		return nil
	}
}

func WithAddKclPkg(kclPkg *pkg.KclPkg) AddOption {
	return func(opts *AddOptions) error {
		opts.KclPkg = kclPkg
		return nil
	}
}

// WithIsDevDep marks the dependency being added as a dev-only dep. It will
// be written under [dev_dependencies] in kcl.mod (and kcl.mod.lock) so that
// it is only resolved by `kcl test`, not by `kcl run`.
func WithIsDevDep(isDev bool) AddOption {
	return func(opts *AddOptions) error {
		opts.IsDevDep = isDev
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
	depSource := opts.Source
	specOnly := depSource.SpecOnly()

	var succeedMsgInfo string

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
		reporter.ReportMsgTo(
			fmt.Sprintf("adding dependency '%s'", depPkg.GetPkgName()),
			c.logWriter,
		)

		var depAlias string
		if depSource.ModSpec != nil {
			depAlias = depSource.ModSpec.Alias
		}

		var modSpec *downloader.ModSpec
		if depSource.ModSpec.IsNil() {
			modSpec = &downloader.ModSpec{
				Name:    depPkg.ModFile.Pkg.Name,
				Version: depPkg.ModFile.Pkg.Version,
				Alias:   depAlias,
			}
			depSource.ModSpec = modSpec
		}

		var depName string
		if depAlias != "" {
			depName = depAlias
		} else {
			depName = depPkg.ModFile.Pkg.Name
		}

		dep := pkg.Dependency{
			Name:          depName,
			FullName:      depPkg.GetPkgFullName(),
			Version:       depPkg.ModFile.Pkg.Version,
			LocalFullPath: depPkg.HomePath,
			Source:        *depSource,
		}

		// If the dependency is spec only, add the dependency to the backup dep ui,
		// to generate the dependency like 'helloworld = "0.0.1"' instead of 'helloworld = { oci = "ghcr.io/kcl-lang/helloworld", tag = "0.1.4" }'.
		if specOnly {
			addedPkg.BackupDepUI(dep.Name, &pkg.Dependency{
				Name:    dep.Name,
				Version: dep.Version,
				Source: downloader.Source{
					ModSpec: &downloader.ModSpec{
						Name:    dep.Name,
						Version: dep.Version,
						Alias:   depAlias,
					},
				},
			})
		}

		// Choose the right table in kcl.mod: [dev_dependencies] for dev-only
		// deps (used by `kcl test`), [dependencies] for everything else.
		targetDeps := &addedPkg.ModFile.Dependencies
		if opts.IsDevDep {
			if addedPkg.ModFile.DevDependencies.Deps == nil {
				addedPkg.ModFile.DevDependencies.Deps = orderedmap.NewOrderedMap[string, pkg.Dependency]()
			}
			targetDeps = &addedPkg.ModFile.DevDependencies
		}

		if ok, err := features.Enabled(features.SupportMVS); err == nil && ok {
			// Add the dependency to the chosen section and select the greater
			// version of the dependency in dependencies graph.
			if modExistDep, ok := targetDeps.Deps.Get(dep.Name); ok {
				if less, err := modExistDep.VersionLessThan(&dep); less && err == nil {
					targetDeps.Deps.Set(dep.Name, dep)
				}
			} else {
				targetDeps.Deps.Set(dep.Name, dep)
			}
		} else {
			// Add the dependency to the chosen section directly.
			targetDeps.Deps.Set(dep.Name, dep)
		}
		succeedMsgInfo = fmt.Sprintf("add dependency '%s:%s' successfully", depPkg.ModFile.Pkg.Name, depPkg.ModFile.Pkg.Version)
		return nil
	})
	if err != nil {
		return err
	}

	// Iterate the dependencies and update the kcl.mod and kcl.mod.lock respectively.
	_, err = c.Update(
		WithUpdatedKclPkg(addedPkg),
	)

	if err != nil {
		return err
	}
	reporter.ReportMsgTo(succeedMsgInfo, c.logWriter)
	return nil
}
