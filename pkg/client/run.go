/*
This file provides the api for running a kcl package.
You can use `RunOptions` to set the options for running a kcl package.
Before running a kcl package, you should create a instance for `KpmClient`.

```go
kpmcli := client.NewKpmClient()
```

You can use the method `Run` for `KpmClient` with options `RunOptions` to run a kcl package.

```go
kpmcli.Run(
	WithWorkDir("path/to/workdir"),
	WithRunSourceUrl("path/to/source"),
)
```

You can set the KCL package sources by `WithRunSourceUrls` or `WithRunSourceUrl`.
The KCL package sources include the local path, the remote git/oci path, etc.
For remote git/oci path, you can set the package sources

```go
// The KCL package sources are the remote git repo.
kpmcli.Run(
	WithRunSourceUrl("git://github.com/test/test.git"),
)

// The KCL package sources are the remote oci registry.
kpmcli.Run(
	WithRunSourceUrl("oci://ghcr.com/test/test"),
)
```

For local paths, you can set multiple *.k files or directories.
likg:

```go
kpmcli.Run(
	WithRunSourceUrl("local/usr/test1/main.k"),
	WithRunSourceUrl("local/usr/test2/base.k"),
	WithRunSourceUrl("local/usr/test3/"),
)
```

For the source above, `kpmcli.Run()` will do :
1.find a package root path from the sources.
2.load the package from the package root path.
3.take all the sources as the compile entry to compile the package.

NOTE: `kpmcli.Run()` do not support compiling multiple packages at the same time. so, all the sources should belong to the same package root path.

`kpmcli.Run()` will iterate all the sources and find the source root path.
For source `local/usr/test1/main.k`, `kpmcli.Run()` will start from the path `local/usr/test1` and iterate all the parent directories.

If `kcl.mod` are found, the path of `kcl.mod` will be used as the source root path.
If `kcl.mod` are not found, the path of the source will be used as the source root path.

So if the kcl.mod is located in the path `local/usr/`, `kpmcli.Run()` will load package from `local/usr/` and load dependencies from `local/usr/kcl.mod`.
And take all the KCL program files in the sources as the compile entry to compile the package.
*/

package client

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/runner"
	"kcl-lang.io/kpm/pkg/utils"
)

// RunOptions contains the options for running a kcl package.
type RunOptions struct {
	settingYamlFiles []string
	vendor           bool
	// Sources is the sources of the package.
	// It can be a local *.k path, a local *.tar/*.tgz path, a local directory, a remote git/oci path,.
	Sources []*downloader.Source
	*kcl.Option
}

type RunOption func(*RunOptions) error

func WithLogger(l io.Writer) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		ro.Merge(kcl.WithLogger(l))
		return nil
	}
}

// WithWorkDir sets the work directory for running the kcl package.
func WithWorkDir(workDir string) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		ro.WorkDir = workDir
		return nil
	}
}

// WithRunSources sets the sources for running the kcl package.
func WithRunSources(sources []*downloader.Source) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		ro.Sources = sources
		return nil
	}
}

// WithRunSources sets the source for running the kcl package.
func WithRunSource(source *downloader.Source) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		if ro.Sources == nil {
			ro.Sources = make([]*downloader.Source, 0)
		}
		ro.Sources = append(ro.Sources, source)
		return nil
	}
}

// WithRunSourceUrls sets the source urls for running the kcl package.
func WithRunSourceUrls(sourceUrls []string) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		var sources []*downloader.Source
		for _, sourceUrl := range sourceUrls {
			source, err := downloader.NewSourceFromStr(sourceUrl)
			if err != nil {
				return err
			}
			sources = append(sources, source)
		}
		ro.Sources = sources
		return nil
	}
}

// WithRunSourceUrl sets the source url for running the kcl package.
func WithRunSourceUrl(sourceUrl string) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		if ro.Sources == nil {
			ro.Sources = make([]*downloader.Source, 0)
		}
		source, err := downloader.NewSourceFromStr(sourceUrl)
		if err != nil {
			return err
		}
		ro.Sources = append(ro.Sources, source)
		return nil
	}
}

