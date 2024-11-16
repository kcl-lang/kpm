package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/dominikbraun/graph"
	"github.com/elliotchance/orderedmap/v2"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/mod/module"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"oras.land/oras-go/pkg/auth"
	"oras.land/oras-go/v2"
	remoteauth "oras.land/oras-go/v2/registry/remote/auth"

	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/features"
	"kcl-lang.io/kpm/pkg/git"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/runner"
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
	// The flag of whether to check the checksum of the package and update kcl.mod.lock.
	noSumCheck bool
	// The flag of whether to skip the verification of TLS.
	insecureSkipTLSverify bool
	// HttpClient is used to make HTTP requests.
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTP client configured with proxy settings.
func NewHTTPClient(cfg *settings.Config) (*http.Client, error) {
	proxyFunc := http.ProxyFromEnvironment // Use environment variables by default

	if cfg.Proxy.HTTP != "" { // Override with custom proxy settings if provided
		proxyURL, err := url.Parse(cfg.Proxy.HTTP)
		if err != nil {
			return nil, err // Handle the error if URL parsing fails
		}
		proxyFunc = http.ProxyURL(proxyURL)
	}

	transport := &http.Transport{
		Proxy: proxyFunc,
	}

	return &http.Client{Transport: transport}, nil
}

// NewKpmClient creates a new kpm client with default settings.
func NewKpmClient() (*KpmClient, error) {
	// Load settings using the appropriate method that returns both settings and error
	cfg, err := settings.LoadConfig() // Assuming LoadConfig is a method that returns (*Config, error)
	if err != nil {
		return nil, err // Handle configuration load error
	}

	// Create an HTTP client with the loaded settings
	httpClient, err := NewHTTPClient(cfg)
	if err != nil {
		return nil, err // Handle HTTP client creation error
	}

	// Check if there is an error event stored in settings
	if cfg.ErrorEvent != nil { // Assuming ErrorEvent is a pointer to a KpmEvent or similar error handling
		return nil, cfg.ErrorEvent
	}

	// Retrieve the absolute package path from environment
	homePath, err := env.GetAbsPkgPath()
	if err != nil {
		return nil, err
	}

	// Construct the KpmClient instance with loaded settings and components
	return &KpmClient{
		logWriter:     os.Stdout,
		settings:      *cfg,
		httpClient:    httpClient, // Store the HTTP client in the KpmClient structure
		homePath:      homePath,
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

func (c *KpmClient) LoadPkgFromPath(path string) (*pkg.KclPkg, error) {
	return pkg.LoadKclPkgWithOpts(
		pkg.WithPath(path),
		pkg.WithSettings(&c.settings),
	)
}

func (c *KpmClient) LoadModFile(path string) (*pkg.ModFile, error) {
	return pkg.LoadAndFillModFileWithOpts(
		pkg.WithPath(path),
		pkg.WithSettings(&c.settings),
	)
}

// Load the kcl.mod.lock and acquire the checksum of the dependencies from OCI registry.
func (c *KpmClient) LoadLockDeps(pkgPath string) (*pkg.Dependencies, error) {
	deps, err := pkg.LoadLockDeps(pkgPath)
	if err != nil {
		return nil, err
	}

	return deps, nil
}

// AcquireDepSum will acquire the checksum of the dependency from the OCI registry.
func (c *KpmClient) AcquireDepSum(dep pkg.Dependency) (string, error) {
	// Only the dependencies from the OCI need can be checked.
	if dep.Source.Oci != nil {
		if len(dep.Source.Oci.Reg) == 0 {
			dep.Source.Oci.Reg = c.GetSettings().DefaultOciRegistry()
		}

		if len(dep.Source.Oci.Repo) == 0 {
			urlpath := utils.JoinPath(c.GetSettings().DefaultOciRepo(), dep.Name)
			dep.Source.Oci.Repo = urlpath
		}
		// Fetch the metadata of the OCI manifest.
		manifest := ocispec.Manifest{}
		jsonDesc, err := c.FetchOciManifestIntoJsonStr(opt.OciFetchOptions{
			FetchBytesOptions: oras.DefaultFetchBytesOptions,
			OciOptions: opt.OciOptions{
				Reg:  dep.Source.Oci.Reg,
				Repo: dep.Source.Oci.Repo,
				Tag:  dep.Source.Oci.Tag,
			},
		})

		if err != nil {
			return "", reporter.NewErrorEvent(reporter.FailedFetchOciManifest, err, fmt.Sprintf("failed to fetch the manifest of '%s'", dep.Name))
		}

		err = json.Unmarshal([]byte(jsonDesc), &manifest)
		if err != nil {
			return "", err
		}

		// Check the dependency checksum.
		if value, ok := manifest.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_SUM]; ok {
			return value, nil
		}
	}

	return "", nil
}

// ResolveDepsIntoMap will calculate the map of kcl package name and local storage path of the external packages.
func (c *KpmClient) ResolveDepsIntoMap(kclPkg *pkg.KclPkg) (map[string]string, error) {
	err := c.ResolvePkgDepsMetadata(kclPkg, true)
	if err != nil {
		return nil, err
	}

	depMetadatas, err := kclPkg.GetDepsMetadata()
	if err != nil {
		return nil, err
	}
	var pkgMap map[string]string = make(map[string]string)
	for _, d := range depMetadatas.Deps {
		pkgMap[d.GetAliasName()] = d.GetLocalFullPath(kclPkg.HomePath)
	}

	return pkgMap, nil
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

// ResolveDepsMetadata will calculate the local storage path of the external package,
// and check whether the package exists locally.
// If the package does not exist, it will re-download to the local.
// Since redownloads are not triggered if local dependencies exists,
// indirect dependencies are also synchronized to the lock file by `lockDeps`.
func (c *KpmClient) ResolvePkgDepsMetadata(kclPkg *pkg.KclPkg, update bool) error {
	if kclPkg.IsVendorMode() {
		// In the vendor mode, the search path is the vendor subdirectory of the current package.
		err := c.VendorDeps(kclPkg)
		if err != nil {
			return err
		}
	} else {
		// In the non-vendor mode, the search path is the KCL_PKG_PATH.
		err := c.resolvePkgDeps(kclPkg, &kclPkg.Dependencies, update)
		if err != nil {
			return err
		}

	}
	return nil
}

func (c *KpmClient) resolvePkgDeps(kclPkg *pkg.KclPkg, lockDeps *pkg.Dependencies, update bool) error {
	var searchPath string
	kclPkg.NoSumCheck = c.noSumCheck

	// If under the mode of '--no_sum_check', the checksum of the package will not be checked.
	// There is no kcl.mod.lock, and the dependencies in kcl.mod and kcl.mod.lock do not need to be aligned.
	if !c.noSumCheck {
		// If not under the mode of '--no_sum_check',
		// all the dependencies in kcl.mod.lock are the dependencies of the current package.
		//
		// alian the dependencies between kcl.mod and kcl.mod.lock
		// clean the dependencies in kcl.mod.lock which not in kcl.mod
		// clean the dependencies in kcl.mod.lock and kcl.mod which have different version
		for _, name := range kclPkg.Dependencies.Deps.Keys() {
			dep, ok := kclPkg.Dependencies.Deps.Get(name)
			if !ok {
				break
			}
			modDep, ok := kclPkg.ModFile.Dependencies.Deps.Get(name)
			if !ok || !dep.Equals(modDep) {
				kclPkg.Dependencies.Deps.Delete(name)
			}
		}
		// add the dependencies in kcl.mod which not in kcl.mod.lock
		for _, name := range kclPkg.ModFile.Dependencies.Deps.Keys() {
			d, ok := kclPkg.ModFile.Dependencies.Deps.Get(name)
			if !ok {
				break
			}
			if _, ok := kclPkg.Dependencies.Deps.Get(name); !ok {
				kclPkg.Dependencies.Deps.Set(name, d)
			}
		}
	} else {
		// If under the mode of '--no_sum_check', the checksum of the package will not be checked.
		// All the dependencies in kcl.mod are the dependencies of the current package.
		kclPkg.Dependencies.Deps = kclPkg.ModFile.Dependencies.Deps
	}

	for _, name := range kclPkg.Dependencies.Deps.Keys() {
		d, ok := kclPkg.Dependencies.Deps.Get(name)
		if !ok {
			break
		}
		searchPath = c.getDepStorePath(kclPkg.HomePath, &d, kclPkg.IsVendorMode())
		depPath := searchPath
		// if the dependency is not exist
		if !utils.DirExists(searchPath) {
			if d.IsFromLocal() {
				// If the dependency is from the local path, and it does not exist locally, raise an error
				return reporter.NewErrorEvent(reporter.DependencyNotFound, fmt.Errorf("dependency '%s' not found in '%s'", d.Name, searchPath))
			} else {
				// redownload the dependency to the local path.
				if update {
					// re-vendor it.
					if kclPkg.IsVendorMode() {
						err := c.vendorDeps(kclPkg, kclPkg.LocalVendorPath())
						if err != nil {
							return err
						}
					} else {
						// re-download it.
						err := c.AddDepToPkg(kclPkg, &d)
						if err != nil {
							return err
						}

						depPath = c.getDepStorePath(kclPkg.HomePath, &d, kclPkg.IsVendorMode())
					}
				} else {
					continue
				}
			}
		}

		if d.GetPackage() != "" {
			depPath, _ = utils.FindPackage(depPath, d.GetPackage())
		}

		// If the dependency exists locally, load the dependency package.
		depPkg, err := c.LoadPkgFromPath(depPath)
		if err != nil {
			return reporter.NewErrorEvent(
				reporter.DependencyNotFound,
				fmt.Errorf("dependency '%s' not found in '%s'", d.Name, searchPath),
				// todo: add command to clean the package cache
			)
		}
		d.FromKclPkg(depPkg)
		err = c.resolvePkgDeps(depPkg, lockDeps, update)
		if err != nil {
			return err
		}
		if d.Source.Git != nil && d.Source.Git.GetPackage() != "" {
			if d.Source.Git != nil && d.Source.Git.GetPackage() != "" {
				name := utils.ParseRepoNameFromGitUrl(d.Source.Git.Url)
				if len(d.Source.Git.Tag) != 0 {
					d.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, d.Source.Git.Tag)
				} else if len(d.Source.Git.Commit) != 0 {
					d.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, d.Source.Git.Commit)
				} else {
					d.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, d.Source.Git.Branch)
				}
			}
		}
		kclPkg.Dependencies.Deps.Set(name, d)
		lockDeps.Deps.Set(name, d)
	}

	// Generate file kcl.mod.lock.
	if kclPkg.ModFile.Dependencies.Deps.Len() > 0 && !kclPkg.NoSumCheck || !update {
		err := kclPkg.LockDepsVersion()
		if err != nil {
			return err
		}
	}

	return nil
}

