package client

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/utils"
)

// RunOptions contains the options for running a kcl package.
type RunOptions struct {
	// CompileOptions is the options for kcl compiler.
	CompileOptions *opt.CompileOptions
	// Sources is the sources of the package.
	// It can be a local *.k path, a local *.tar path, a local directory, a remote git/oci path,.
	Sources []*downloader.Source
}

// Only one kcl module can be compiled at a time.
// So, the sources must have the same root path.
func (ro *RunOptions) Validate() error {
	if len(ro.Sources) == 0 {
		return errors.New("no source provided")
	}

	// More than one source, all sources must have the same root path.
	if len(ro.Sources) > 1 {
		rootPath, err := ro.Sources[0].FindRootPath()
		if err != nil {
			return err
		}
		for _, source := range ro.Sources {
			// By now, each remote path is a kcl module.
			// And, only one remote path is allowed.
			// So when more than one remote path or local path and remote path are provided, return an error.
			if (!source.IsLocalPath() || source.IsLocalTarPath()) && len(ro.Sources) > 1 {
				return errors.New("only one kcl module root path is allowed")
			}

			tmpRootPath, err := source.FindRootPath()
			if err != nil {
				return err
			}
			if tmpRootPath != rootPath {
				return errors.New("only one kcl module root path is allowed: root path conflicts between " + rootPath + " with " + tmpRootPath)
			}
		}
	}

	return nil
}

type RunOption func(*RunOptions) error

func WithRunSources(sources []*downloader.Source) RunOption {
	return func(ro *RunOptions) error {
		ro.Sources = sources
		return nil
	}
}

func WithSource(source *downloader.Source) RunOption {
	return func(ro *RunOptions) error {
		ro.Sources = append(ro.Sources, source)
		return nil
	}
}

func WithEntries(entries []string) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}
		ro.CompileOptions.ExtendEntries(entries)
		return nil
	}
}

func WithSettingFiles(settingFiles []string) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}
		for _, settingFile := range settingFiles {
			ro.CompileOptions.Merge(kcl.WithSettings(settingFile))
		}
		ro.CompileOptions.SetHasSettingsYaml(true)
		return nil
	}
}

func WithArguments(args []string) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}

		ro.CompileOptions.Merge(kcl.WithOptions(args...))

		return nil
	}
}

func WithOverrides(overrides []string, debug bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}

		ro.CompileOptions.Merge(kcl.WithOverrides(overrides...))
		ro.CompileOptions.PrintOverrideAst = debug

		return nil
	}
}

func WithPathSelectors(pathSelectors []string) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}

		ro.CompileOptions.Merge(kcl.WithSelectors(pathSelectors...))

		return nil
	}
}

func WithDebug(debug bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}

		if debug {
			ro.CompileOptions.Debug = 1
		}

		return nil
	}
}

func WithDisableNone(disableNone bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}

		if disableNone {
			ro.CompileOptions.Merge(kcl.WithDisableNone(disableNone))
		}

		return nil
	}
}

func WithExternalPkgs(externalPkgs []string) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}

		ro.CompileOptions.Merge(kcl.WithExternalPkgs(externalPkgs...))

		return nil
	}
}

func WithSortKeys(sortKeys bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}

		if sortKeys {
			ro.CompileOptions.Merge(kcl.WithSortKeys(sortKeys))
		}

		return nil
	}
}

func WithShowHidden(showHidden bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}

		if showHidden {
			ro.CompileOptions.Merge(kcl.WithShowHidden(showHidden))
		}

		return nil
	}
}

func WithStrictRange(strictRange bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}

		if strictRange {
			ro.CompileOptions.StrictRangeCheck = strictRange
		}

		return nil
	}
}

func WithCompileOnly(compileOnly bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}

		if compileOnly {
			ro.CompileOptions.CompileOnly = compileOnly
		}

		return nil
	}
}

