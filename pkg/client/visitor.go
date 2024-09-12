package client

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/utils"
)

type VisitOptions struct {
	Source *downloader.Source
	// The funcs to visit the package.
	visitFuncs []visitFunc
	// The func to download/access the package.
	// For remote package, it will download the package.
	// For local package, it will find the package.
	accessFunc *accessFunc
}

type VisitOption func(*VisitOptions) error

// The func help to download or find the package from the source.
type accessFunc func(source *downloader.Source) (*pkg.KclPkg, error)

// The func help to visit the package after access the package.
type visitFunc func(pkg *pkg.KclPkg) error

// WithVisitFunc sets the visitFunc for visiting the package.
func WithVisitFunc(visitFunc visitFunc) VisitOption {
	return func(opts *VisitOptions) error {
		opts.visitFuncs = append(opts.visitFuncs, visitFunc)
		return nil
	}
}

// WithAccessFunc sets the accessFunc for accessing the package.
func WithAccessFunc(accessFunc accessFunc) VisitOption {
	return func(opts *VisitOptions) error {
		opts.accessFunc = &accessFunc
		return nil
	}
}

// WithSource sets the source of the package to be visited.
func WithVisitSource(source *downloader.Source) VisitOption {
	return func(opts *VisitOptions) error {
		opts.Source = source
		return nil
	}
}

// Visitor is the interface for visiting a package which is a local path, a remote git/oci path, or a local tar path.
type Visitor interface {
	Visit(options ...VisitOption) error
}

// PkgVisitor is the visitor for visiting a local package.
type PkgVisitor struct {
	kpmcli *KpmClient
}

// NewPkgVisitor creates a new PkgVisitor.
func NewPkgVisitor(kpmcli *KpmClient) *PkgVisitor {
	return &PkgVisitor{
		kpmcli: kpmcli,
	}
}

// Visit visits a local package.
func (pv *PkgVisitor) Visit(options ...VisitOption) error {
	opts := &VisitOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return err
		}
	}

	s := opts.Source
	if s == nil {
		return fmt.Errorf("source is nil")
	}

	if !s.IsLocalPath() {
		return fmt.Errorf("source is not local")
	}

	var accessFunc accessFunc
	var kclPkg *pkg.KclPkg
	if opts.accessFunc == nil || *opts.accessFunc == nil {
		accessFunc = func(source *downloader.Source) (*pkg.KclPkg, error) {
			// Find the root path of the source.
			// There must be a kcl.mod file in the root path.
			rootPath, err := s.FindRootPath()
			if err != nil {
				return nil, err
			}

			kclPkg, err = pv.kpmcli.LoadPkgFromPath(rootPath)
			if err != nil {
				return nil, err
			}
			return kclPkg, nil
		}
	} else {
		accessFunc = *opts.accessFunc
	}

	kclPkg, err := accessFunc(s)
	if err != nil {
		return err
	}

	for _, visitFunc := range opts.visitFuncs {
		err = visitFunc(kclPkg)
		if err != nil {
			return err
		}
	}

	return nil
}

// VirtualPkgVisitor is the visitor for visiting a package which do not have a kcl.mod file in the root path.
type VirtualPkgVisitor struct {
	*PkgVisitor
}

// NewVirtualPkgVisitor creates a new VirtualPkgVisitor.
func NewVirtualPkgVisitor(pv *PkgVisitor) *VirtualPkgVisitor {
	return &VirtualPkgVisitor{
		PkgVisitor: pv,
	}
}