func GetReleasesFromSource(sourceType, uri string) ([]string, error) {
	var releases []string
	var err error

	switch sourceType {
	case pkg.GIT:
		releases, err = git.GetAllGithubReleases(uri)
	case pkg.OCI:
		releases, err = oci.GetAllImageTags(uri)
	}
	if err != nil {
		return nil, err
	}

	return releases, nil
}

// UpdateDeps will update the dependencies.
func (c *KpmClient) UpdateDeps(kclPkg *pkg.KclPkg) error {
	_, err := c.ResolveDepsMetadataInJsonStr(kclPkg, true)
	if err != nil {
		return err
	}

	if ok, err := features.Enabled(features.SupportMVS); err != nil && ok {
		_, err = c.Update(
			WithUpdatedKclPkg(kclPkg),
		)
		if err != nil {
			return err
		}
	} else {
		// update kcl.mod
		err = kclPkg.UpdateModAndLockFile()
		if err != nil {
			return err
		}
	}

	return nil
}

// ResolveDepsMetadataInJsonStr will calculate the local storage path of the external package,
// and check whether the package exists locally. If the package does not exist, it will re-download to the local.
// Finally, the calculated metadata of the dependent packages is serialized into a json string and returned.
func (c *KpmClient) ResolveDepsMetadataInJsonStr(kclPkg *pkg.KclPkg, update bool) (string, error) {
	// 1. Calculate the dependency path, check whether the dependency exists
	// and re-download the dependency that does not exist.
	err := c.ResolvePkgDepsMetadata(kclPkg, update)
	if err != nil {
		return "", err
	}

	// 2. Serialize to JSON
	depMetadatas, err := kclPkg.GetDepsMetadata()
	if err != nil {
		return "", err
	}
	jsonData, err := json.Marshal(&depMetadatas)
	if err != nil {
		return "", reporter.NewErrorEvent(reporter.Bug, err, "internal bug: failed to marshal the dependencies into json")
	}

	return string(jsonData), nil
}

