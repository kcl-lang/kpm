// Copyright 2023 The KCL Authors. All rights reserved.

package opt

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/path"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
)

// CompileOptions is the input options of 'kpm run'.
type CompileOptions struct {
	isVendor        bool
	hasSettingsYaml bool
	entries         []string
	noSumCheck      bool
	// Add a writer to control the output of the compiler.
	writer io.Writer
	*kcl.Option
}

type Option func(*CompileOptions)

// WithKclOption will add a kcl option to the compiler.
func WithKclOption(opt kcl.Option) Option {
	return func(opts *CompileOptions) {
		opts.Merge(opt)
	}
}

// WithEntries will add entries to the compiler.
func WithEntries(entries []string) Option {
	return func(opts *CompileOptions) {
		opts.entries = append(opts.entries, entries...)
	}
}

// WithEntry will add an entry to the compiler.
func WithVendor(isVendor bool) Option {
	return func(opts *CompileOptions) {
		opts.isVendor = isVendor
	}
}

// WithNoSumCheck will set the 'no_sum_check' flag.
func WithNoSumCheck(is bool) Option {
	return func(opts *CompileOptions) {
		opts.noSumCheck = is
	}
}

// WithLogWriter will set the log writer of the compiler.
func WithLogWriter(writer io.Writer) Option {
	return func(opts *CompileOptions) {
		opts.writer = writer
	}
}

// DefaultCompileOptions returns a default CompileOptions.
func DefaultCompileOptions() *CompileOptions {
	return &CompileOptions{
		writer: os.Stdout,
		Option: kcl.NewOption(),
	}
}

// SetNoSumCheck will set the 'no_sum_check' flag.
func (opts *CompileOptions) SetNoSumCheck(noSumCheck bool) {
	opts.noSumCheck = noSumCheck
}

// NoSumCheck will return the 'no_sum_check' flag.
func (opts *CompileOptions) NoSumCheck() bool {
	return opts.noSumCheck
}

// AddEntry will add a compile entry file to the compiler.
func (opts *CompileOptions) AddEntry(entry string) {
	opts.entries = append(opts.entries, entry)
}

// SetLogWriter will set the log writer of the compiler.
func (opts *CompileOptions) SetLogWriter(writer io.Writer) {
	opts.writer = writer
}

// Entrirs will return the entries of the compiler.
func (opts *CompileOptions) Entries() []string {
	return opts.entries
}

// ExtendEntries will extend the entries of the compiler.
func (opts *CompileOptions) ExtendEntries(entries []string) {
	opts.entries = append(opts.entries, entries...)
}

// SetEntries will set the entries of the compiler.
func (opts *CompileOptions) SetEntries(entries []string) {
	opts.entries = entries
}

// SetHasSettingsYaml will set the 'hasSettingsYaml' flag.
func (opts *CompileOptions) SetHasSettingsYaml(hasSettingsYaml bool) {
	opts.hasSettingsYaml = hasSettingsYaml
}

// HasSettingsYaml will return the 'hasSettingsYaml' flag.
func (opts *CompileOptions) HasSettingsYaml() bool {
	return opts.hasSettingsYaml
}

// SetVendor will set the 'isVendor' flag.
func (opts *CompileOptions) SetVendor(isVendor bool) {
	opts.isVendor = isVendor
}

// SetPackage will set the 'package' flag.
func (opts *CompileOptions) SetPackage(pkg string) {
	opts.Package = pkg
}

// IsVendor will return the 'isVendor' flag.
func (opts *CompileOptions) IsVendor() bool {
	return opts.isVendor
}

// PkgPath will return the home path for a kcl package during compilation
func (opts *CompileOptions) PkgPath() string {
	return opts.WorkDir
}

// SetPkgPath will set the home path for a kcl package during compilation
func (opts *CompileOptions) SetPkgPath(pkgPath string) {
	opts.Merge(kcl.WithWorkDir(pkgPath))
}

// LogWriter will return the log writer of the compiler.
func (opts *CompileOptions) LogWriter() io.Writer {
	return opts.writer
}

// Input options of 'kpm init'.
type InitOptions struct {
	Name     string
	InitPath string
	Version  string
}

func (opts *InitOptions) Validate() error {
	if len(opts.Name) == 0 {
		return errors.InvalidInitOptions
	} else if len(opts.InitPath) == 0 {
		return errors.InternalBug
	}
	if opts.Version != "" {
		if _, err := version.NewSemver(opts.Version); err != nil {
			return errors.InvalidVersionFormat
		}
	}

	return nil
}

type AddOptions struct {
	LocalPath    string
	NewPkgName   string
	RegistryOpts RegistryOptions
	NoSumCheck   bool
}

func (opts *AddOptions) Validate() error {
	if len(opts.LocalPath) == 0 {
		return errors.InternalBug
	} else if opts.RegistryOpts.Git != nil {
		return opts.RegistryOpts.Git.Validate()
	} else if opts.RegistryOpts.Oci != nil {
		return opts.RegistryOpts.Oci.Validate()
	} else if opts.RegistryOpts.Local != nil {
		return opts.RegistryOpts.Local.Validate()
	}
	return nil
}