func WithVendor(vendor bool) RunOption {
	return func(ro *RunOptions) error {
		if ro.CompileOptions == nil {
			ro.CompileOptions = opt.DefaultCompileOptions()
		}

		if vendor {
			ro.CompileOptions.SetVendor(vendor)
		}

		return nil
	}
}

func (c *KpmClient) Run(options ...RunOption) (*kcl.KCLResultList, error) {
	opts := &RunOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	err := opts.Validate()
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, err
	}
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	var pkgRootPath string
	var entries []string

	// Only one source
	if len(opts.Sources) == 1 {
		// download the package from remote source and get the package root path
		source := opts.Sources[0]
		if !source.IsLocalPath() {
			kpkg, err := c.downloadPkg(
				downloader.WithLocalPath(tmpDir),
				downloader.WithSource(*source),
			)
			if err != nil {
				return nil, err
			}
			pkgRootPath = kpkg.HomePath
		} else if source.IsLocalTarPath() {
			// Untar the package to the temp dir and get the package root path
			sourcePath, err := source.ToFilePath()
			if err != nil {
				return nil, err
			}
			err = utils.UnTarDir(sourcePath, tmpDir)
			if err != nil {
				return nil, err
			}
			pkgRootPath = tmpDir
		} else {
			// get the package root path
			pkgRootPath, err = source.FindRootPath()
			if err != nil {
				return nil, err
			}
			sourcePath, err := source.ToFilePath()
			if err != nil {
				return nil, err
			}
			sourcePath, err = filepath.Abs(sourcePath)
			if err != nil {
				return nil, err
			}
			entries = append(entries, sourcePath)
		}
	} else {
		// More than one source，all sources must have the same root path.
		for _, source := range opts.Sources {
			sourceRootPath, err := source.FindRootPath()
			if err != nil {
				return nil, err
			}

			if pkgRootPath == "" {
				pkgRootPath = sourceRootPath
			}

			if pkgRootPath != sourceRootPath {
				return nil, fmt.Errorf("cannot compile multiple packages at the same time")
			}

			sourcePath, err := source.ToFilePath()
			if err != nil {
				return nil, err
			}
			entries = append(entries, sourcePath)
		}
	}

	// If the kcl package root path does not have a kcl.mod file, create a virtual one.
	vKclModPath := filepath.Join(pkgRootPath, pkg.MOD_FILE)
	vKclModLockPath := filepath.Join(pkgRootPath, pkg.MOD_LOCK_FILE)

	// remove the log writer when create the virtual mod file, err is enough.
	// TODO: debug mode should record this log.
	logWriter := c.GetLogWriter()

	if !utils.DirExists(vKclModPath) {
		// clean the virtual mod file and lock file.
		defer func() {
			if err := os.Remove(vKclModPath); err != nil {
				log.Printf("Failed to remove %s: %v", vKclModPath, err)
			}
			if err := os.Remove(vKclModLockPath); err != nil {
				log.Printf("Failed to remove %s: %v", vKclModLockPath, err)
			}
		}()

		initOpts := opt.InitOptions{
			Name:     "vPkg_" + uuid.New().String(),
			InitPath: pkgRootPath,
		}

		kclPkg := pkg.NewKclPkg(&initOpts)

		c.logWriter = nil
		err := c.createIfNotExist(kclPkg.ModFile.GetModFilePath(), kclPkg.ModFile.StoreModFile)
		if err != nil {
			return nil, err
		}

		err = c.createIfNotExist(kclPkg.ModFile.GetModLockFilePath(), kclPkg.LockDepsVersion)
		if err != nil {
			return nil, err
		}
	}
	c.logWriter = logWriter
	// Set the package root path
	opts.CompileOptions.SetPkgPath(pkgRootPath)
	// Set the entries
	opts.CompileOptions.ExtendEntries(entries)
	return c.CompileWithOpts(opts.CompileOptions)
}