// Compile will call kcl compiler to compile the current kcl package and its dependent packages.
func (c *KpmClient) Compile(kclPkg *pkg.KclPkg, kclvmCompiler *runner.Compiler) (*kcl.KCLResultList, error) {
	pkgMap, err := c.ResolveDepsIntoMap(kclPkg)
	if err != nil {
		return nil, err
	}

	// Fill the dependency path.
	for dName, dPath := range pkgMap {
		if !filepath.IsAbs(dPath) {
			dPath = filepath.Join(c.homePath, dPath)
		}
		kclvmCompiler.AddDepPath(dName, dPath)
	}

	return kclvmCompiler.Run()
}

// createIfNotExist will create a file if it does not exist.
func (c *KpmClient) createIfNotExist(filepath string, storeFunc func() error) error {
	reporter.ReportMsgTo(fmt.Sprintf("creating new :%s", filepath), c.GetLogWriter())
	err := utils.CreateFileIfNotExist(
		filepath,
		storeFunc,
	)
	if err != nil {
		if errEvent, ok := err.(*reporter.KpmEvent); ok {
			if errEvent.Type() != reporter.FileExists {
				return err
			} else {
				reporter.ReportMsgTo(fmt.Sprintf("'%s' already exists", filepath), c.GetLogWriter())
			}
		} else {
			return err
		}
	}

	return nil
}

// InitEmptyPkg will initialize an empty kcl package.
func (c *KpmClient) InitEmptyPkg(kclPkg *pkg.KclPkg) error {
	err := c.createIfNotExist(kclPkg.ModFile.GetModFilePath(), kclPkg.ModFile.StoreModFile)
	if err != nil {
		return err
	}

	err = c.createIfNotExist(kclPkg.ModFile.GetModLockFilePath(), kclPkg.LockDepsVersion)
	if err != nil {
		return err
	}

	err = c.createIfNotExist(filepath.Join(kclPkg.ModFile.HomePath, constants.DEFAULT_KCL_FILE_NAME), kclPkg.CreateDefaultMain)
	if err != nil {
		return err
	}

	return nil
}

// AddDepWithOpts will add a dependency to the current kcl package.
func (c *KpmClient) AddDepWithOpts(kclPkg *pkg.KclPkg, opt *opt.AddOptions) (*pkg.KclPkg, error) {
	c.noSumCheck = opt.NoSumCheck
	kclPkg.NoSumCheck = opt.NoSumCheck

	// 1. get the name and version of the repository/package from the input arguments.
	d, err := pkg.ParseOpt(&opt.RegistryOpts)
	if err != nil {
		return nil, err
	}

	// Backup the dependency used in kcl.mod
	if opt.RegistryOpts.Registry != nil {
		kclPkg.BackupDepUI(d.Name, &pkg.Dependency{
			Name:    d.Name,
			Version: d.Version,
			Source: downloader.Source{
				ModSpec: &downloader.ModSpec{
					Name:    d.Name,
					Version: d.Version,
				},
			},
		})
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("adding dependency '%s'", d.Name),
		c.logWriter,
	)

	// 2. download the dependency to the local path.
	err = c.AddDepToPkg(kclPkg, d)
	if err != nil {
		return nil, err
	}

	// 3. update the kcl.mod and kcl.mod.lock.
	if opt.NewPkgName != "" {
		// update the kcl.mod with NewPkgName
		tempDeps, ok := kclPkg.ModFile.Dependencies.Deps.Get(d.Name)
		if !ok {
			return nil, fmt.Errorf("dependency '%s' not found in 'kcl.mod'", d.Name)
		}
		tempDeps.Name = opt.NewPkgName
		kclPkg.ModFile.Dependencies.Deps.Set(d.Name, tempDeps)

		// update the kcl.mod.lock with NewPkgName
		tempDeps, ok = kclPkg.Dependencies.Deps.Get(d.Name)
		if !ok {
			return nil, fmt.Errorf("dependency '%s' not found in 'kcl.mod.lock'", d.Name)
		}
		tempDeps.Name = opt.NewPkgName
		tempDeps.FullName = opt.NewPkgName + "_" + tempDeps.Version
		kclPkg.Dependencies.Deps.Set(d.Name, tempDeps)

		// update the key of kclPkg.Dependencies.Deps from d.Name to opt.NewPkgName
		kclPkg.Dependencies.Deps.Set(opt.NewPkgName, kclPkg.Dependencies.Deps.GetOrDefault(d.Name, pkg.TestPkgDependency))
		kclPkg.Dependencies.Deps.Delete(d.Name)
	}

	if ok, err := features.Enabled(features.SupportMVS); err != nil && ok {
		// After adding the new dependency,
		// Iterate through all the dependencies and select the version by mvs
		_, err = c.Update(
			WithUpdatedKclPkg(kclPkg),
		)

		if err != nil {
			return nil, err
		}
	} else {
		err = kclPkg.UpdateModAndLockFile()
		if err != nil {
			return nil, err
		}
	}

	succeedMsgInfo := d.Name
	if len(d.Version) != 0 {
		succeedMsgInfo = fmt.Sprintf("%s:%s", d.Name, d.Version)
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("add dependency '%s' successfully", succeedMsgInfo),
		c.logWriter,
	)
	return kclPkg, nil
}

