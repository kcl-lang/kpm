// Copyright 2022 The KCL Authors. All rights reserved.
package pkg

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"oras.land/oras-go/v2/registry"

	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/runner"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

const (
	MOD_FILE      = "kcl.mod"
	MOD_LOCK_FILE = "kcl.mod.lock"
	GIT           = "git"
	OCI           = "oci"
	LOCAL         = "local"
)

// 'Package' is the kcl package section of 'kcl.mod'.
type Package struct {
	// The name of the package.
	Name string `toml:"name,omitempty"`
	// The kcl compiler version
	Edition string `toml:"edition,omitempty"`
	// The version of the package.
	Version string `toml:"version,omitempty"`
	// Description denotes the description of the package.
	Description string `toml:"description,omitempty"` // kcl package description
	// Exclude denote the files to include when publishing.
	Include []string `toml:"include,omitempty"`
	// Exclude denote the files to exclude when publishing.
	Exclude []string `toml:"exclude,omitempty"`
}

// 'ModFile' is kcl package file 'kcl.mod'.
type ModFile struct {
	HomePath string  `toml:"-"`
	Pkg      Package `toml:"package,omitempty"`
	// Whether the current package uses the vendor mode
	// In the vendor mode, kpm will look for the package in the vendor subdirectory
	// in the current package directory.
	VendorMode bool     `toml:"-"`
	Profiles   *Profile `toml:"profile"`
	Dependencies
}

// Profile is the profile section of 'kcl.mod'.
// It is used to specify the compilation options of the current package.
type Profile struct {
	Entries     *[]string `toml:"entries"`
	DisableNone *bool     `toml:"disable_none"`
	SortKeys    *bool     `toml:"sort_keys"`
	Selectors   *[]string `toml:"selectors"`
	Overrides   *[]string `toml:"overrides"`
	Options     *[]string `toml:"arguments"`
}

// NewProfile will create a new profile.
func NewProfile() Profile {
	return Profile{}
}

// IntoKclOptions will transform the profile into kcl options.
func (profile *Profile) IntoKclOptions() *kcl.Option {

	opts := kcl.NewOption()

	if profile.Entries != nil {
		for _, entry := range *profile.Entries {
			ext := filepath.Ext(entry)
			if ext == ".yaml" {
				opts.Merge(kcl.WithSettings(entry))
			} else {
				opts.Merge(kcl.WithKFilenames(entry))
			}
		}
	}

	if profile.DisableNone != nil {
		opts.Merge(kcl.WithDisableNone(*profile.DisableNone))
	}

	if profile.SortKeys != nil {
		opts.Merge(kcl.WithSortKeys(*profile.SortKeys))
	}

	if profile.Selectors != nil {
		opts.Merge(kcl.WithSelectors(*profile.Selectors...))
	}

	if profile.Overrides != nil {
		opts.Merge(kcl.WithOverrides(*profile.Overrides...))
	}

	if profile.Options != nil {
		opts.Merge(kcl.WithOptions(*profile.Options...))
	}

	return opts
}

// GetEntries will get the entry kcl files from profile.
func (profile *Profile) GetEntries() []string {
	if profile == nil || profile.Entries == nil {
		return []string{}
	}
	return *profile.Entries
}

// FillDependenciesInfo will fill registry information for all dependencies in a kcl.mod.
func (modFile *ModFile) FillDependenciesInfo() error {
	for k, v := range modFile.Deps {
		err := v.FillDepInfo(modFile.HomePath)
		if err != nil {
			return err
		}
		modFile.Deps[k] = v
	}
	return nil
}

// GetEntries will get the entry kcl files from kcl.mod.
func (modFile *ModFile) GetEntries() []string {
	if modFile.Profiles == nil {
		return []string{}
	}
	return modFile.Profiles.GetEntries()
}

// 'Dependencies' is dependencies section of 'kcl.mod'.
type Dependencies struct {
	Deps map[string]Dependency `json:"packages" toml:"dependencies,omitempty"`
}