// WithSettingFiles sets the setting files for running the kcl package.
func WithSettingFiles(settingFiles []string) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		ro.settingYamlFiles = settingFiles
		return nil
	}
}

// WithArguments sets the arguments for running the kcl package.
func WithArguments(args []string) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		ro.Merge(kcl.WithOptions(args...))

		return nil
	}
}

// WithOverrides sets the overrides for running the kcl package.
func WithOverrides(overrides []string, debug bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		ro.Merge(kcl.WithOverrides(overrides...))
		ro.PrintOverrideAst = debug
		return nil
	}
}

// WithPathSelectors sets the path selectors for running the kcl package.
func WithPathSelectors(pathSelectors []string) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		ro.Merge(kcl.WithSelectors(pathSelectors...))
		return nil
	}
}

// WithDebug sets the debug mode for running the kcl package.
func WithDebug(debug bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		if debug {
			ro.Debug = 1
		}
		return nil
	}
}

// WithDisableNone sets the disable none mode for running the kcl package.
func WithDisableNone(disableNone bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		if disableNone {
			ro.Merge(kcl.WithDisableNone(disableNone))
		}
		return nil
	}
}

// WithExternalPkgs sets the external packages for running the kcl package.
func WithExternalPkgs(externalPkgs []string) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		ro.Merge(kcl.WithExternalPkgs(externalPkgs...))
		return nil
	}
}

// WithSortKeys sets the sort keys for running the kcl package.
func WithSortKeys(sortKeys bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		if sortKeys {
			ro.Merge(kcl.WithSortKeys(sortKeys))
		}

		return nil
	}
}

// WithShowHidden sets the show hidden mode for running the kcl package.
func WithShowHidden(showHidden bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		if showHidden {
			ro.Merge(kcl.WithShowHidden(showHidden))
		}

		return nil
	}
}

// WithStrictRange sets the strict range mode for running the kcl package.
func WithStrictRange(strictRange bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		if strictRange {
			ro.StrictRangeCheck = strictRange
		}

		return nil
	}
}

// WithCompileOnly sets the compile only mode for running the kcl package.
func WithCompileOnly(compileOnly bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		if compileOnly {
			ro.CompileOnly = compileOnly
		}

		return nil
	}
}

// WithVendor sets the vendor mode for running the kcl package.
func WithVendor(vendor bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.Option == nil {
			ro.Option = kcl.NewOption()
		}
		if vendor {
			ro.vendor = vendor
		}

		return nil
	}
}

// applyCompileOptionsFromYaml applies the compile options from the kcl.yaml file.
func (o *RunOptions) applyCompileOptionsFromYaml(workdir string) bool {
	succeed := false
	// load the kcl.yaml from cli
	if len(o.settingYamlFiles) != 0 {
		// If kcl.yaml is set by cli, the kfile list will override, so clean the kfile list.
		o.KFilenameList = []string{}
		for _, settingYamlFile := range o.settingYamlFiles {
			o.Merge(kcl.WithSettings(settingYamlFile))
			succeed = true
		}
	} else {
		// load the kcl.yaml from the workdir
		// If the workdir is not empty, try to find the settings.yaml file in the workdir.
		settingsYamlPath := filepath.Join(workdir, constants.KCL_YAML)
		if utils.DirExists(settingsYamlPath) {
			// If kcl.yaml is set by cli, the kfile list will override, so clean the kfile list.
			o.KFilenameList = []string{}
			o.Merge(kcl.WithSettings(settingsYamlPath))
			succeed = true
		}
	}

	// transform the relative path to the absolute path in kcl.yaml by workdir
	var updatedKFilenameList []string
	for _, kfile := range o.KFilenameList {
		if !filepath.IsAbs(kfile) && !utils.IsModRelativePath(kfile) {
			kfile = filepath.Join(workdir, kfile)
		}
		updatedKFilenameList = append(updatedKFilenameList, kfile)
	}
	o.KFilenameList = updatedKFilenameList

	return succeed
}