// Visit visits a package which do not have a kcl.mod file in the root path.
// It will create a virtual kcl.mod file in the root path.
// And then kcl.mod file will be cleaned after the visitFunc is executed.
func (vpv *VirtualPkgVisitor) Visit(options ...VisitOption) error {
	opts := &VisitOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return err
		}
	}

	s := opts.Source
	if s == nil {
		return fmt.Errorf("source is nil")
	}

	if !s.IsLocalPath() {
		return fmt.Errorf("source is not local")
	}

	var accessFunc accessFunc
	var kclPkg *pkg.KclPkg
	if opts.accessFunc == nil || *opts.accessFunc == nil {
		accessFunc = func(source *downloader.Source) (*pkg.KclPkg, error) {
			sourcePath, err := s.ToFilePath()
			if err != nil {
				return nil, err
			}
			initOpts := opt.InitOptions{
				Name:     "vPkg_" + uuid.New().String(),
				InitPath: sourcePath,
			}
			kpkg := pkg.NewKclPkg(&initOpts)
			return &kpkg, nil
		}
	} else {
		accessFunc = *opts.accessFunc
	}

	kclPkg, err := accessFunc(s)
	if err != nil {
		return err
	}

	for _, visitFunc := range opts.visitFuncs {
		err = visitFunc(kclPkg)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoteVisitor is the visitor for visiting a remote package.
type RemoteVisitor struct {
	*PkgVisitor
	EnableCache bool
	CachePath   string
}

// NewRemoteVisitor creates a new RemoteVisitor.
func NewRemoteVisitor(pv *PkgVisitor) *RemoteVisitor {
	return &RemoteVisitor{
		PkgVisitor: pv,
	}
}

// Visit visits a remote package.
// It will download the remote package to a temporary directory.
// And the tmp directory will be cleaned after the visitFunc is executed.
func (rv *RemoteVisitor) Visit(options ...VisitOption) error {
	opts := &VisitOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return err
		}
	}

	s := opts.Source
	if s == nil {
		return fmt.Errorf("source is nil")
	}

	if !s.IsRemote() {
		return fmt.Errorf("source is not remote")
	}

	var accessFunc accessFunc
	var kclPkg *pkg.KclPkg
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}

	if s.Git != nil {
		tmpDir = filepath.Join(tmpDir, constants.GitScheme)
	}
	defer os.RemoveAll(tmpDir)

	if opts.accessFunc == nil || *opts.accessFunc == nil {
		accessFunc = func(source *downloader.Source) (*pkg.KclPkg, error) {

			credCli, err := rv.kpmcli.GetCredsClient()
			if err != nil {
				return nil, err
			}

			err = rv.kpmcli.DepDownloader.Download(*downloader.NewDownloadOptions(
				downloader.WithLocalPath(tmpDir),
				downloader.WithSource(*s),
				downloader.WithLogWriter(rv.kpmcli.GetLogWriter()),
				downloader.WithSettings(*rv.kpmcli.GetSettings()),
				downloader.WithCredsClient(credCli),
				downloader.WithCachePath(rv.CachePath),
				downloader.WithEnableCache(rv.EnableCache),
				downloader.WithInsecureSkipTLSverify(rv.kpmcli.insecureSkipTLSverify),
			))

			if err != nil {
				return nil, err
			}
			pkgPath := tmpDir
			if s.Git != nil && len(s.Git.Package) > 0 {
				pkgPath, err = utils.FindPackage(tmpDir, s.Git.Package)
				if err != nil {
					return nil, err
				}
			}

			kclPkg, err := rv.kpmcli.LoadPkgFromPath(pkgPath)
			if err != nil {
				return nil, err
			}
			return kclPkg, nil
		}
	} else {
		accessFunc = *opts.accessFunc
	}

	kclPkg, err = accessFunc(s)
	if err != nil {
		return err
	}

	for _, visitFunc := range opts.visitFuncs {
		err = visitFunc(kclPkg)
		if err != nil {
			return err
		}
	}

	return nil
}

// ArchiveVisitor is the visitor for visiting a package which is a local tar/tgz path.
type ArchiveVisitor struct {
	*PkgVisitor
}

// NewArchiveVisitor creates a new ArchiveVisitor.
func NewArchiveVisitor(pv *PkgVisitor) *ArchiveVisitor {
	return &ArchiveVisitor{
		PkgVisitor: pv,
	}
}

// extractArchiveFromSource will extract the archive file to the extractedPath.
func (av *ArchiveVisitor) extractArchiveFromSource(s *downloader.Source, extractedPath string) error {
	if !s.IsLocalTarPath() && !s.IsLocalTgzPath() {
		return fmt.Errorf("source is not local tar path")
	}

	filepath, err := s.ToFilePath()
	if err != nil {
		return err
	}

	if s.IsLocalTarPath() {
		err = utils.UnTarDir(filepath, extractedPath)
	} else if s.IsLocalTgzPath() {
		err = utils.ExtractTarball(filepath, extractedPath)
	}

	if err != nil {
		return err
	}

	return nil
}

// Visit visits a package which is a local tar/tgz path.
// It will extract the archive file to a temporary directory.
// And the tmp directory will be cleaned after the visitFunc is executed.
func (av *ArchiveVisitor) Visit(options ...VisitOption) error {

	opts := &VisitOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return err
		}
	}

	s := opts.Source
	if s == nil {
		return fmt.Errorf("source is nil")
	}

	if !s.IsLocalTarPath() && !s.IsLocalTgzPath() {
		return fmt.Errorf("source is not local tar path")
	}

	var accessFunc accessFunc
	var kclPkg *pkg.KclPkg
	if opts.accessFunc == nil || *opts.accessFunc == nil {
		accessFunc = func(source *downloader.Source) (*pkg.KclPkg, error) {
			tmpDir, err := os.MkdirTemp("", "")
			if err != nil {
				return nil, err
			}

			defer os.RemoveAll(tmpDir)

			err = av.extractArchiveFromSource(s, tmpDir)

			if err != nil {
				return nil, err
			}

			kclPkg, err := av.kpmcli.LoadPkgFromPath(tmpDir)

			if err != nil {
				return nil, err
			}
			return kclPkg, nil
		}
	} else {
		accessFunc = *opts.accessFunc
	}

	kclPkg, err := accessFunc(s)
	if err != nil {
		return err
	}

	for _, visitFunc := range opts.visitFuncs {
		err = visitFunc(kclPkg)
		if err != nil {
			return err
		}
	}

	return nil
}

// NewVisitor is a factory function to create a new Visitor.
func NewVisitor(source downloader.Source, kpmcli *KpmClient) Visitor {
	if source.IsRemote() {
		return NewRemoteVisitor(NewPkgVisitor(kpmcli))
	} else if source.IsLocalTarPath() || source.IsLocalTgzPath() {
		return NewArchiveVisitor(NewPkgVisitor(kpmcli))
	} else if source.IsLocalPath() {
		rootPath, err := source.FindRootPath()
		if err != nil {
			return nil
		}
		kclmodpath := filepath.Join(rootPath, constants.KCL_MOD)
		if utils.DirExists(kclmodpath) {
			return NewPkgVisitor(kpmcli)
		} else {
			return NewVirtualPkgVisitor(NewPkgVisitor(kpmcli))
		}
	} else {
		return nil
	}
}
