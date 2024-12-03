package client

import (
	"io"
	"os"
	"path/filepath"

	remoteauth "oras.land/oras-go/v2/registry/remote/auth"

	"kcl-lang.io/kpm/pkg/checker"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/env"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
	"kcl-lang.io/kpm/pkg/visitor"
)

// KpmClient is the client of kpm.
type KpmClient struct {
	// The writer of the log.
	logWriter io.Writer
	// The downloader of the dependencies.
	DepDownloader *downloader.DepDownloader
	// credential store
	credsClient *downloader.CredClient
	// The home path of kpm for global configuration file and kcl package storage path.
	homePath string
	// The settings of kpm loaded from the global configuration file.
	settings settings.Settings
	// The checker to validate dependencies
	ModChecker *checker.ModChecker
	// The flag of whether to check the checksum of the package and update kcl.mod.lock.
	noSumCheck bool
	// The flag of whether to skip the verification of TLS.
	insecureSkipTLSverify bool
}

// NewKpmClient will create a new kpm client with default settings.
func NewKpmClient() (*KpmClient, error) {
	settings := settings.GetSettings()

	if settings.ErrorEvent != (*reporter.KpmEvent)(nil) {
		return nil, settings.ErrorEvent
	}

	homePath, err := env.GetAbsPkgPath()
	if err != nil {
		return nil, err
	}

	ModChecker := checker.NewModChecker(
		checker.WithCheckers(checker.NewIdentChecker(), checker.NewVersionChecker(), checker.NewSumChecker(
			checker.WithSettings(*settings))),
	)

	return &KpmClient{
		logWriter:     os.Stdout,
		settings:      *settings,
		homePath:      homePath,
		ModChecker:    ModChecker,
		DepDownloader: &downloader.DepDownloader{},
	}, nil
}

// SetInsecureSkipTLSverify will set the flag of whether to skip the verification of TLS.
func (c *KpmClient) SetInsecureSkipTLSverify(insecureSkipTLSverify bool) {
	c.insecureSkipTLSverify = insecureSkipTLSverify
}

// SetNoSumCheck will set the 'noSumCheck' flag.
func (c *KpmClient) SetNoSumCheck(noSumCheck bool) {
	c.noSumCheck = noSumCheck
}

// GetCredsClient will return the credential client.
func (c *KpmClient) GetCredsClient() (*downloader.CredClient, error) {
	if c.credsClient == nil {
		credCli, err := downloader.LoadCredentialFile(c.settings.CredentialsFile)
		if err != nil {
			return nil, err
		}
		c.credsClient = credCli
	}
	return c.credsClient, nil
}

// GetCredentials will return the credentials of the host.
func (c *KpmClient) GetCredentials(hostName string) (*remoteauth.Credential, error) {
	credCli, err := c.GetCredsClient()
	if err != nil {
		return nil, err
	}

	creds, err := credCli.Credential(hostName)
	if err != nil {
		return nil, err
	}

	return creds, nil
}

// GetNoSumCheck will return the 'noSumCheck' flag.
func (c *KpmClient) GetNoSumCheck() bool {
	return c.noSumCheck
}

func (c *KpmClient) SetLogWriter(writer io.Writer) {
	c.logWriter = writer
}

func (c *KpmClient) GetLogWriter() io.Writer {
	return c.logWriter
}

// SetHomePath will set the home path of kpm.
func (c *KpmClient) SetHomePath(homePath string) {
	c.homePath = homePath
}

// AcquirePackageCacheLock will acquire the lock of the package cache.
func (c *KpmClient) AcquirePackageCacheLock() error {
	return c.settings.AcquirePackageCacheLock(c.logWriter)
}

// ReleasePackageCacheLock will release the lock of the package cache.
func (c *KpmClient) ReleasePackageCacheLock() error {
	return c.settings.ReleasePackageCacheLock()
}

// GetSettings will return the settings of kpm client.
func (c *KpmClient) GetSettings() *settings.Settings {
	return &c.settings
}

const PKG_NAME_PATTERN = "%s_%s"

// Get the local store path for the dependency.
// 1. in the KCL_PKG_PATH: default is $HOME/.kcl/kpm
// 2. in the vendor subdirectory of the current package.
// 3. the dependency is from the local path.
func (c *KpmClient) getDepStorePath(search_path string, d *pkg.Dependency, isVendor bool) string {
	storePkgName := d.GenPathSuffix()
	if d.IsFromLocal() {
		return d.GetLocalFullPath(search_path)
	} else {
		path := ""
		if isVendor {
			path = filepath.Join(search_path, "vendor", storePkgName)
		} else {
			path = filepath.Join(c.homePath, storePkgName)
		}
		return path
	}
}

// dependencyExists will check whether the dependency exists in the local filesystem.
func (c *KpmClient) dependencyExistsLocal(searchPath string, dep *pkg.Dependency, isVendor bool) (*pkg.Dependency, error) {
	// If the flag '--no_sum_check' is set, skip the checksum check.
	deppath := c.getDepStorePath(searchPath, dep, isVendor)
	if utils.DirExists(deppath) {
		depPkg, err := c.LoadPkgFromPath(deppath)
		if err != nil {
			return nil, err
		}
		dep.FromKclPkg(depPkg)
		// TODO: new local dependency structure will replace this
		// issue: https://github.com/kcl-lang/kpm/issues/384
		dep.FullName = dep.GenDepFullName()

		if dep.GetPackage() != "" {
			dep.LocalFullPath, err = utils.FindPackage(dep.LocalFullPath, dep.GetPackage())
			if err != nil {
				return nil, err
			}
		}
		return dep, nil
	}
	return nil, nil
}

// NewVisitor is a factory function to create a new Visitor.
func newVisitor(source downloader.Source, kpmcli *KpmClient) visitor.Visitor {
	PkgVisitor := &visitor.PkgVisitor{
		Settings:  kpmcli.GetSettings(),
		LogWriter: kpmcli.logWriter,
	}

	if source.IsRemote() {
		return &visitor.RemoteVisitor{
			PkgVisitor:            PkgVisitor,
			Downloader:            kpmcli.DepDownloader,
			InsecureSkipTLSverify: kpmcli.insecureSkipTLSverify,
		}
	} else if source.IsLocalTarPath() || source.IsLocalTgzPath() {
		return visitor.NewArchiveVisitor(PkgVisitor)
	} else if source.IsLocalPath() {
		rootPath, err := source.FindRootPath()
		if err != nil {
			return nil
		}
		kclmodpath := filepath.Join(rootPath, constants.KCL_MOD)
		if utils.DirExists(kclmodpath) {
			return PkgVisitor
		} else {
			return visitor.NewVirtualPkgVisitor(PkgVisitor)
		}
	} else {
		return nil
	}
}
