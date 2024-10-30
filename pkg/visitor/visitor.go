package visitor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

type visitFunc func(pkg *pkg.KclPkg) error

// Visitor is the interface for visiting a package which is a local path, a remote git/oci path, or a local tar path.
type Visitor interface {
	Visit(s *downloader.Source, v visitFunc) error
}

// PkgVisitor is the visitor for visiting a local package.
type PkgVisitor struct {
	Settings  *settings.Settings
	LogWriter io.Writer
}

// Visit visits a local package.
func (pv *PkgVisitor) Visit(s *downloader.Source, v visitFunc) error {
	if !s.IsLocalPath() {
		return fmt.Errorf("source is not local")
	}
	// Find the root path of the source.
	// There must be a kcl.mod file in the root path.
	modPath, err := s.FindRootPath()
	if err != nil {
		return err
	}

	if !s.ModSpec.IsNil() {
		modPath, err = utils.FindPackage(modPath, s.ModSpec.Name)
		if err != nil {
			return err
		}
	}

	kclPkg, err := pkg.LoadKclPkgWithOpts(
		pkg.WithPath(modPath),
		pkg.WithSettings(pv.Settings),
	)
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
	EnableCache           bool
	CachePath             string
	VisitedSpace          string
	Downloader            downloader.Downloader
	InsecureSkipTLSverify bool
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

	var modPath string
	var err error
	if len(rv.VisitedSpace) != 0 {
		modPath = filepath.Join(rv.VisitedSpace, s.LocalPath())
	} else {
		tmpDir, err := os.MkdirTemp("", "")
		if err != nil {
			return err
		}

		modPath = tmpDir
		defer os.RemoveAll(tmpDir)
	}

	if !utils.DirExists(modPath) {
		err := os.MkdirAll(modPath, 0755)
		if err != nil {
			return err
		}
	}

	credCli, err := downloader.LoadCredentialFile(rv.Settings.CredentialsFile)
	if err != nil {
		return err
	}

	err = rv.Downloader.Download(*downloader.NewDownloadOptions(
		downloader.WithLocalPath(modPath),
		downloader.WithSource(*s),
		downloader.WithLogWriter(rv.LogWriter),
		downloader.WithSettings(*rv.Settings),
		downloader.WithCredsClient(credCli),
		downloader.WithCachePath(rv.CachePath),
		downloader.WithEnableCache(rv.EnableCache),
		downloader.WithInsecureSkipTLSverify(rv.InsecureSkipTLSverify),
	))

	if err != nil {
		return err
	}
	if !s.ModSpec.IsNil() {
		modPath, err = utils.FindPackage(modPath, s.ModSpec.Name)
		if err != nil {
			return err
		}
	}

	kclPkg, err := pkg.LoadKclPkgWithOpts(
		pkg.WithPath(modPath),
		pkg.WithSettings(rv.Settings),
	)
	if err != nil {
		return err
	}

	if !s.ModSpec.IsNil() {
		if kclPkg.ModFile.Pkg.Version != s.ModSpec.Version {
			return fmt.Errorf(
				"version mismatch: %s != %s, version %s not found",
				kclPkg.ModFile.Pkg.Version, s.ModSpec.Version, s.ModSpec.Version,
			)
		}
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

	kclPkg, err := pkg.LoadKclPkgWithOpts(
		pkg.WithPath(tmpDir),
		pkg.WithSettings(av.Settings),
	)

	if err != nil {
		return err
	}

	return v(kclPkg)
}
