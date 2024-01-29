// Copyright 2023 The KCL Authors. All rights reserved.

package opt

import (
	"io"
	"net/url"
	"os"
	"path/filepath"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/reporter"
	"oras.land/oras-go/v2"
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
}

func (opts *InitOptions) Validate() error {
	if len(opts.Name) == 0 {
		return errors.InvalidInitOptions
	} else if len(opts.InitPath) == 0 {
		return errors.InternalBug
	}
	return nil
}

type AddOptions struct {
	LocalPath    string
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
	Git   *GitOptions
	Oci   *OciOptions
	Local *LocalOptions
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
func (oci *OciOptions) AddStoragePathSuffix(pathPrefix string) string {
	return filepath.Join(filepath.Join(filepath.Join(pathPrefix, oci.Reg), oci.Repo), oci.Tag)
}

type OciManifestOptions struct {
	Annotations map[string]string
}

// OciFetchOptions is the input options of the api to fetch oci manifest.
type OciFetchOptions struct {
	oras.FetchBytesOptions
	OciOptions
}
