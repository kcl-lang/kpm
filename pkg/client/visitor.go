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

type visitFunc func(pkg *pkg.KclPkg) error

// Visitor is the interface for visiting a package which is a local path, a remote git/oci path, or a local tar path.
type Visitor interface {
	Visit(s *downloader.Source, v visitFunc) error
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
func (pv *PkgVisitor) Visit(s *downloader.Source, v visitFunc) error {
	if !s.IsLocalPath() {
		return fmt.Errorf("source is not local")
	}
	// Find the root path of the source.
	// There must be a kcl.mod file in the root path.
	rootPath, err := s.FindRootPath()
	if err != nil {
		return err
	}

	kclPkg, err := pv.kpmcli.LoadPkgFromPath(rootPath)
	if err != nil {
		return err
	}

	return v(kclPkg)
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
func (vpv *VirtualPkgVisitor) Visit(s *downloader.Source, v visitFunc) error {
	if !s.IsLocalPath() {
		return fmt.Errorf("source is not local")
	}

	sourcePath, err := s.ToFilePath()
	if err != nil {
		return err
	}

	// If the source path does not contain a kcl.mod file, create a virtual kcl.mod file.
	vKclModPath := filepath.Join(sourcePath, constants.KCL_MOD)
	if !utils.DirExists(vKclModPath) {
		// After the visitFunc is executed, clean the virtual kcl.mod file.
		defer func() error {
			vKclModLockPath := filepath.Join(sourcePath, constants.KCL_MOD_LOCK)
			if utils.DirExists(vKclModLockPath) {
				err := os.RemoveAll(vKclModLockPath)
				if err != nil {
					return err
				}
			}
			return nil
		}()

	}
	initOpts := opt.InitOptions{
		Name:     "vPkg_" + uuid.New().String(),
		InitPath: sourcePath,
	}

	kpkg := pkg.NewKclPkg(&initOpts)
	// If the required files are present, proceed with the visitFunc
	return v(&kpkg)
}

// RemoteVisitor is the visitor for visiting a remote package.
type RemoteVisitor struct {
	*PkgVisitor
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
func (rv *RemoteVisitor) Visit(s *downloader.Source, v visitFunc) error {
	if !s.IsRemote() {
		return fmt.Errorf("source is not remote")
	}

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}

	if s.Git != nil {
		tmpDir = filepath.Join(tmpDir, constants.GitScheme)
	}

	credCli, err := rv.kpmcli.GetCredsClient()
	if err != nil {
		return err
	}

	defer os.RemoveAll(tmpDir)
	err = rv.kpmcli.DepDownloader.Download(*downloader.NewDownloadOptions(
		downloader.WithLocalPath(tmpDir),
		downloader.WithSource(*s),
		downloader.WithLogWriter(rv.kpmcli.GetLogWriter()),
		downloader.WithSettings(*rv.kpmcli.GetSettings()),
		downloader.WithCredsClient(credCli),
	))

	if err != nil {
		return err
	}

	kclPkg, err := rv.kpmcli.LoadPkgFromPath(tmpDir)
	if err != nil {
		return err
	}

	return v(kclPkg)
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
func (av *ArchiveVisitor) Visit(s *downloader.Source, v visitFunc) error {

	if !s.IsLocalTarPath() && !s.IsLocalTgzPath() {
		return fmt.Errorf("source is not local tar path")
	}

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}

	defer os.RemoveAll(tmpDir)

	err = av.extractArchiveFromSource(s, tmpDir)

	if err != nil {
		return err
	}

	kclPkg, err := av.kpmcli.LoadPkgFromPath(tmpDir)

	if err != nil {
		return err
	}

	return v(kclPkg)
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