// AddDepToPkg will add a dependency to the kcl package.
func (c *KpmClient) AddDepToPkg(kclPkg *pkg.KclPkg, d *pkg.Dependency) error {

	// If the dependency is from the local path, do nothing.
	if d.IsFromLocal() {
		kclPkg.ModFile.Dependencies.Deps.Set(d.Name, *d)
		kclPkg.Dependencies.Deps.Set(d.Name, *d)
		return nil
	}

	// Some field will be empty when the dependency is add from CLI.
	// For avoiding re-download the dependency, just complete part of the fields not all of them.
	if !kclPkg.ModFile.Dependencies.Deps.GetOrDefault(d.Name, pkg.TestPkgDependency).Equals(*d) {
		// the dep passed on the cli is different from the kcl.mod.
		kclPkg.ModFile.Dependencies.Deps.Set(d.Name, *d)
	}

	// download all the dependencies.
	_, _, err := c.InitGraphAndDownloadDeps(kclPkg)

	if err != nil {
		return err
	}

	return err
}

// PackagePkg will package the current kcl package into a "*.tar" file in under the package path.
func (c *KpmClient) PackagePkg(kclPkg *pkg.KclPkg, vendorMode bool) (string, error) {
	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return "", err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != (*reporter.KpmEvent)(nil) {
		return "", err
	}

	err = c.Package(kclPkg, kclPkg.DefaultTarPath(), vendorMode)

	if err != nil {
		reporter.ExitWithReport("failed to package pkg " + kclPkg.GetPkgName() + ".")
		return "", err
	}
	return kclPkg.DefaultTarPath(), nil
}

// Package will package the current kcl package into a "*.tar" file into 'tarPath'.
func (c *KpmClient) Package(kclPkg *pkg.KclPkg, tarPath string, vendorMode bool) error {
	// Vendor all the dependencies into the current kcl package.
	if vendorMode {
		err := c.VendorDeps(kclPkg)
		if err != nil {
			return reporter.NewErrorEvent(reporter.FailedVendor, err, "failed to vendor dependencies")
		}
	}

	// Tar the current kcl package into a "*.tar" file.
	err := utils.TarDir(kclPkg.HomePath, tarPath, kclPkg.GetPkgInclude(), kclPkg.GetPkgExclude())
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedPackage, err, "failed to package the kcl module")
	}
	return nil
}

// FillDepInfo will fill registry information for a dependency.
func (c *KpmClient) FillDepInfo(dep *pkg.Dependency, homepath string) error {
	// Homepath for a dependency is the homepath of the kcl package.
	if dep.Source.Local != nil {
		dep.LocalFullPath = dep.Source.Local.Path
		return nil
	}
	if dep.Source.Oci != nil {
		if len(dep.Source.Oci.Reg) == 0 {
			dep.Source.Oci.Reg = c.GetSettings().DefaultOciRegistry()
		}

		if len(dep.Source.Oci.Repo) == 0 {
			urlpath := utils.JoinPath(c.GetSettings().DefaultOciRepo(), dep.Name)
			dep.Source.Oci.Repo = urlpath
		}
	}
	if dep.Source.Git != nil && dep.Source.Git.GetPackage() != "" {
		name := utils.ParseRepoNameFromGitUrl(dep.Source.Git.Url)
		if len(dep.Source.Git.Tag) != 0 {
			dep.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, dep.Source.Git.Tag)
		} else if len(dep.Source.Git.Commit) != 0 {
			dep.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, dep.Source.Git.Commit)
		} else {
			dep.FullName = fmt.Sprintf(PKG_NAME_PATTERN, name, dep.Source.Git.Branch)
		}
	}
	return nil
}

// FillDependenciesInfo will fill registry information for all dependencies in a kcl.mod.
func (c *KpmClient) FillDependenciesInfo(modFile *pkg.ModFile) error {
	for _, k := range modFile.Deps.Keys() {
		v, ok := modFile.Deps.Get(k)
		if !ok {
			break
		}
		err := c.FillDepInfo(&v, modFile.HomePath)
		if err != nil {
			return err
		}
		modFile.Deps.Set(k, v)
	}
	return nil
}

// AcquireTheLatestOciVersion will acquire the latest version of the OCI reference.
func (c *KpmClient) AcquireTheLatestOciVersion(ociSource downloader.Oci) (string, error) {
	repoPath := utils.JoinPath(ociSource.Reg, ociSource.Repo)
	cred, err := c.GetCredentials(ociSource.Reg)
	if err != nil {
		return "", err
	}

	ociClient, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
		oci.WithInsecureSkipTLSverify(c.insecureSkipTLSverify),
	)

	if err != nil {
		return "", err
	}

	return ociClient.TheLatestTag()
}

