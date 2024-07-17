package downloader

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

// Source is the package source from registry.
type Source struct {
	*Registry
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

// Git is the package source from git registry.
type Git struct {
	Url     string `toml:"url,omitempty"`
	Branch  string `toml:"branch,omitempty"`
	Commit  string `toml:"commit,omitempty"`
	Tag     string `toml:"git_tag,omitempty"`
	Version string `toml:"version,omitempty"`
}

type Registry struct {
	*Oci    `toml:"-"`
	Name    string `toml:"-"`
	Version string `toml:"-"`
}

func NewSourceFromStr(sourceStr string) (*Source, error) {
	source := &Source{}
	err := source.FromString(sourceStr)
	if err != nil {
		return nil, err
	}
	return source, nil
}

func (source *Source) IsLocalPath() bool {
	return source.Local != nil
}

func (source *Source) IsLocalTarPath() bool {
	return source.Local.IsLocalTarPath()
}

func (source *Source) IsLocalTgzPath() bool {
	return source.Local.IsLocalTgzPath()
}

func (source *Source) IsRemote() bool {
	return source.Git != nil || source.Oci != nil || source.Registry != nil
}

func (source *Source) IsPackaged() bool {
	return source.IsLocalTarPath() || source.Git != nil || source.Oci != nil || source.Registry != nil
}

func (source *Source) FindRootPath() (string, error) {
	if source == nil {
		return "", fmt.Errorf("source is nil")
	}
	if source.Git != nil {
		return source.Git.ToFilePath()
	}
	if source.Oci != nil {
		return source.Oci.ToFilePath()
	}
	if source.Local != nil {
		return source.Local.FindRootPath()
	}
	if source.Registry != nil {
		return source.Registry.ToFilePath()
	}
	return "", fmt.Errorf("source is nil")

}

func (local *Local) IsLocalTarPath() bool {
	return local != nil && filepath.Ext(local.Path) == constants.TarPathSuffix
}

func (local *Local) IsLocalTgzPath() bool {
	return local != nil && filepath.Ext(local.Path) == constants.TarPathSuffix
}

func (local *Local) IsLocalKPath() bool {
	return local != nil && filepath.Ext(local.Path) == constants.KFilePathSuffix
}

func (local *Local) IsDir() bool {
	if local == nil {
		return false
	}
	fileInfo, err := os.Stat(local.Path)
	if err != nil {
		return false
	}

	return local != nil && utils.DirExists(local.Path) && fileInfo.IsDir()
}

func (local *Local) FindRootPath() (string, error) {
	if local == nil {
		return "", fmt.Errorf("local source is nil")
	}

	// if local.Path is a directory, judge if it has kcl.mod file
	if utils.DirExists(filepath.Join(local.Path, constants.KCL_MOD)) {
		abspath, err := filepath.Abs(local.Path)
		if err != nil {
			return "", err
		}
		return abspath, nil
	}

	// if local.Path is a *.k file, find the kcl.mod file in the same directory and in the parent directory

	dir := filepath.Dir(local.Path)
	for {
		kclModPath := filepath.Join(dir, constants.KCL_MOD)
		if utils.DirExists(kclModPath) {
			abspath, err := filepath.Abs(kclModPath)
			if err != nil {
				return "", err
			}
			return filepath.Dir(abspath), nil
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break
		}
		dir = parentDir
	}

	// If no kcl.mod file is found, return the directory of the original file
	var abspath string
	var err error
	if local.IsLocalKPath() {
		abspath, err = filepath.Abs(filepath.Dir(local.Path))
		if err != nil {
			return "", err
		}
	} else {
		abspath, err = filepath.Abs(local.Path)
		if err != nil {
			return "", err
		}
	}

	return abspath, nil
}

func (source *Source) ToFilePath() (string, error) {
	if source == nil {
		return "", fmt.Errorf("source is nil")
	}
	if source.Git != nil {
		return source.Git.ToFilePath()
	}
	if source.Oci != nil {
		return source.Oci.ToFilePath()
	}
	if source.Local != nil {
		return source.Local.ToFilePath()
	}
	if source.Registry != nil {
		return source.Registry.ToFilePath()
	}
	return "", fmt.Errorf("source is nil")
}

func (git *Git) ToFilePath() (string, error) {
	if git == nil {
		return "", fmt.Errorf("git source is nil")
	}

	gitUrl, err := url.Parse(git.Url)
	if err != nil {
		return "", err
	}

	return filepath.Join(
		constants.GitScheme,
		gitUrl.Host,
		gitUrl.Path,
		git.Tag,
		git.Commit,
		git.Branch,
	), nil
}

func (oci *Oci) ToFilePath() (string, error) {
	if oci == nil {
		return "", fmt.Errorf("oci source is nil")
	}

	ociUrl := &url.URL{
		Host: oci.Reg,
		Path: oci.Repo,
	}

	return filepath.Join(constants.OciScheme, ociUrl.Host, ociUrl.Path, oci.Tag), nil
}

func (local *Local) ToFilePath() (string, error) {
	if local == nil {
		return "", fmt.Errorf("local source is nil")
	}

	return local.ToString()
}

func (registry *Registry) ToFilePath() (string, error) {
	if registry == nil {
		return "", fmt.Errorf("registry is nil")
	}

	ociPath, err := registry.Oci.ToFilePath()
	if err != nil {
		return "", err
	}
	return ociPath, nil
}

func (source *Source) ToString() (string, error) {
	if source == nil {
		return "", fmt.Errorf("source is nil")
	}
	if source.Git != nil {
		return source.Git.ToString()
	}
	if source.Oci != nil {
		return source.Oci.ToString()
	}
	if source.Local != nil {
		return source.Local.ToString()
	}
	if source.Registry != nil {
		return source.Registry.ToString()
	}
	return "", fmt.Errorf("source is nil")
}

func (git *Git) ToString() (string, error) {
	if git == nil {
		return "", fmt.Errorf("git source is nil")
	}

	gitUrl, err := url.Parse(git.Url)
	if err != nil {
		return "", err
	}
	q := gitUrl.Query()

	if git.Tag != "" {
		q.Set(constants.Tag, git.Tag)
	}

	if git.Commit != "" {
		q.Set(constants.GitCommit, git.Commit)
	}

	if git.Branch != "" {
		q.Set(constants.GitBranch, git.Branch)
	}

	gitUrl.RawQuery = q.Encode()

	return gitUrl.String(), nil
}

func (oci *Oci) ToString() (string, error) {
	if oci == nil {
		return "", fmt.Errorf("oci source is nil")
	}

	ociUrl := &url.URL{
		Scheme: constants.OciScheme,
		Host:   oci.Reg,
		Path:   oci.Repo,
	}
	q := ociUrl.Query()
	if oci.Tag != "" {
		q.Set(constants.Tag, oci.Tag)
	}
	ociUrl.RawQuery = q.Encode()

	return ociUrl.String(), nil
}

func (local *Local) ToString() (string, error) {
	if local == nil {
		return "", fmt.Errorf("local source is nil")
	}

	pathUrl, error := url.Parse(local.Path)
	if error != nil {
		return "", error
	}

	return pathUrl.String(), nil
}

func (registry *Registry) ToString() (string, error) {
	ociStr, err := registry.Oci.ToString()
	if err != nil {
		return "", err
	}

	return ociStr, nil
}

func (source *Source) FromString(sourceStr string) error {
	if source == nil {
		return fmt.Errorf("source is nil")
	}

	sourceUrl, err := url.Parse(sourceStr)
	if err != nil {
		return err
	}

	if sourceUrl.Scheme == constants.GitScheme || sourceUrl.Scheme == constants.SshScheme {
		source.Git = &Git{}
		source.Git.FromString(sourceStr)
	} else if sourceUrl.Scheme == constants.OciScheme {
		source.Oci = &Oci{}
		source.Oci.FromString(sourceStr)
	} else if sourceUrl.Scheme == constants.DefaultOciScheme {
		source.Registry = &Registry{}
		source.Registry.FromString(sourceStr)
	} else {
		source.Local = &Local{}
		source.Local.FromString(sourceStr)
	}

	return nil
}

func (git *Git) FromString(gitStr string) error {
	if git == nil {
		return fmt.Errorf("git source is nil")
	}
	u, err := url.Parse(gitStr)
	if err != nil {
		return err
	}

	if u.Scheme != constants.GitScheme && u.Scheme != constants.SshScheme {
		return fmt.Errorf("invalid git url with schema: %s", u.Scheme)
	}

	if u.Scheme == constants.GitScheme {
		u.Scheme = constants.HttpsScheme
	}

	git.Tag = u.Query().Get(constants.Tag)
	git.Commit = u.Query().Get(constants.GitCommit)
	git.Branch = u.Query().Get(constants.GitBranch)
	u.RawQuery = ""
	git.Url = u.String()

	return nil
}

func (oci *Oci) FromString(ociStr string) error {
	if oci == nil {
		return fmt.Errorf("oci source is nil")
	}

	u, err := url.Parse(ociStr)
	if err != nil {
		return err
	}

	if u.Scheme != constants.OciScheme {
		return fmt.Errorf("invalid oci url with schema: %s", u.Scheme)
	}

	oci.Reg = u.Host
	oci.Repo = strings.TrimPrefix(u.Path, "/")
	oci.Tag = u.Query().Get(constants.Tag)

	return nil
}

func (local *Local) FromString(localStr string) error {
	if local == nil {
		return fmt.Errorf("local source is nil")
	}

	local.Path = localStr
	return nil
}

// default::oci://ghcr.io/kcl-lang/k8s?name=k8s?version=0.1.1
func (registry *Registry) FromString(registryStr string) error {
	if registry == nil {
		return fmt.Errorf("registry is nil")
	}

	registryUrl, err := url.Parse(registryStr)
	if err != nil {
		return err
	}

	if registryUrl.Scheme != constants.DefaultOciScheme {
		return fmt.Errorf("invalid registry url with schema: %s", registryUrl.Scheme)
	}

	oci := &Oci{}
	registryUrl.Scheme = constants.OciScheme
	err = oci.FromString(registryUrl.String())
	if err != nil {
		return err
	}

	registry.Name = registryUrl.Query().Get("name")
	registry.Version = registryUrl.Query().Get("version")
	registry.Oci = oci

	return nil
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

// Deprecated: Use ToString instead
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

func ParseSourceUrlFrom(sourceStr string, settings *settings.Settings) (*url.URL, error) {
	regOpts, err := opt.NewRegistryOptionsFrom(sourceStr, settings)
	if err != nil {
		return nil, err
	}

	var url url.URL
	query := url.Query()

	if regOpts.Git != nil {
		url, err := url.Parse(regOpts.Git.Url)
		if err != nil {
			return nil, err
		}

		if url.Scheme != constants.GitScheme && url.Scheme != constants.SshScheme {
			url.Scheme = constants.GitScheme
		}
		url.RawQuery = query.Encode()
		return url, nil
	} else if regOpts.Oci != nil {
		url.Scheme = constants.OciScheme
		url.Host = regOpts.Oci.Reg
		url.Path = regOpts.Oci.Repo
		if regOpts.Oci.Tag != "" {
			query.Add(constants.Tag, regOpts.Oci.Tag)
		}
		url.RawQuery = query.Encode()
		return &url, nil
	} else if regOpts.Registry != nil {
		url.Scheme = constants.DefaultOciScheme
		url.Host = regOpts.Registry.Reg
		url.Path = regOpts.Registry.Repo
		if regOpts.Registry.Tag != "" {
			query.Add(constants.Tag, regOpts.Registry.Tag)
		}
		url.RawQuery = query.Encode()
		return &url, nil
	} else if regOpts.Local != nil {
		url.Path = regOpts.Local.Path
		return &url, nil
	}
	return nil, fmt.Errorf("invalid source url: %s", sourceStr)
}
