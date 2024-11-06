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
	Source     *downloader.Source
	KclPkg     *pkg.KclPkg
	NewPkgName string
}

type AddOption func(*AddOptions) error

func WithNewPkgName(newPkgName string) AddOption {
	return func(opts *AddOptions) error {
		opts.NewPkgName = newPkgName
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
		var modSpec *downloader.ModSpec
		if depSource.ModSpec.IsNil() {
			modSpec = &downloader.ModSpec{
				Name:    depPkg.ModFile.Pkg.Name,
				Version: depPkg.ModFile.Pkg.Version,
			}
			depSource.ModSpec = modSpec
		}

		var depName string
		if opts.NewPkgName != "" {
			depName = opts.NewPkgName
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

	// Iterate the dependencies and update the kcl.mod and kcl.mod.lock respectively.
	_, err = c.Update(
		WithUpdatedKclPkg(addedPkg),
	)

	if err != nil {
		return err
	}
	return nil
}
