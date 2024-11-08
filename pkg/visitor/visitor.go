package visitor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/features"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
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

	if s.ModSpec != nil && s.ModSpec.Name != "" {
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
	var err error
	if !s.IsRemote() {
		return fmt.Errorf("source is not remote")
	}

	// For some sources with only the spec, the default registry and repo will be used.
	if s.SpecOnly() {
		s.Oci = &downloader.Oci{
			Reg:  rv.Settings.DefaultOciRegistry(),
			Repo: utils.JoinPath(rv.Settings.DefaultOciRepo(), s.ModSpec.Name),
			Tag:  s.ModSpec.Version,
		}
	}

	var cacheFullPath string
	var modFullPath string

	// Generate the cache path first, for the cache path is needed to get the latest version.
	if ok, err := features.Enabled(features.SupportNewStorage); err == nil && !ok {
		if rv.EnableCache {
			cacheFullPath = s.CachePath(rv.CachePath)
		}
	} else {
		if rv.EnableCache {
			cacheFullPath = s.CachePath(filepath.Join(rv.CachePath, s.Type(), "cache"))
		}
	}

	// If the cache is not enabled,
	// create a temporary directory to get the latest commit of git repo
	if !rv.EnableCache {
		cacheFullPath, err = os.MkdirTemp("", "")
		if err != nil {
			return err
		}

		defer os.RemoveAll(cacheFullPath)
	}

	// 1. Load the credential file.
	credCli, err := downloader.LoadCredentialFile(rv.Settings.CredentialsFile)
	if err != nil {
		return err
	}

	// 2. If the version is not specified, get the latest version.
	// For Oci, the latest tag
	// For Git, the main branch
	if (s.Oci != nil && s.Oci.NoRef()) || (s.Git != nil && s.Git.NoRef()) {
		latest, err := rv.Downloader.LatestVersion(downloader.NewDownloadOptions(
			downloader.WithSource(*s),
			downloader.WithLogWriter(rv.LogWriter),
			downloader.WithSettings(*rv.Settings),
			downloader.WithCredsClient(credCli),
			downloader.WithInsecureSkipTLSverify(rv.InsecureSkipTLSverify),
			downloader.WithCachePath(cacheFullPath),
			downloader.WithEnableCache(rv.EnableCache),
		))

		if err != nil {
			return err
		}

		reporter.ReportMsgTo(
			fmt.Sprintf("the lastest version '%s' will be downloaded", latest),
			rv.LogWriter,
		)

		if !s.ModSpec.IsNil() && s.ModSpec.Version == "" {
			s.ModSpec.Version = latest
		}

		if s.Oci != nil {
			s.Oci.Tag = latest
		}
		if s.Git != nil {
			s.Git.Commit = latest
		}
	}

	// Generate the local path for the remote package after the version is specified.
	if ok, err := features.Enabled(features.SupportNewStorage); err == nil && !ok {
		// update the local module path with the latest version.
		if len(rv.VisitedSpace) != 0 {
			modFullPath = s.LocalPath(rv.VisitedSpace)
		}
		// update the cache path with the latest version.
		if rv.EnableCache {
			cacheFullPath = s.CachePath(rv.CachePath)
		}
	} else {
		// update the local module path with the latest version.
		if len(rv.VisitedSpace) != 0 {
			modFullPath = s.LocalPath(filepath.Join(rv.VisitedSpace, s.Type(), "src"))
		}
		// update the cache path with the latest version.
		if rv.EnableCache {
			cacheFullPath = s.CachePath(filepath.Join(rv.CachePath, s.Type(), "cache"))
		}
	}

	if len(rv.VisitedSpace) == 0 {
		tmpDir, err := os.MkdirTemp("", "")
		if err != nil {
			return err
		}

		modFullPath = tmpDir
		defer os.RemoveAll(tmpDir)
	}

	if !utils.DirExists(modFullPath) {
		err := os.MkdirAll(modFullPath, 0755)
		if err != nil {
			return err
		}
	}

	credCli, err = downloader.LoadCredentialFile(rv.Settings.CredentialsFile)
	if err != nil {
		return err
	}

	err = rv.Downloader.Download(downloader.NewDownloadOptions(
		downloader.WithLocalPath(modFullPath),
		downloader.WithSource(*s),
		downloader.WithLogWriter(rv.LogWriter),
		downloader.WithSettings(*rv.Settings),
		downloader.WithCredsClient(credCli),
		downloader.WithCachePath(cacheFullPath),
		downloader.WithEnableCache(rv.EnableCache),
		downloader.WithInsecureSkipTLSverify(rv.InsecureSkipTLSverify),
	))

	if err != nil {
		return err
	}
	if !s.ModSpec.IsNil() {
		modFullPath, err = utils.FindPackage(modFullPath, s.ModSpec.Name)
		if err != nil {
			return err
		}
	}

	kclPkg, err := pkg.LoadKclPkgWithOpts(
		pkg.WithPath(modFullPath),
		pkg.WithSettings(rv.Settings),
	)
	if err != nil {
		return err
	}

	if !s.ModSpec.IsNil() {
		if s.ModSpec.Version == "" {
			s.ModSpec.Version = kclPkg.ModFile.Pkg.Version
		} else if kclPkg.ModFile.Pkg.Version != s.ModSpec.Version {
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