type RegistryOptions struct {
	Git      *GitOptions
	Oci      *OciOptions
	Local    *LocalOptions
	Registry *OciOptions
}

// NewRegistryOptionsFrom will parse the registry options from oci url, oci ref and git url.
// If you do not know the url of the package is git or oci, you can use this function to parse the options.
// By default:
// 'git://', "https://", "http://" will be parsed as git options.
// 'oci://', will be parsed as oci options.
// 'file://' or a file path will be parsed as local options.
//
// If you know the url is git or oci, you can use 'NewGitOptionsFromUrl' or 'NewOciOptionsFromUrl' to parse the options.
// 'oci' and 'http', 'https' are supported for 'NewOciOptionsFromUrl'.
// 'git', 'http', 'https', 'ssh' are supported for 'NewGitOptionsFromUrl'.
func NewRegistryOptionsFrom(rawUrlorOciRef string, settings *settings.Settings) (*RegistryOptions, error) {
	parsedUrl, err := url.Parse(rawUrlorOciRef)
	if err != nil {
		return nil, err
	}

	// parse the options from the local file path
	if parsedUrl.Scheme == "" || parsedUrl.Scheme == constants.FileEntry {
		localOptions, err := NewLocalOptionsFromUrl(parsedUrl)
		if localOptions != nil && err == (*reporter.KpmEvent)(nil) {
			return &RegistryOptions{
				Local: localOptions,
			}, nil
		}
	}

	// parse the options from the git url
	// https, http, git and ssh are supported
	if parsedUrl.Scheme == constants.GitScheme ||
		parsedUrl.Scheme == constants.HttpScheme ||
		parsedUrl.Scheme == constants.HttpsScheme ||
		parsedUrl.Scheme == constants.SshScheme {
		gitOptions := NewGitOptionsFromUrl(parsedUrl)

		if gitOptions != nil {
			return &RegistryOptions{
				Git: gitOptions,
			}, nil
		}
	}

	// parse the options from the oci url
	// oci is supported
	if parsedUrl.Scheme == constants.OciScheme ||
		parsedUrl.Scheme == constants.HttpScheme ||
		parsedUrl.Scheme == constants.HttpsScheme {
		ociOptions := NewOciOptionsFromUrl(parsedUrl)

		if ociOptions != nil {
			return &RegistryOptions{
				Oci: ociOptions,
			}, nil
		}
	}

	// If all the url are invalid, try to parse the options from the oci ref.
	ociOptions, err := NewOciOptionsFromRef(rawUrlorOciRef, settings)
	if err != nil {
		return nil, err
	}

	if ociOptions != nil {
		return &RegistryOptions{
			Registry: ociOptions,
		}, nil
	}

	return nil, fmt.Errorf("invalid dependencies source: %s", rawUrlorOciRef)
}

// NewGitOptionsFromUrl will parse the git options from the git url.
// https, http, git and ssh are supported.
func NewGitOptionsFromUrl(parsedUrl *url.URL) *GitOptions {
	if parsedUrl.Scheme == "" || parsedUrl.Scheme == constants.GitScheme {
		// go-getter do not supports git scheme, so we need to convert it to https scheme.
		parsedUrl.Scheme = constants.HttpsScheme
	}

	commit := parsedUrl.Query().Get(constants.GitCommit)
	branch := parsedUrl.Query().Get(constants.GitBranch)
	tag := parsedUrl.Query().Get(constants.Tag)

	// clean the query in git url
	parsedUrl.RawQuery = ""
	url := parsedUrl.String()

	return &GitOptions{
		Url:    url,
		Branch: branch,
		Commit: commit,
		Tag:    tag,
	}
}

// NewOciOptionsFromUrl will parse the oci options from the oci url.
// https, http, oci is supported.
func NewOciOptionsFromUrl(parsedUrl *url.URL) *OciOptions {
	if parsedUrl.Scheme == "" {
		parsedUrl.Scheme = constants.HttpsScheme
	}
	return &OciOptions{
		Reg:     parsedUrl.Host,
		Repo:    parsedUrl.Path,
		Tag:     parsedUrl.Query().Get(constants.Tag),
		PkgName: filepath.Base(parsedUrl.Path),
	}
}

// NewOciOptionsFromRef will parse the oci options from the oci ref.
// The ref should be in the format of 'repoName:repoTag'.
func NewOciOptionsFromRef(refStr string, settings *settings.Settings) (*OciOptions, error) {
	reg := settings.DefaultOciRegistry()
	repo := settings.DefaultOciRepo()
	tag := ""

	ref, err := registry.ParseReference(refStr)
	if err != nil {
		var pkgName string
		pkgName, tag, err = ParseOciPkgNameAndVersion(refStr)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(pkgName, "/") {
			repo = fmt.Sprintf("%s/%s", repo, pkgName)
		} else {
			repo = fmt.Sprintf("%s%s", repo, pkgName)
		}
	} else {
		reg = ref.Registry
		repo = ref.Repository
		tag = ref.ReferenceOrDefault()
	}

	return &OciOptions{
		Reg:     reg,
		Repo:    repo,
		Tag:     tag,
		PkgName: filepath.Base(repo),
	}, nil
}