// applyCompileOptionsFromKclMod applies the compile options from the kcl.mod file.
func (o *RunOptions) applyCompileOptionsFromKclMod(kclPkg *pkg.KclPkg) bool {
	o.Merge(*kclPkg.GetKclOpts())

	var updatedKFilenameList []string
	// transform the relative path to the absolute path in kcl.yaml by kcl.mod path
	for _, kfile := range o.KFilenameList {
		if !filepath.IsAbs(kfile) && !utils.IsModRelativePath(kfile) {
			kfile = filepath.Join(kclPkg.HomePath, kfile)
		}
		updatedKFilenameList = append(updatedKFilenameList, kfile)
	}
	o.KFilenameList = updatedKFilenameList

	return len(o.KFilenameList) != 0
}

// applyCompileOptions applies the compile options from cli, kcl.yaml and kcl.mod.
func (o *RunOptions) applyCompileOptions(compiledKfiles []string, kclPkg *pkg.KclPkg, workDir string) error {
	o.Merge(kcl.WithWorkDir(workDir))

	// The compile options from cli
	cliopts := o.Option

	// Create an empty compile option to get the options from kcl.mod and kcl.yaml
	o.Option = kcl.NewOption()

	// Load the compile options from kcl.mod
	loadmod := o.applyCompileOptionsFromKclMod(kclPkg)
	// Load the compile options from kcl.yaml, it will override the compile options from kcl.mod
	loadyaml := o.applyCompileOptionsFromYaml(workDir)

	// Finally, the compile options from cli will override the compile options from kcl.yaml and kcl.mod
	o.Merge(*cliopts)

	// The k files from cli will override kcl.yaml and kcl.mod
	if len(compiledKfiles) != 0 {
		var tmpKFilenameList []string
		for _, compiledKfile := range compiledKfiles {
			if !filepath.IsAbs(compiledKfile) {
				compiledKfile = filepath.Join(workDir, compiledKfile)
			}
			tmpKFilenameList = append(tmpKFilenameList, compiledKfile)
		}
		o.KFilenameList = tmpKFilenameList
	}

	// If there is no compiled k files set by cli, kcl.yaml and kcl.mod,
	// use the kcl package path as the compiled k files.
	if !loadmod && !loadyaml && len(o.KFilenameList) == 0 {
		o.KFilenameList = []string{kclPkg.HomePath}
	}

	return nil
}

// getPkgSource returns the package source.
// Compiling multiple packages at the same time will cause an error.
// If more than one source is set, `getPkgSource` will get the root path of the package.
// If no source is set, `getPkgSource` will find the root path of the package from the workdir.
func (o *RunOptions) getPkgSource() (*downloader.Source, error) {
	workDir := o.WorkDir

	var pkgSource *downloader.Source
	if len(o.Sources) == 0 {
		workDir, err := filepath.Abs(workDir)
		if err != nil {
			return nil, err
		}
		// If no sources set by options, return a localSource to facilitate searching for
		// compilation entries from configurations such as kcl.yaml and kcl.mod files.
		tmpSource := &downloader.Source{
			Local: &downloader.Local{
				Path: workDir,
			},
		}

		// PkgSource is the root path of the package.
		// WorkDir maybe not the root path of the package.
		// Find the root package path from WorkDir.
		pkgRootPath, err := tmpSource.FindRootPath()
		if err != nil {
			return nil, err
		}
		return downloader.NewSourceFromStr(pkgRootPath)
	} else {
		var rootPath string
		var err error
		for _, source := range o.Sources {
			if pkgSource == nil {
				pkgSource = source
				rootPath, err = source.FindRootPath()
				if err != nil {
					return nil, err
				}
			} else {
				rootSourceStr, err := pkgSource.ToString()
				if err != nil {
					return nil, err
				}

				sourceStr, err := source.ToString()
				if err != nil {
					return nil, err
				}

				if pkgSource.IsPackaged() || source.IsPackaged() {
					return nil, reporter.NewErrorEvent(
						reporter.CompileFailed,
						fmt.Errorf("cannot compile multiple packages %s at the same time", []string{rootSourceStr, sourceStr}),
						"only allows one package to be compiled at a time",
					)
				}

				if !pkgSource.IsPackaged() && !source.IsPackaged() {
					tmpRootPath, err := source.FindRootPath()
					if err != nil {
						return nil, err
					}
					if tmpRootPath != rootPath {
						return nil, reporter.NewErrorEvent(
							reporter.CompileFailed,
							fmt.Errorf("cannot compile multiple packages %s at the same time", []string{tmpRootPath, rootPath}),
							"only allows one package to be compiled at a time",
						)
					}
				}
			}
		}

		// A local k file or folder
		if pkgSource.IsLocalKPath() || pkgSource.IsDir() {
			rootPath, err = pkgSource.FindRootPath()
			if err != nil {
				return nil, err
			}

			pkgSource, err = downloader.NewSourceFromStr(rootPath)
			if err != nil {
				return nil, err
			}
		}
	}

	if pkgSource == nil {
		return nil, errors.New("no source provided")
	}

	return pkgSource, nil
}