func (c *KpmClient) downloadPkg(options ...downloader.Option) (*pkg.KclPkg, error) {
	opts := downloader.DownloadOptions{}
	for _, option := range options {
		option(&opts)
	}

	localPath := opts.LocalPath
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, err
	}
	tmpDir = filepath.Join(tmpDir, constants.GitScheme)
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	credCli, err := c.GetCredsClient()
	if err != nil {
		return nil, err
	}

	err = c.DepDownloader.Download(downloader.NewDownloadOptions(
		downloader.WithLocalPath(tmpDir),
		downloader.WithSource(opts.Source),
		downloader.WithLogWriter(c.GetLogWriter()),
		downloader.WithSettings(*c.GetSettings()),
		downloader.WithCredsClient(credCli),
		downloader.WithInsecureSkipTLSverify(opts.InsecureSkipTLSverify),
	))

	if err != nil {
		return nil, err
	}

	if utils.DirExists(localPath) {
		err := os.RemoveAll(localPath)
		if err != nil {
			return nil, err
		}
	}

	destDir := filepath.Dir(localPath)
	if !utils.DirExists(destDir) {
		err = os.MkdirAll(destDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	err = utils.MoveFile(tmpDir, localPath)
	if err != nil {
		return nil, err
	}

	localPath, err = filepath.Abs(localPath)
	if err != nil {
		return nil, err
	}

	pkg, err := c.LoadPkgFromPath(localPath)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

// Download will download the dependency to the local path.
func (c *KpmClient) Download(dep *pkg.Dependency, homePath, localPath string) (*pkg.Dependency, error) {
	if dep.Source.Git != nil {
		err := c.DepDownloader.Download(downloader.NewDownloadOptions(
			downloader.WithLocalPath(localPath),
			downloader.WithSource(dep.Source),
			downloader.WithLogWriter(c.logWriter),
			downloader.WithSettings(c.settings),
		))
		if err != nil {
			return nil, err
		}

		dep.FullName = dep.GenDepFullName()

		if dep.GetPackage() != "" {
			localFullPath, err := utils.FindPackage(localPath, dep.GetPackage())
			if err != nil {
				return nil, err
			}
			dep.LocalFullPath = localFullPath
			dep.Name = dep.GetPackage()
		} else {
			dep.LocalFullPath = localPath
		}

		modFile, err := c.LoadModFile(dep.LocalFullPath)
		if err != nil {
			return nil, err
		}
		dep.Version = modFile.Pkg.Version
	}

	if dep.Source.Oci != nil {
		var ociSource *downloader.Oci
		if dep.Source.Oci != nil {
			ociSource = dep.Source.Oci
		}
		// Select the latest tag, if the tag, the user inputed, is empty.
		if ociSource.Tag == "" || ociSource.Tag == constants.LATEST {
			latestTag, err := c.AcquireTheLatestOciVersion(*ociSource)
			if err != nil {
				return nil, err
			}
			ociSource.Tag = latestTag
			if dep.Source.ModSpec != nil {
				dep.Source.ModSpec.Version = latestTag
			}

			// Complete some information that the local three dependencies depend on.
			// The invalid path such as '$HOME/.kcl/kpm/k8s_' is placed because the version field is missing.
			dep.Version = latestTag
			dep.FullName = dep.GenDepFullName()
			dep.LocalFullPath = filepath.Join(filepath.Dir(localPath), dep.FullName)
			localPath = dep.LocalFullPath

			if utils.DirExists(dep.LocalFullPath) {
				dpkg, err := c.LoadPkgFromPath(localPath)
				if err != nil {
					// If the package is invalid, delete it and re-download it.
					err := os.RemoveAll(dep.LocalFullPath)
					if err != nil {
						return nil, err
					}
				} else {
					dep.FromKclPkg(dpkg)
					return dep, nil
				}
			}
		}

		credCli, err := c.GetCredsClient()
		if err != nil {
			return nil, err
		}
		err = c.DepDownloader.Download(downloader.NewDownloadOptions(
			downloader.WithLocalPath(localPath),
			downloader.WithSource(dep.Source),
			downloader.WithLogWriter(c.logWriter),
			downloader.WithSettings(c.settings),
			downloader.WithCredsClient(credCli),
			downloader.WithInsecureSkipTLSverify(c.insecureSkipTLSverify),
		))
		if err != nil {
			return nil, err
		}

		// load the package from the local path.
		dpkg, err := c.LoadPkgFromPath(localPath)
		if err != nil {
			return nil, err
		}

		dep.FromKclPkg(dpkg)
		dep.Sum, err = c.AcquireDepSum(*dep)
		if err != nil {
			return nil, err
		}
		if dep.Sum == "" {
			dep.Sum, err = utils.HashDir(localPath)
			if err != nil {
				return nil, err
			}
		}

		if dep.LocalFullPath == "" {
			dep.LocalFullPath = localPath
		}

		if localPath != dep.LocalFullPath {
			err = os.Rename(localPath, dep.LocalFullPath)
			if err != nil {
				return nil, err
			}
		}

		// Creating symbolic links in a global cache is not an optimal solution.
		// This allows kclvm to locate the package by default.
		// This feature is unstable and will be removed soon.
		// err = createDepRef(dep.LocalFullPath, filepath.Join(filepath.Dir(localPath), dep.Name))
		// if err != nil {
		//     return nil, err
		// }
	}

	if dep.Source.Local != nil {
		kpkg, err := pkg.FindFirstKclPkgFrom(c.getDepStorePath(homePath, dep, false))
		if err != nil {
			return nil, err
		}
		dep.FromKclPkg(kpkg)
	}

	return dep, nil
}

// DownloadFromGit will download the dependency from the git repository.
func (c *KpmClient) DownloadFromGit(dep *downloader.Git, localPath string) (string, error) {
	var msg string
	if len(dep.Tag) != 0 {
		msg = fmt.Sprintf("with tag '%s'", dep.Tag)
	}

	if len(dep.Commit) != 0 {
		msg = fmt.Sprintf("with commit '%s'", dep.Commit)
	}

	if len(dep.Branch) != 0 {
		msg = fmt.Sprintf("with branch '%s'", dep.Branch)
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("cloning '%s' %s", dep.Url, msg),
		c.logWriter,
	)

	_, err := git.CloneWithOpts(
		git.WithCommit(dep.Commit),
		git.WithTag(dep.Tag),
		git.WithRepoURL(dep.Url),
		git.WithLocalPath(localPath),
		git.WithWriter(c.logWriter),
	)

	if err != nil {
		return localPath, reporter.NewErrorEvent(
			reporter.FailedCloneFromGit,
			err,
			fmt.Sprintf("failed to clone from '%s' into '%s'.", dep.Url, localPath),
		)
	}

	return localPath, err
}

func (c *KpmClient) ParseKclModFile(kclPkg *pkg.KclPkg) (map[string]map[string]string, error) {
	// Get path to kcl.mod file
	modFilePath := kclPkg.ModFile.GetModFilePath()

	// Read the content of the kcl.mod file
	modFileBytes, err := os.ReadFile(modFilePath)
	if err != nil {
		return nil, err
	}

	// Normalize line endings for Windows systems
	modFileContent := strings.ReplaceAll(string(modFileBytes), "\r\n", "\n")

	// Parse the TOML content
	var modFileData map[string]interface{}
	if err := toml.Unmarshal([]byte(modFileContent), &modFileData); err != nil {
		return nil, err
	}

	// Extract dependency information
	dependencies := make(map[string]map[string]string)
	if deps, ok := modFileData["dependencies"].(map[string]interface{}); ok {
		for dep, details := range deps {
			dependency := make(map[string]string)
			switch d := details.(type) {
			case string:
				// For simple version strings
				dependency["version"] = d
			case map[string]interface{}:
				// For dependencies with attributes
				for key, value := range d {
					dependency[key] = fmt.Sprintf("%v", value)
				}
			default:
				return nil, fmt.Errorf("unsupported dependency format")
			}
			dependencies[dep] = dependency
		}
	}

	return dependencies, nil
}

// LoadPkgFromOci will download the kcl package from the oci repository and return an `KclPkg`.
func (c *KpmClient) DownloadPkgFromOci(dep *downloader.Oci, localPath string) (*pkg.KclPkg, error) {
	repoPath := utils.JoinPath(dep.Reg, dep.Repo)
	cred, err := c.GetCredentials(dep.Reg)
	if err != nil {
		return nil, err
	}

	ociClient, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
	)

	if err != nil {
		return nil, err
	}

	ociClient.SetLogWriter(c.logWriter)
	// Select the latest tag, if the tag, the user inputed, is empty.
	var tagSelected string
	if len(dep.Tag) == 0 {
		tagSelected, err = ociClient.TheLatestTag()
		if err != nil {
			return nil, err
		}

		reporter.ReportMsgTo(
			fmt.Sprintf("the lastest version '%s' will be added", tagSelected),
			c.logWriter,
		)

		dep.Tag = tagSelected
		localPath = localPath + dep.Tag
	} else {
		tagSelected = dep.Tag
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("downloading '%s:%s' from '%s/%s:%s'", dep.Repo, tagSelected, dep.Reg, dep.Repo, tagSelected),
		c.logWriter,
	)

	// Pull the package with the tag.
	err = ociClient.Pull(localPath, tagSelected)
	if err != nil {
		return nil, err
	}

	pkg, err := pkg.FindFirstKclPkgFrom(localPath)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

// PullFromOci will pull a kcl package from oci registry and unpack it.
func (c *KpmClient) PullFromOci(localPath, source, tag string) error {
	localPath, err := filepath.Abs(localPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err)
	}
	if len(source) == 0 {
		return reporter.NewErrorEvent(
			reporter.UnKnownPullWhat,
			errors.FailedPull,
			"oci url or package name must be specified",
		)
	}

	if len(tag) == 0 {
		reporter.ReportMsgTo(
			fmt.Sprintf("start to pull '%s'", source),
			c.logWriter,
		)
	} else {
		reporter.ReportMsgTo(
			fmt.Sprintf("start to pull '%s' with tag '%s'", source, tag),
			c.logWriter,
		)
	}

	ociOpts, err := c.ParseOciOptionFromString(source, tag)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err, fmt.Sprintf("failed to create temp dir '%s'.", tmpDir))
	}
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	storepath := ociOpts.SanitizePathWithSuffix(tmpDir)
	err = c.pullTarFromOci(storepath, ociOpts)
	if err != nil {
		return err
	}

	// Get the (*.tar) file path.
	tarPath := filepath.Join(storepath, constants.KCL_PKG_TAR)
	matches, err := filepath.Glob(tarPath)
	if err != nil || len(matches) != 1 {
		if err == nil {
			err = errors.InvalidPkg
		}

		return reporter.NewErrorEvent(
			reporter.InvalidKclPkg,
			err,
			fmt.Sprintf("failed to find the kcl package tar from '%s'.", tarPath),
		)
	}

	// Untar the tar file.
	storagePath := ociOpts.SanitizePathWithSuffix(localPath)
	err = utils.UnTarDir(matches[0], storagePath)
	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedUntarKclPkg,
			err,
			fmt.Sprintf("failed to untar the kcl package tar from '%s' into '%s'.", matches[0], storagePath),
		)
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("pulled '%s' in '%s' successfully", source, storagePath),
		c.logWriter,
	)
	return nil
}