// SortMapByKey will sort the dependencies by key.
func (deps *Dependencies) SortMapByKey() {
	keys := make([]string, 0, len(deps.Deps))
	for k := range deps.Deps {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sortedDeps := make(map[string]Dependency)
	for _, k := range keys {
		sortedDeps[k] = deps.Deps[k]
	}
	deps.Deps = sortedDeps
}

// ToDepMetadata will transform the dependencies into metadata.
// And check whether the dependency name conflicts.
func (deps *Dependencies) ToDepMetadata() (*Dependencies, error) {
	depMetadata := Dependencies{
		Deps: make(map[string]Dependency),
	}
	for _, d := range deps.Deps {
		if _, ok := depMetadata.Deps[d.GetAliasName()]; ok {
			return nil, reporter.NewErrorEvent(
				reporter.PathIsEmpty,
				fmt.Errorf("dependency name conflict, '%s' already exists", d.GetAliasName()),
				"because '-' in the original dependency names is replaced with '_'\n",
				"please check your dependencies with '-' or '_' in dependency name",
			)
		}
		d.Name = d.GetAliasName()
		depMetadata.Deps[d.GetAliasName()] = d
	}

	return &depMetadata, nil
}

func (deps *Dependencies) CheckForLocalDeps() bool {
	for _, dep := range deps.Deps {
		if dep.IsFromLocal() {
			return true
		}
	}
	return false
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

func (d *Dependency) FromKclPkg(pkg *KclPkg) {
	d.FullName = pkg.GetPkgFullName()
	d.Version = pkg.GetPkgVersion()
	d.LocalFullPath = pkg.HomePath
}

// SetName will set the name and alias name of a dependency.
func (d *Dependency) GetAliasName() string {
	return strings.ReplaceAll(d.Name, "-", "_")
}

func (d Dependency) Equals(other Dependency) bool {
	var sameVersion = true
	if len(d.Version) != 0 && len(other.Version) != 0 {
		sameVersion = d.Version == other.Version

	}
	sameNameAndVersion := d.Name == other.Name && sameVersion
	sameGitSrc := true
	if d.Source.Git != nil && other.Source.Git != nil {
		sameGitSrc = d.Source.Git.Url == other.Source.Git.Url &&
			(d.Source.Git.Branch == other.Source.Git.Branch ||
				d.Source.Git.Commit == other.Source.Git.Commit ||
				d.Source.Git.Tag == other.Source.Git.Tag)
	}

	sameOciSrc := true
	if d.Source.Oci != nil && other.Source.Oci != nil {
		sameOciSrc = d.Source.Oci.Reg == other.Source.Oci.Reg &&
			d.Source.Oci.Repo == other.Source.Oci.Repo &&
			d.Source.Oci.Tag == other.Source.Oci.Tag
	}

	return sameNameAndVersion && sameGitSrc && sameOciSrc
}

// GetLocalFullPath will get the local path of a dependency.
func (dep *Dependency) GetLocalFullPath(rootpath string) string {
	if !filepath.IsAbs(dep.LocalFullPath) && dep.IsFromLocal() {
		if filepath.IsAbs(dep.Source.Local.Path) {
			return dep.Source.Local.Path
		}
		return filepath.Join(rootpath, dep.Source.Local.Path)
	}
	return dep.LocalFullPath
}

func (dep *Dependency) IsFromLocal() bool {
	return dep.Source.Oci == nil && dep.Source.Git == nil && dep.Source.Local != nil
}

// FillDepInfo will fill registry information for a dependency.
func (dep *Dependency) FillDepInfo(homepath string) error {
	if dep.Source.Oci != nil {
		settings := settings.GetSettings()
		if settings.ErrorEvent != nil {
			return settings.ErrorEvent
		}
		if dep.Source.Oci.Reg == "" {
			dep.Source.Oci.Reg = settings.DefaultOciRegistry()
		}

		if dep.Source.Oci.Repo == "" {
			urlpath := utils.JoinPath(settings.DefaultOciRepo(), dep.Name)
			dep.Source.Oci.Repo = urlpath
		}
	}
	if dep.Source.Local != nil {
		dep.LocalFullPath = dep.Source.Local.Path
	}
	return nil
}

// GenDepFullName will generate the full name of a dependency by its name and version
// based on the '<package_name>_<package_tag>' format.
func (dep *Dependency) GenDepFullName() string {
	dep.FullName = fmt.Sprintf(PKG_NAME_PATTERN, dep.Name, dep.Version)
	return dep.FullName
}

// GetDownloadPath will get the download path of a dependency.
func (dep *Dependency) GetDownloadPath() string {
	if dep.Source.Git != nil {
		return dep.Source.Git.Url
	}
	if dep.Source.Oci != nil {
		return dep.Source.Oci.IntoOciUrl()
	}
	return ""
}

func GenSource(sourceType string, uri string, tagName string) (Source, error) {
	source := Source{}
	if sourceType == GIT {
		source.Git = &Git{
			Url: uri,
			Tag: tagName,
		}
		return source, nil
	}
	if sourceType == OCI {
		oci := Oci{}
		_, err := oci.FromString(uri + ":" + tagName)
		if err != nil {
			return Source{}, err
		}
		source.Oci = &oci
	}
	if sourceType == LOCAL {
		source.Local = &Local{
			Path: uri,
		}
	}
	return source, nil
}

// GetSourceType will get the source type of a dependency.
func (dep *Dependency) GetSourceType() string {
	if dep.Source.Git != nil {
		return GIT
	}
	if dep.Source.Oci != nil {
		return OCI
	}
	if dep.Source.Local != nil {
		return LOCAL
	}
	return ""
}

// Source is the package source from registry.
type Source struct {
	*Git
	*Oci
	*Local `toml:"-"`
}

type Local struct {
	Path string `toml:"path,omitempty"`
}

type Oci struct {
	Reg  string `toml:"reg,omitempty"`
	Repo string `toml:"repo,omitempty"`
	Tag  string `toml:"oci_tag,omitempty"`
}

func (oci *Oci) IntoOciUrl() string {
	if oci != nil {
		u := &url.URL{
			Scheme: constants.OciScheme,
			Host:   oci.Reg,
			Path:   oci.Repo,
		}

		return u.String()
	}
	return ""
}

func (oci *Oci) FromString(ociUrl string) (*Oci, error) {
	u, err := url.Parse(ociUrl)
	if err != nil {
		return nil, err
	}

	if u.Scheme != constants.OciScheme {
		return nil, fmt.Errorf("invalid oci url with schema: %s", u.Scheme)
	}

	ref, err := registry.ParseReference(u.Host + u.Path)
	if err != nil {
		return nil, fmt.Errorf("'%s' invalid URL format: %w", ociUrl, err)
	}

	oci.Reg = ref.Registry
	oci.Repo = ref.Repository
	oci.Tag = ref.ReferenceOrDefault()

	return oci, nil
}

// Git is the package source from git registry.
type Git struct {
	Url     string `toml:"url,omitempty"`
	Branch  string `toml:"branch,omitempty"`
	Commit  string `toml:"commit,omitempty"`
	Tag     string `toml:"git_tag,omitempty"`
	Version string `toml:"version,omitempty"`
}

// GetValidGitReference will get the valid git reference from git source.
// Only one of branch, tag or commit is allowed.
func (git *Git) GetValidGitReference() (string, error) {
	nonEmptyFields := 0
	var nonEmptyRef string

	if git.Tag != "" {
		nonEmptyFields++
		nonEmptyRef = git.Tag
	}
	if git.Commit != "" {
		nonEmptyFields++
		nonEmptyRef = git.Commit
	}
	if git.Branch != "" {
		nonEmptyFields++
		nonEmptyRef = git.Branch
	}

	if nonEmptyFields != 1 {
		return "", errors.New("only one of branch, tag or commit is allowed")
	}

	return nonEmptyRef, nil
}

// ModFileExists returns whether a 'kcl.mod' file exists in the path.
func ModFileExists(path string) (bool, error) {
	return utils.Exists(filepath.Join(path, MOD_FILE))
}

// ModLockFileExists returns whether a 'kcl.mod.lock' file exists in the path.
func ModLockFileExists(path string) (bool, error) {
	return utils.Exists(filepath.Join(path, MOD_LOCK_FILE))
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

var defaultEdition = runner.GetKclVersion()

func NewModFile(opts *opt.InitOptions) *ModFile {
	if opts.Version == "" {
		opts.Version = defaultVerion
	}
	return &ModFile{
		HomePath: opts.InitPath,
		Pkg: Package{
			Name:    opts.Name,
			Version: opts.Version,
			Edition: defaultEdition,
		},
		Dependencies: Dependencies{
			Deps: make(map[string]Dependency),
		},
	}
}

// Load the kcl.mod file.
func (mod *ModFile) LoadModFile(filepath string) error {

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

// LoadModFile load the contents of the 'kcl.mod' file in the path.
func LoadModFile(homePath string) (*ModFile, error) {
	modFile := new(ModFile)
	err := modFile.LoadModFile(filepath.Join(homePath, MOD_FILE))
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

// Load the kcl.mod.lock file.
func (deps *Dependencies) loadLockFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if os.IsNotExist(err) {
		return err
	}

	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedLoadKclModLock, err, fmt.Sprintf("failed to load '%s'", filepath))
	}

	err = deps.UnmarshalLockTOML(string(data))

	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedLoadKclModLock, err, fmt.Sprintf("failed to load '%s'", filepath))
	}

	return nil
}

