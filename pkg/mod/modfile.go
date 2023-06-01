// Copyright 2022 The KCL Authors. All rights reserved.
package modfile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/git"
	"kusionstack.io/kpm/pkg/oci"
	"kusionstack.io/kpm/pkg/opt"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/settings"
	"kusionstack.io/kpm/pkg/utils"
)

const (
	MOD_FILE      = "kcl.mod"
	MOD_LOCK_FILE = "kcl.mod.lock"
)

// 'Package' is the kcl package section of 'kcl.mod'.
type Package struct {
	Name    string `toml:"name,omitempty"`    // kcl package name
	Edition string `toml:"edition,omitempty"` // kclvm version
	Version string `toml:"version,omitempty"` // kcl package version
}

// 'ModFile' is kcl package file 'kcl.mod'.
type ModFile struct {
	HomePath string  `toml:"-"`
	Pkg      Package `toml:"package,omitempty"`
	// Whether the current package uses the vendor mode
	// In the vendor mode, kpm will look for the package in the vendor subdirectory
	// in the current package directory.
	VendorMode bool `toml:"-"`
	Dependencies
}

// FillDependenciesInfo will fill registry information for all dependencies in a kcl.mod.
func (modFile *ModFile) FillDependenciesInfo() error {
	for k, v := range modFile.Deps {
		err := v.FillDepInfo()
		if err != nil {
			return err
		}
		modFile.Deps[k] = v
	}
	return nil
}

// 'Dependencies' is dependencies section of 'kcl.mod'.
type Dependencies struct {
	Deps map[string]Dependency `json:"packages" toml:"dependencies,omitempty"`
}

type Dependency struct {
	Name     string `json:"name" toml:"name,omitempty"`
	FullName string `json:"-" toml:"full_name,omitempty"`
	Version  string `json:"-" toml:"version,omitempty"`
	Sum      string `json:"-" toml:"sum,omitempty"`
	// The actual local path of the package.
	// In vendor mode is "current_kcl_package/vendor"
	// In non-vendor mode is "$KCL_PKG_PATH"
	LocalFullPath string `json:"manifest_path" toml:"-"`
	Source        `json:"-"`
}

// FillDepInfo will fill registry information for a dependency.
func (dep *Dependency) FillDepInfo() error {
	if dep.Source.Oci != nil {
		settings, err := settings.GetSettings()
		if err != nil {
			return err
		}
		dep.Source.Oci.Reg = settings.DefaultOciRegistry()
		dep.Source.Oci.Repo = filepath.Join(settings.DefaultOciRepo(), dep.Name)
	}
	return nil
}

// Download will download the kcl package to localPath from registory.
func (dep *Dependency) Download(localPath string) (*Dependency, error) {
	reporter.Report("kpm: adding dependency", dep.Name)
	if dep.Source.Git != nil {
		_, err := dep.Source.Git.Download(localPath)
		if err != nil {
			return nil, err
		}
	}

	if dep.Source.Oci != nil {
		_, err := dep.Source.Oci.Download(localPath)
		if err != nil {
			return nil, err
		}
	}

	var err error
	dep.Sum, err = utils.HashDir(localPath)
	if err != nil {
		return nil, err
	}
	dep.LocalFullPath = localPath
	// Creating symbolic links in a global cache is not an optimal solution.
	// This allows kclvm to locate the package by default.
	// This feature is unstable and will be removed soon.
	err = utils.CreateSymlink(dep.LocalFullPath, filepath.Join(filepath.Dir(localPath), dep.Name))
	if err != nil {
		return nil, err
	}

	return dep, nil
}

// Download will download the kcl package to localPath from git url.
func (dep *Git) Download(localPath string) (string, error) {
	_, err := git.Clone(dep.Url, dep.Tag, localPath)

	if err != nil {
		reporter.Report("kpm: git clone error:", err)
		return localPath, err
	}

	return localPath, err
}

func (dep *Oci) Download(localPath string) (string, error) {
	ociClient, err := oci.NewOciClient(dep.Reg, dep.Repo)
	if err != nil {
		return "", err
	}
	// Select the latest tag, if the tag, the user inputed, is empty.
	var tagSelected string
	if len(dep.Tag) == 0 {
		tagSelected, err = ociClient.TheLatestTag()
		if err != nil {
			return "", err
		}
		reporter.Report("kpm: the lastest version", tagSelected, "will be pulled.")
		dep.Tag = tagSelected
	} else {
		tagSelected = dep.Tag
	}

	// Pull the package with the tag.
	err = ociClient.Pull(localPath, tagSelected)
	if err != nil {
		return "", err
	}

	matches, err := filepath.Glob(filepath.Join(localPath, "*.tar"))
	if err != nil || len(matches) != 1 {
		return "", errors.FailedPullFromOci
	}

	tarPath := matches[0]
	err = utils.UnTarDir(tarPath, localPath)
	if err != nil {
		return "", err
	}

	// After untar the downloaded kcl package tar file, remove the tar file.
	if utils.DirExists(tarPath) {
		err = os.Remove(tarPath)
		if err != nil {
			return "", err
		}
	}

	return localPath, nil
}

// Source is the package source from registry.
type Source struct {
	*Git
	*Oci
}

type Oci struct {
	Reg  string `toml:"reg,omitempty"`
	Repo string `toml:"repo,omitempty"`
	Tag  string `toml:"oci_tag,omitempty"`
}