// PushToOci will push a kcl package to oci registry.
func (c *KpmClient) PushToOci(localPath string, ociOpts *opt.OciOptions) error {
	repoPath := utils.JoinPath(ociOpts.Reg, ociOpts.Repo)
	cred, err := c.GetCredentials(ociOpts.Reg)
	if err != nil {
		return err
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
		oci.WithInsecureSkipTLSverify(c.insecureSkipTLSverify),
	)

	if err != nil {
		return err
	}

	ociCli.SetLogWriter(c.logWriter)

	exist, err := ociCli.ContainsTag(ociOpts.Tag)
	if err != (*reporter.KpmEvent)(nil) {
		return err
	}

	if exist {
		return reporter.NewErrorEvent(
			reporter.PkgTagExists,
			fmt.Errorf("package version '%s' already exists", ociOpts.Tag),
		)
	}

	return ociCli.PushWithOciManifest(localPath, ociOpts.Tag, &opt.OciManifestOptions{
		Annotations: ociOpts.Annotations,
	})
}

// LoginOci will login to the oci registry.
func (c *KpmClient) LoginOci(hostname, username, password string) error {

	credCli, err := c.GetCredsClient()
	if err != nil {
		return err
	}

	err = credCli.GetAuthClient().LoginWithOpts(
		[]auth.LoginOption{
			auth.WithLoginHostname(hostname),
			auth.WithLoginUsername(username),
			auth.WithLoginSecret(password),
		}...,
	)

	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedLogin,
			err,
			fmt.Sprintf("failed to login '%s', please check registry, username and password is valid", hostname),
		)
	}

	return nil
}