// NewLocalOptionsFromUrl will parse the local options from the local path.
// scheme 'file' and only path is supported.
func NewLocalOptionsFromUrl(parsedUrl *url.URL) (*LocalOptions, error) {
	return ParseLocalPathOptions(parsedUrl.Path)
}

// parseOciPkgNameAndVersion will parse package name and version
// from string "<pkg_name>:<pkg_version>".
func ParseOciPkgNameAndVersion(s string) (string, string, error) {
	parts := strings.Split(s, ":")
	if len(parts) == 1 {
		return parts[0], "", nil
	}

	if len(parts) > 2 {
		return "", "", reporter.NewErrorEvent(reporter.InvalidPkgRef, fmt.Errorf("invalid oci package reference '%s'", s))
	}

	if parts[1] == "" {
		return "", "", reporter.NewErrorEvent(reporter.InvalidPkgRef, fmt.Errorf("invalid oci package reference '%s'", s))
	}

	return parts[0], parts[1], nil
}

// ParseLocalPathOptions will parse the local path information from user cli inputs.
func ParseLocalPathOptions(localPath string) (*LocalOptions, *reporter.KpmEvent) {
	if localPath == "" {
		return nil, reporter.NewErrorEvent(reporter.PathIsEmpty, errors.PathIsEmpty)
	}
	// check if the local path exists.
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil, reporter.NewErrorEvent(reporter.LocalPathNotExist, err)
	} else {
		return &LocalOptions{
			Path: localPath,
		}, nil
	}
}

type GitOptions struct {
	Url    string
	Branch string
	Commit string
	Tag    string
}

func (opts *GitOptions) Validate() error {
	if len(opts.Url) == 0 {
		return reporter.NewErrorEvent(reporter.InvalidGitUrl, errors.InvalidAddOptionsInvalidGitUrl)
	}
	return nil
}

// OciOptions for download oci packages.
// kpm will download packages from oci registry by '{Reg}/{Repo}/{PkgName}:{Tag}'.
type OciOptions struct {
	Reg         string
	Repo        string
	Tag         string
	PkgName     string
	Annotations map[string]string
}

func (opts *OciOptions) Validate() error {
	if len(opts.Repo) == 0 {
		return reporter.NewErrorEvent(reporter.InvalidRepo, errors.InvalidAddOptionsInvalidOciRepo)
	}
	return nil
}

// LocalOptions for local packages.
// kpm will find packages from local path.
type LocalOptions struct {
	Path string
}

func (opts *LocalOptions) Validate() error {
	if len(opts.Path) == 0 {
		return errors.PathIsEmpty
	}
	if _, err := os.Stat(opts.Path); err != nil {
		return err
	}
	return nil
}

// ParseOciOptionFromOciUrl will parse oci url into an 'OciOptions'.
// If the 'tag' is empty, ParseOciOptionFromOciUrl will take 'latest' as the default tag.
func ParseOciOptionFromOciUrl(url, tag string) (*OciOptions, *reporter.KpmEvent) {
	ociOpt, err := ParseOciUrl(url)
	if err != nil {
		return nil, err
	}
	ociOpt.Tag = tag
	return ociOpt, nil
}

// ParseOciUrl will parse 'oci://hostName/repoName:repoTag' into OciOptions without tag.
func ParseOciUrl(ociUrl string) (*OciOptions, *reporter.KpmEvent) {
	u, err := url.Parse(ociUrl)
	if err != nil {
		return nil, reporter.NewEvent(reporter.IsNotUrl)
	}

	if u.Scheme != "oci" {
		return nil, reporter.NewEvent(reporter.UrlSchemeNotOci)
	}

	return &OciOptions{
		Reg:  u.Host,
		Repo: u.Path,
	}, nil
}

// AddStoragePathSuffix will take 'Registry/Repo/Tag' as a path suffix.
// e.g. Take '/usr/test' as input,
// and oci options is
//
//	OciOptions {
//	  Reg: 'docker.io',
//	  Repo: 'test/testRepo',
//	  Tag: 'v0.0.1'
//	}
//
// You will get a path '/usr/test/docker.io/test/testRepo/v0.0.1'.
// Deprecated: This function will be deprecated, use 'SanitizePathWithSuffix' instead.
func (oci *OciOptions) AddStoragePathSuffix(pathPrefix string) string {
	return filepath.Join(filepath.Join(filepath.Join(pathPrefix, oci.Reg), oci.Repo), oci.Tag)
}

// SanitizePathSuffix will take 'Registry/Repo/Tag' as a path suffix and sanitize it.
func (oci *OciOptions) SanitizePathWithSuffix(pathPrefix string) string {
	return path.SanitizePath(filepath.Join(filepath.Join(filepath.Join(pathPrefix, oci.Reg), oci.Repo), oci.Tag))
}

type OciManifestOptions struct {
	Annotations map[string]string
}

// OciFetchOptions is the input options of the api to fetch oci manifest.
type OciFetchOptions struct {
	oras.FetchBytesOptions
	OciOptions
}
