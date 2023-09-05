package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/runner"
	"kcl-lang.io/kpm/pkg/utils"
)

// RunTar will compile the kcl package from a kcl package tar.
func RunTar(tarPath string, opts *opt.CompileOptions) (string, error) {
	// The directory after extracting the tar package is taken as the root directory of the package,
	// and kclvm is called to compile the kcl program under the 'destDir'.
	// e.g.
	// if the tar path is 'xxx/xxx/xxx/test.tar',
	// the 'xxx/xxx/xxx/test' will be taken as the root path of the kcl package to compile.
	compileResult, compileErr := RunTarPkg(tarPath, opts)

	if compileErr != nil {
		return "", compileErr
	}
	return compileResult.GetRawYamlResult(), nil
}

const KCL_PKG_TAR = "*.tar"

// RunOci will compile the kcl package from an OCI reference.
func RunOci(ociRef, version string, opts *opt.CompileOptions) (string, error) {
	compileResult, compileErr := RunOciPkg(ociRef, version, opts)

	if compileErr != nil {
		return "", compileErr
	}
	return compileResult.GetRawYamlResult(), nil
}

// RunPkg will compile current kcl package.
func RunPkg(opts *opt.CompileOptions) (string, error) {

	compileResult, err := RunCurrentPkg(opts)
	if err != nil {
		return "", err
	}

	return compileResult.GetRawYamlResult(), nil
}

// RunPkgInPath will load the 'KclPkg' from path 'pkgPath'.
// And run the kcl package with entry file in 'entryFilePath' in 'vendorMode'.
func RunPkgInPath(opts *opt.CompileOptions) (string, error) {
	// Call the kcl compiler.
	compileResult, err := RunPkgWithOpt(opts)
	if err != nil {
		return "", err
	}

	return compileResult.GetRawYamlResult(), nil
}

// CompileWithOpts will compile the kcl program without kcl package.
func CompileWithOpts(opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	if len(opts.Entries()) > 0 {
		for _, entry := range opts.Entries() {
			if filepath.IsAbs(entry) {
				opts.Merge(kcl.WithKFilenames(entry))
			} else {
				opts.Merge(kcl.WithKFilenames(filepath.Join(opts.PkgPath(), entry)))
			}
		}
	} else {
		// no entry
		opts.Merge(kcl.WithKFilenames(opts.PkgPath()))
	}
	opts.Merge(kcl.WithWorkDir(opts.PkgPath()))
	return kcl.RunWithOpts(*opts.Option)
}

// absTarPath checks whether path 'tarPath' exists and whether path 'tarPath' ends with '.tar'
// And after checking, absTarPath return the abs path for 'tarPath'.
func absTarPath(tarPath string) (string, error) {
	absTarPath, err := filepath.Abs(tarPath)
	if err != nil {
		return "", errors.InternalBug
	}

	if filepath.Ext(absTarPath) != ".tar" {
		return "", errors.InvalidKclPacakgeTar
	} else if !utils.DirExists(absTarPath) {
		return "", errors.KclPacakgeTarNotFound
	}

	return absTarPath, nil
}

// getAbsInputPath will return the abs path of the file path described by '--input'.
// If the path exists after 'inputPath' is computed as a full path, it will be returned.
// If not, the kpm checks whether the full path of 'pkgPath/inputPath' exists,
// If the full path of 'pkgPath/inputPath' exists, it will be returned.
// If not, getAbsInputPath returns 'entry file not found' error.
func getAbsInputPath(pkgPath string, inputPath string) (string, error) {
	absPath, err := filepath.Abs(filepath.Join(pkgPath, inputPath))
	if err != nil {
		return "", err
	}

	if utils.DirExists(absPath) {
		return absPath, nil
	}

	return "", errors.EntryFileNotFound
}

// RunPkgWithOpt will compile the kcl package with the compile options.
func RunPkgWithOpt(opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	pkgPath, err := filepath.Abs(opts.PkgPath())
	if err != nil {
		return nil, errors.InternalBug
	}

	kclPkg, err := pkg.LoadKclPkg(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("kpm: failed to load package, please check the package path '%s' is valid", pkgPath)
	}

	kclPkg.SetVendorMode(opts.IsVendor())

	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return nil, err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != (*reporter.KpmEvent)(nil) {
		return nil, err
	}

	if len(opts.Entries()) > 0 {
		// add entry from '--input'
		for _, entry := range opts.Entries() {
			if filepath.IsAbs(entry) {
				opts.Merge(kcl.WithKFilenames(entry))
			} else {
				opts.Merge(kcl.WithKFilenames(filepath.Join(opts.PkgPath(), entry)))
			}
		}
		// add entry from 'kcl.mod'
	} else if len(kclPkg.GetEntryKclFilesFromModFile()) > 0 {
		opts.Merge(*kclPkg.GetKclOpts())
	} else if !opts.HasSettingsYaml() {
		// no entry
		opts.Merge(kcl.WithKFilenames(opts.PkgPath()))
	}
	opts.Merge(kcl.WithWorkDir(opts.PkgPath()))

	// Calculate the absolute path of entry file described by '--input'.
	compiler := runner.NewCompilerWithOpts(opts)

	// Call the kcl compiler.
	compileResult, err := kclPkg.Compile(
		globalPkgPath,
		compiler,
	)

	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.CompileFailed, err, "failed to compile the kcl package")
	}

	return compileResult, nil
}

// RunCurrentPkg will compile the current kcl package.
func RunCurrentPkg(opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	pwd, err := os.Getwd()
	opts.SetPkgPath(pwd)

	if err != nil {
		reporter.ExitWithReport("kpm: internal bug: failed to load working directory")
	}

	return RunPkgWithOpt(opts)
}

// RunTarPkg will compile the kcl package from a kcl package tar.
func RunTarPkg(tarPath string, opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	absTarPath, err := absTarPath(tarPath)
	if err != nil {
		return nil, err
	}
	// Extract the tar package to a directory with the same name.
	// e.g.
	// 'xxx/xxx/xxx/test.tar' will be extracted to the directory 'xxx/xxx/xxx/test'.
	destDir := strings.TrimSuffix(absTarPath, filepath.Ext(absTarPath))
	err = utils.UnTarDir(absTarPath, destDir)
	if err != nil {
		return nil, err
	}

	opts.SetPkgPath(destDir)
	// The directory after extracting the tar package is taken as the root directory of the package,
	// and kclvm is called to compile the kcl program under the 'destDir'.
	// e.g.
	// if the tar path is 'xxx/xxx/xxx/test.tar',
	// the 'xxx/xxx/xxx/test' will be taken as the root path of the kcl package to compile.
	return RunPkgWithOpt(opts)
}

// RunOciPkg will compile the kcl package from an OCI reference.
func RunOciPkg(ociRef, version string, opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	ociOpts, err := opt.ParseOciOptionFromString(ociRef, version)

	if err != nil {
		return nil, err
	}

	// 1. Create the temporary directory to pull the tar.
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, errors.InternalBug
	}
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	localPath := ociOpts.AddStoragePathSuffix(tmpDir)

	// 2. Pull the tar.
	err = oci.Pull(localPath, ociOpts.Reg, ociOpts.Repo, ociOpts.Tag)

	if err != (*reporter.KpmEvent)(nil) {
		return nil, err
	}

	// 3.Get the (*.tar) file path.
	matches, err := filepath.Glob(filepath.Join(localPath, KCL_PKG_TAR))
	if err != nil || len(matches) != 1 {
		return nil, errors.FailedPull
	}

	return RunTarPkg(matches[0], opts)
}