// getCompileEntry returns the compile entry.
// Depending on the sources, the compile entry finds the package root path and the compiled files.
func (o *RunOptions) getCompileEntry() (*runner.Entry, error) {
	// 1. find the root path of the package by the sources or workdir.
	ps, err := o.getPkgSource()
	if err != nil {
		return nil, err
	}

	compiledFiles := make([]string, 0)
	if len(o.Sources) != 0 {
		// If the package root path is the first source,
		// the other sources are compiled files.
		if reflect.DeepEqual(ps, o.Sources[0]) {
			for i := 1; i < len(o.Sources); i++ {
				filePath, err := o.Sources[i].ToFilePath()
				if err != nil {
					return nil, err
				}
				compiledFiles = append(compiledFiles, filePath)
			}
		} else {
			// If the package root path not the first source,
			// all sources are compiled files.
			for _, source := range o.Sources {
				filePath, err := source.ToFilePath()
				if err != nil {
					return nil, err
				}
				compiledFiles = append(compiledFiles, filePath)
			}
		}
	} else {
		// If there is no source, workdir is the package source.
		// Compile the workdir.
		workDirSource, err := downloader.NewSourceFromStr(o.WorkDir)
		if err != nil {
			return nil, err
		}

		if !reflect.DeepEqual(ps, workDirSource) {
			compiledFiles = append(compiledFiles, o.WorkDir)
		}
	}

	return runner.NewEntry(
		runner.WithEntryFiles(compiledFiles),
		runner.WithPackageSource(ps),
	), nil
}

// Run runs the kcl package.
func (c *KpmClient) Run(options ...RunOption) (*kcl.KCLResultList, error) {
	opts := &RunOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	// Set the work directory to pwd if not set the workdir
	var err error
	if opts.WorkDir == "" {
		opts.WorkDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	// Find the package source, note is maybe local compile entries or paths that contains `kcl.mod`
	compileEntry, err := opts.getCompileEntry()
	if err != nil {
		return nil, err
	}

	// Visit the root package source.
	var res *kcl.KCLResultList
	err = NewVisitor(*compileEntry.GetPackageSource(), c).
		Visit(compileEntry.GetPackageSource(), func(kclPkg *pkg.KclPkg) error {
			// Apply the compile options from cli, kcl.yaml or kcl.mod
			err = opts.applyCompileOptions(compileEntry.EntryFiles(), kclPkg, opts.WorkDir)
			if err != nil {
				return err
			}

			kclPkg.SetVendorMode(opts.vendor)

			// Resolve and update the dependencies into a map.
			pkgMap, err := c.ResolveDepsIntoMap(kclPkg)
			if err != nil {
				return err
			}

			// Fill the dependency path.
			for dName, dPath := range pkgMap {
				if !filepath.IsAbs(dPath) {
					dPath = filepath.Join(c.homePath, dPath)
				}

				opts.Merge(kcl.WithExternalPkgs(fmt.Sprintf(constants.EXTERNAL_PKGS_ARG_PATTERN, dName, dPath)))
			}

			// Compile the kcl package.
			res, err = kcl.RunWithOpts(*opts.Option)
			if err != nil {
				return err
			}

			return nil
		})

	if err != nil {
		return nil, err
	}
	return res, nil
}