// LogoutOci will logout from the oci registry.
func (c *KpmClient) LogoutOci(hostname string) error {

	credCli, err := c.GetCredsClient()
	if err != nil {
		return err
	}

	err = credCli.GetAuthClient().Logout(context.Background(), hostname)

	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedLogout, err, fmt.Sprintf("failed to logout '%s'", hostname))
	}

	return nil
}

// ParseOciRef will parser '<repo_name>:<repo_tag>' into an 'OciOptions'.
func (c *KpmClient) ParseOciRef(ociRef string) (*opt.OciOptions, error) {
	oci_address := strings.Split(ociRef, constants.OCI_SEPARATOR)
	if len(oci_address) == 1 {
		return &opt.OciOptions{
			Reg:  c.GetSettings().DefaultOciRegistry(),
			Repo: utils.JoinPath(c.GetSettings().DefaultOciRepo(), oci_address[0]),
		}, nil
	} else if len(oci_address) == 2 {
		return &opt.OciOptions{
			Reg:  c.GetSettings().DefaultOciRegistry(),
			Repo: utils.JoinPath(c.GetSettings().DefaultOciRepo(), oci_address[0]),
			Tag:  oci_address[1],
		}, nil
	} else {
		return nil, reporter.NewEvent(reporter.IsNotRef)
	}
}

// ParseOciOptionFromString will parser '<repo_name>:<repo_tag>' into an 'OciOptions' with an OCI registry.
// the default OCI registry is 'docker.io'.
// if the 'ociUrl' is only '<repo_name>', ParseOciOptionFromString will take 'latest' as the default tag.
func (c *KpmClient) ParseOciOptionFromString(oci string, tag string) (*opt.OciOptions, error) {
	ociOpt, event := opt.ParseOciUrl(oci)
	if event != nil && (event.Type() == reporter.IsNotUrl || event.Type() == reporter.UrlSchemeNotOci) {
		ociOpt, err := c.ParseOciRef(oci)
		if err != nil {
			return nil, err
		}
		if len(tag) != 0 {
			reporter.ReportEventTo(
				reporter.NewEvent(
					reporter.InvalidFlag,
					"kpm get version from oci reference '<repo_name>:<repo_tag>'",
				),
				c.logWriter,
			)
			reporter.ReportEventTo(
				reporter.NewEvent(
					reporter.InvalidFlag,
					"arg '--tag' is invalid for oci reference",
				),
				c.logWriter,
			)
		}
		return ociOpt, nil
	}

	ociOpt.Tag = tag

	return ociOpt, nil
}

