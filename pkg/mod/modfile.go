// Copyright 2022 The KCL Authors. All rights reserved.
package modfile

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"kusionstack.io/kpm/pkg/git"
	"kusionstack.io/kpm/pkg/opt"
	"kusionstack.io/kpm/pkg/reporter"
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
	Dependencies
}

// 'Dependencies' is dependencies section of 'kcl.mod'.
type Dependencies struct {
	Deps map[string]Dependency `toml:"dependencies,omitempty"`
}

type Dependency struct {
	Name    string `toml:"name,omitempty"`
	Version string `toml:"version,omitempty"`
	Sum     string `toml:"sum,omitempty"`
	Source
}

// Download will download the kcl package to localPath from registory.
func (dep *Dependency) Download(localPath string) (*Dependency, error) {
	if dep.Source.Git != nil {
		_, err := dep.Source.Git.Download(localPath)
		if err != nil {
			return nil, err
		}
		dep.Sum = utils.HashDir(localPath)
	}
	return dep, nil
}

// Download will download the kcl package to localPath from git url.
func (dep *Git) Download(localPath string) (string, error) {
	repoURL := dep.Url
	_, err := git.Clone(repoURL, localPath)

	if err != nil {
		reporter.Report("kpm: git clone error:", err)
		return localPath, err
	}

	return localPath, err
}

// Source is the package source from registry.
type Source struct {
	*Git
}

// Git is the package source from git registry.
type Git struct {
	Url    string `toml:"url,omitempty"`
	Branch string `toml:"branch,omitempty"`
	Commit string `toml:"commit,omitempty"`
	Tag    string `toml:"tag,omitempty"`
}

// ModFileExists returns whether a 'kcl.mod' file exists in the path.
func ModFileExists(path string) (bool, error) {
	return exists(filepath.Join(path, MOD_FILE))
}

// ModLockFileExists returns whether a 'kcl.mod.lock' file exists in the path.
func ModLockFileExists(path string) (bool, error) {
	return exists(filepath.Join(path, MOD_LOCK_FILE))
}

// LoadModFile load the contents of the 'kcl.mod' file in the path.
func LoadModFile(homePath string) (*ModFile, error) {
	modFile := new(ModFile)
	err := loadFile(homePath, MOD_FILE, modFile)
	if err != nil {
		return nil, err
	}

	modFile.HomePath = homePath

	if modFile.Dependencies.Deps == nil {
		modFile.Dependencies.Deps = make(map[string]Dependency)
	}

	return modFile, nil
}

// LoadLockDeps will load all dependencies from 'kcl.mod.lock'.
func LoadLockDeps(homePath string) (*Dependencies, error) {
	deps := new(Dependencies)
	err := loadFile(homePath, MOD_FILE, deps)
	if err != nil {
		return nil, err
	}

	if deps.Deps == nil {
		deps.Deps = make(map[string]Dependency)
	}

	return deps, nil
}

// Write the contents of 'ModFile' to 'kcl.mod' file
func (mfile *ModFile) Store() error {
	fullPath := filepath.Join(mfile.HomePath, MOD_FILE)
	return StoreToFile(fullPath, mfile)
}

func (mfile *ModFile) GetModFilePath() string {
	return filepath.Join(mfile.HomePath, MOD_FILE)
}

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
		Dependencies: Dependencies{Deps: make(map[string]Dependency)},
	}
}

// StoreToFile will store 'data' into toml file under 'filePath'.
func StoreToFile(filePath string, data interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		reporter.ExitWithReport("kpm: failed to create file: ", filePath, err)
		return err
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(data); err != nil {
		reporter.ExitWithReport("kpm: failed to encode TOML:", err)
		return err
	}
	return nil
}

func loadFile(homePath string, fileName string, file interface{}) error {
	readFile, err := os.OpenFile(filepath.Join(homePath, fileName), os.O_RDWR, 0644)
	if err != nil {
		reporter.Report("kpm: failed to load", fileName)
		return err
	}
	defer readFile.Close()

	_, err = toml.NewDecoder(readFile).Decode(file)
	if err != nil {
		return err
	}

	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

/// Parse out some information for a Dependency from registry url.
func ParseOpt(opt *opt.RegistryOptions) *Dependency {
	if opt.Git != nil {
		gitSource := Git{
			Url:    opt.Git.Url,
			Branch: opt.Git.Branch,
			Commit: opt.Git.Commit,
			Tag:    opt.Git.Tag,
		}

		name := ParseRepoNameFromGitUrl(gitSource.Url)

		return &Dependency{
			Name: name,
			Source: Source{
				Git: &gitSource,
			},
		}
	}
	return nil
}

// ParseRepoNameFromGitUrl will extract the kcl package name from the git url.
func ParseRepoNameFromGitUrl(gitUrl string) string {
	name := filepath.Base(gitUrl)
	return name[:len(name)-len(filepath.Ext(name))]
}