// Git is the package source from git registry.
type Git struct {
	Url    string `toml:"url,omitempty"`
	Branch string `toml:"branch,omitempty"`
	Commit string `toml:"commit,omitempty"`
	Tag    string `toml:"git_tag,omitempty"`
}

// ModFileExists returns whether a 'kcl.mod' file exists in the path.
func ModFileExists(path string) (bool, error) {
	return utils.Exists(filepath.Join(path, MOD_FILE))
}

// ModLockFileExists returns whether a 'kcl.mod.lock' file exists in the path.
func ModLockFileExists(path string) (bool, error) {
	return utils.Exists(filepath.Join(path, MOD_LOCK_FILE))
}

// LoadModFile load the contents of the 'kcl.mod' file in the path.
func LoadModFile(homePath string) (*ModFile, error) {
	modFile := new(ModFile)
	err := modFile.loadModFile(filepath.Join(homePath, MOD_FILE))
	if err != nil {
		return nil, err
	}

	modFile.HomePath = homePath

	if modFile.Dependencies.Deps == nil {
		modFile.Dependencies.Deps = make(map[string]Dependency)
	}
	err = modFile.FillDependenciesInfo()
	if err != nil {
		return nil, err
	}

	return modFile, nil
}

// LoadLockDeps will load all dependencies from 'kcl.mod.lock'.
func LoadLockDeps(homePath string) (*Dependencies, error) {
	deps := new(Dependencies)
	deps.Deps = make(map[string]Dependency)
	err := deps.loadLockFile(filepath.Join(homePath, MOD_LOCK_FILE))

	if os.IsNotExist(err) {
		return deps, nil
	}

	if err != nil {
		return nil, err
	}

	return deps, nil
}

// Write the contents of 'ModFile' to 'kcl.mod' file
func (mfile *ModFile) StoreModFile() error {
	fullPath := filepath.Join(mfile.HomePath, MOD_FILE)
	return utils.StoreToFile(fullPath, mfile.MarshalTOML())
}

// Returns the path to the kcl.mod file
func (mfile *ModFile) GetModFilePath() string {
	return filepath.Join(mfile.HomePath, MOD_FILE)
}

// Returns the path to the kcl.mod.lock file
func (mfile *ModFile) GetModLockFilePath() string {
	return filepath.Join(mfile.HomePath, MOD_LOCK_FILE)
}

const defaultVerion = "0.0.1"
const defaultEdition = "0.0.1"

func NewModFile(opts *opt.InitOptions) *ModFile {
	return &ModFile{
		HomePath: opts.InitPath,
		Pkg: Package{
			Name:    opts.Name,
			Version: defaultVerion,
			Edition: defaultEdition,
		},
		Dependencies: Dependencies{
			Deps: make(map[string]Dependency),
		},
	}
}

// Load the kcl.mod file.
func (mod *ModFile) loadModFile(filepath string) error {

	modData, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	err = toml.Unmarshal(modData, &mod)

	if err != nil {
		return err
	}

	return nil
}

// Load the kcl.mod.lock file.
func (deps *Dependencies) loadLockFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if os.IsNotExist(err) {
		return err
	}

	if err != nil {
		reporter.Report("kpm: failed to load", filepath)
		return err
	}

	err = deps.UnmarshalLockTOML(string(data))

	if err != nil {
		reporter.Report("kpm: failed to load", filepath)
		return err
	}

	return nil
}

// Parse out some information for a Dependency from registry url.
func ParseOpt(opt *opt.RegistryOptions) *Dependency {
	if opt.Git != nil {
		gitSource := Git{
			Url:    opt.Git.Url,
			Branch: opt.Git.Branch,
			Commit: opt.Git.Commit,
			Tag:    opt.Git.Tag,
		}

		return &Dependency{
			Name:     ParseRepoNameFromGitSource(gitSource),
			FullName: ParseRepoFullNameFromGitSource(gitSource),
			Source: Source{
				Git: &gitSource,
			},
			Version: gitSource.Tag,
		}
	}
	if opt.Oci != nil {
		ociSource := Oci{
			Reg:  opt.Oci.Reg,
			Repo: filepath.Join(opt.Oci.Repo, opt.Oci.PkgName),
			Tag:  opt.Oci.Tag,
		}

		return &Dependency{
			Name:     opt.Oci.PkgName,
			FullName: opt.Oci.PkgName + "_" + opt.Oci.Tag,
			Source: Source{
				Oci: &ociSource,
			},
			Version: opt.Oci.Tag,
		}
	}
	return nil
}

const PKG_NAME_PATTERN = "%s_%s"

// ParseRepoFullNameFromGitSource will extract the kcl package name from the git url.
func ParseRepoFullNameFromGitSource(gitSrc Git) string {
	if len(gitSrc.Tag) != 0 {
		return fmt.Sprintf(PKG_NAME_PATTERN, utils.ParseRepoNameFromGitUrl(gitSrc.Url), gitSrc.Tag)
	}
	return utils.ParseRepoNameFromGitUrl(gitSrc.Url)
}

// ParseRepoNameFromGitSource will extract the kcl package name from the git url.
func ParseRepoNameFromGitSource(gitSrc Git) string {
	return utils.ParseRepoNameFromGitUrl(gitSrc.Url)
}