// InitGraphAndDownloadDeps initializes a dependency graph and call downloadDeps function.
func (c *KpmClient) InitGraphAndDownloadDeps(kclPkg *pkg.KclPkg) (*pkg.Dependencies, graph.Graph[module.Version, module.Version], error) {

	moduleHash := func(m module.Version) module.Version {
		return m
	}
	depGraph := graph.New(moduleHash, graph.Directed(), graph.PreventCycles())

	// add the root vertex(package name) to the dependency graph.
	root := module.Version{Path: kclPkg.GetPkgName(), Version: kclPkg.GetPkgVersion()}
	err := depGraph.AddVertex(root)
	if err != nil {
		return nil, nil, err
	}

	changedDeps, err := c.DownloadDeps(&kclPkg.ModFile.Dependencies, &kclPkg.Dependencies, depGraph, kclPkg.HomePath, root)
	if err != nil {
		return nil, nil, err
	}

	return changedDeps, depGraph, nil
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

// downloadDeps will download all the dependencies of the current kcl package.
func (c *KpmClient) DownloadDeps(deps *pkg.Dependencies, lockDeps *pkg.Dependencies, depGraph graph.Graph[module.Version, module.Version], pkghome string, parent module.Version) (*pkg.Dependencies, error) {

	newDeps := pkg.Dependencies{
		Deps: orderedmap.NewOrderedMap[string, pkg.Dependency](),
	}

	// Traverse all dependencies in kcl.mod
	for _, k := range deps.Deps.Keys() {
		d, _ := deps.Deps.Get(k)
		if len(d.Name) == 0 {
			return nil, errors.InvalidDependency
		}

		existDep, err := c.dependencyExistsLocal(pkghome, &d, false)
		if existDep != nil && err == nil {
			newDeps.Deps.Set(d.Name, *existDep)
			continue
		}

		expectedSum := lockDeps.Deps.GetOrDefault(d.Name, pkg.TestPkgDependency).Sum
		// Clean the cache
		if len(c.homePath) == 0 || len(d.FullName) == 0 {
			return nil, errors.InternalBug
		}

		dir := c.getDepStorePath(c.homePath, &d, false)
		err = os.RemoveAll(dir)
		if err != nil {
			return nil, err
		}

		// download dependencies
		lockedDep, err := c.Download(&d, pkghome, dir)
		if err != nil {
			return nil, err
		}

		if lockedDep.Oci != nil && lockedDep.Equals(lockDeps.Deps.GetOrDefault(d.Name, pkg.TestPkgDependency)) {
			if !c.noSumCheck && expectedSum != "" &&
				lockedDep.Sum != "" &&
				lockedDep.Sum != expectedSum {
				return nil, reporter.NewErrorEvent(
					reporter.CheckSumMismatch,
					errors.CheckSumMismatchError,
					fmt.Sprintf("checksum for '%s' changed in lock file '%s' and '%s'", lockedDep.Name, expectedSum, lockedDep.Sum),
				)
			} else {
				lockedDep.Sum = lockDeps.Deps.GetOrDefault(d.Name, pkg.Dependency{}).Sum
			}
		}

		newDeps.Deps.Set(d.Name, *lockedDep)
		// After downloading the dependency in kcl.mod, update the dep into to the kcl.mod
		// Only the direct dependencies are updated to kcl.mod.
		deps.Deps.Set(d.Name, *lockedDep)
	}

	// necessary to make a copy as when we are updating kcl.mod in below for loop
	// then newDeps.Deps gets updated and range gets an extra value to iterate through
	// this messes up the dependency graph
	newDepsCopy := orderedmap.NewOrderedMap[string, pkg.Dependency]()
	for _, k := range newDeps.Deps.Keys() {
		v, ok := newDeps.Deps.Get(k)
		if !ok {
			break
		}
		newDepsCopy.Set(k, v)
	}

	// Recursively download the dependencies of the new dependencies.
	for _, k := range newDepsCopy.Keys() {
		d, ok := newDepsCopy.Get(k)
		if !ok {
			break
		}
		var err error
		var deppkg *pkg.KclPkg
		if len(d.LocalFullPath) != 0 {
			if d.GetPackage() != "" {
				d.LocalFullPath, _ = utils.FindPackage(d.LocalFullPath, d.GetPackage())
			}
		} else {
			// Load kcl.mod file of the new downloaded dependencies.
			if d.GetPackage() != "" {
				d.LocalFullPath, _ = utils.FindPackage(filepath.Join(c.homePath, d.FullName), d.GetPackage())
			}
		}
		deppkg, err = c.LoadPkgFromPath(d.LocalFullPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		source := module.Version{Path: d.Name, Version: d.Version}

		err = depGraph.AddVertex(source, graph.VertexAttribute(d.GetSourceType(), d.GetDownloadPath()))
		if err != nil && err != graph.ErrVertexAlreadyExists {
			return nil, err
		}

		if parent != (module.Version{}) {
			err = depGraph.AddEdge(parent, source)
			if err != nil {
				if err == graph.ErrEdgeCreatesCycle {
					return nil, reporter.NewErrorEvent(
						reporter.CircularDependencyExist,
						nil,
						fmt.Sprintf("adding %s as a dependency results in a cycle", source),
					)
				}
				return nil, err
			}
		}

		// Download the indirect dependencies.
		nested, err := c.DownloadDeps(&deppkg.ModFile.Dependencies, lockDeps, depGraph, deppkg.HomePath, source)
		if err != nil {
			return nil, err
		}

		for _, k := range nested.Deps.Keys() {
			d, ok := nested.Deps.Get(k)
			if !ok {
				break
			}
			if _, ok := newDeps.Deps.Get(d.Name); !ok {
				newDeps.Deps.Set(d.Name, d)
			}
		}
	}

	// After each dependency is downloaded, update all the new deps to kcl.mod.lock.
	// No matter whether the dependency is directly or indirectly.
	for _, k := range newDeps.Deps.Keys() {
		v, ok := newDeps.Deps.Get(k)
		if !ok {
			break
		}
		lockDeps.Deps.Set(k, v)
	}

	return &newDeps, nil
}

// pullTarFromOci will pull a kcl package tar file from oci registry.
func (c *KpmClient) pullTarFromOci(localPath string, ociOpts *opt.OciOptions) error {
	absPullPath, err := filepath.Abs(localPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err)
	}

	repoPath := utils.JoinPath(ociOpts.Reg, ociOpts.Repo)
	cred, err := c.GetCredentials(ociOpts.Reg)
	if err != nil {
		return err
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
		oci.WithInsecureSkipTLSverify(ociOpts.InsecureSkipTLSverify),
	)

	if err != nil {
		return err
	}

	ociCli.SetLogWriter(c.logWriter)

	var tagSelected string
	if len(ociOpts.Tag) == 0 {
		tagSelected, err = ociCli.TheLatestTag()
		if err != nil {
			return err
		}
		reporter.ReportMsgTo(
			fmt.Sprintf("the lastest version '%s' will be pulled", tagSelected),
			c.logWriter,
		)
	} else {
		tagSelected = ociOpts.Tag
	}

	full_repo := utils.JoinPath(ociOpts.Reg, ociOpts.Repo)
	reporter.ReportMsgTo(
		fmt.Sprintf("pulling '%s:%s' from '%s'", ociOpts.Repo, tagSelected, full_repo),
		c.logWriter,
	)

	err = ociCli.Pull(absPullPath, tagSelected)
	if err != nil {
		return err
	}

	return nil
}

// FetchOciManifestConfIntoJsonStr will fetch the oci manifest config of the kcl package from the oci registry and return it into json string.
func (c *KpmClient) FetchOciManifestIntoJsonStr(opts opt.OciFetchOptions) (string, error) {

	repoPath := utils.JoinPath(opts.Reg, opts.Repo)
	cred, err := c.GetCredentials(opts.Reg)
	if err != nil {
		return "", err
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
	)

	if err != nil {
		return "", err
	}

	manifestJson, err := ociCli.FetchManifestIntoJsonStr(opts)
	if err != nil {
		return "", err
	}
	return manifestJson, nil
}

// NewVisitor is a factory function to create a new Visitor.
func NewVisitor(source downloader.Source, kpmcli *KpmClient) visitor.Visitor {
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

// createDepRef will create a dependency reference for the dependency saved on the local filesystem.
// On the unix-like system, it will create a symbolic link.
// On the windows system, it will create a junction.
// func createDepRef(depName, refName string) error {
// 	if runtime.GOOS == "windows" {
// 		// 'go-getter' continuously occupies files in '.git', causing the copy operation to fail
// 		opt := copy.Options{
// 			Skip: func(srcinfo os.FileInfo, src, dest string) (bool, error) {
// 				return filepath.Base(src) == constants.GitPathSuffix, nil
// 			},
// 		}
// 		return copy.Copy(depName, refName, opt)
// 	} else {
// 		return utils.CreateSymlink(depName, refName)
// 	}
// }