// Parse out some information for a Dependency from registry url.
func ParseOpt(opt *opt.RegistryOptions) (*Dependency, error) {
	if opt.Git != nil {
		gitSource := Git{
			Url:    opt.Git.Url,
			Branch: opt.Git.Branch,
			Commit: opt.Git.Commit,
			Tag:    opt.Git.Tag,
		}

		gitRef, err := gitSource.GetValidGitReference()
		if err != nil {
			return nil, err
		}

		fullName, err := ParseRepoFullNameFromGitSource(gitSource)
		if err != nil {
			return nil, err
		}

		return &Dependency{
			Name:     ParseRepoNameFromGitSource(gitSource),
			FullName: fullName,
			Source: Source{
				Git: &gitSource,
			},
			Version: gitRef,
		}, nil
	}
	if opt.Oci != nil {
		repoPath := utils.JoinPath(opt.Oci.Repo, opt.Oci.PkgName)
		ociSource := Oci{
			Reg:  opt.Oci.Reg,
			Repo: repoPath,
			Tag:  opt.Oci.Tag,
		}

		return &Dependency{
			Name:     opt.Oci.PkgName,
			FullName: opt.Oci.PkgName + "_" + opt.Oci.Tag,
			Source: Source{
				Oci: &ociSource,
			},
			Version: opt.Oci.Tag,
		}, nil
	}
	if opt.Local != nil {
		depPkg, err := LoadKclPkg(opt.Local.Path)
		if err != nil {
			return nil, err
		}

		return &Dependency{
			Name:          depPkg.ModFile.Pkg.Name,
			FullName:      depPkg.ModFile.Pkg.Name + "_" + depPkg.ModFile.Pkg.Version,
			LocalFullPath: opt.Local.Path,
			Source: Source{
				Local: &Local{
					Path: opt.Local.Path,
				},
			},
			Version: depPkg.ModFile.Pkg.Version,
		}, nil

	}
	return nil, nil
}

const PKG_NAME_PATTERN = "%s_%s"

// ParseRepoFullNameFromGitSource will extract the kcl package name from the git url.
func ParseRepoFullNameFromGitSource(gitSrc Git) (string, error) {
	ref, err := gitSrc.GetValidGitReference()
	if err != nil {
		return "", err
	}
	if len(ref) != 0 {
		return fmt.Sprintf(PKG_NAME_PATTERN, utils.ParseRepoNameFromGitUrl(gitSrc.Url), ref), nil
	}
	return utils.ParseRepoNameFromGitUrl(gitSrc.Url), nil
}

// ParseRepoNameFromGitSource will extract the kcl package name from the git url.
func ParseRepoNameFromGitSource(gitSrc Git) string {
	return utils.ParseRepoNameFromGitUrl(gitSrc.Url)
}
