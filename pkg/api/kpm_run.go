package api

import (
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
	absTarPath, err := absTarPath(tarPath)
	if err != nil {
		return "", err
	}
	// Extract the tar package to a directory with the same name.
	// e.g.
	// 'xxx/xxx/xxx/test.tar' will be extracted to the directory 'xxx/xxx/xxx/test'.
	destDir := strings.TrimSuffix(absTarPath, filepath.Ext(absTarPath))
	err = utils.UnTarDir(absTarPath, destDir)
	if err != nil {
		return "", err
	}

	opts.SetPkgPath(destDir)
	// The directory after extracting the tar package is taken as the root directory of the package,
	// and kclvm is called to compile the kcl program under the 'destDir'.
	// e.g.
	// if the tar path is 'xxx/xxx/xxx/test.tar',
	// the 'xxx/xxx/xxx/test' will be taken as the root path of the kcl package to compile.
	compileResult, compileErr := RunPkgInPath(opts)

	if compileErr != nil {
		return "", compileErr
	}
	return compileResult, nil
}

const KCL_PKG_TAR = "*.tar"

// RunOci will compile the kcl package from an OCI reference.
func RunOci(ociRef, version string, opts *opt.CompileOptions) (string, error) {
	ociOpts, err := opt.ParseOciOptionFromString(ociRef, version)

	if err != nil {
		return "", err
	}

	// 1. Create the temporary directory to pull the tar.
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", errors.InternalBug
	}
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	localPath := ociOpts.AddStoragePathSuffix(tmpDir)

	// 2. Pull the tar.
	err = oci.Pull(localPath, ociOpts.Reg, ociOpts.Repo, ociOpts.Tag)

	if err != (*reporter.KpmEvent)(nil) {
		return "", err
	}

	// 3.Get the (*.tar) file path.
	matches, err := filepath.Glob(filepath.Join(localPath, KCL_PKG_TAR))
	if err != nil || len(matches) != 1 {
		return "", errors.FailedPull
	}

	return RunTar(matches[0], opts)
}

// RunPkg will compile current kcl package.
func RunPkg(opts *opt.CompileOptions) (string, error) {

	// If no tar packages specified by "--tar" to run
	// kpm will take the current directory ($PWD) as the root of the kcl package and compile.
	pwd, err := os.Getwd()
	opts.SetPkgPath(pwd)

	if err != nil {
		reporter.ExitWithReport("kpm: internal bug: failed to load working directory")
	}

	compileResult, err := RunPkgInPath(opts)
	if err != nil {
		return "", err
	}

	return compileResult, nil
}

// RunPkgInPath will load the 'KclPkg' from path 'pkgPath'.
// And run the kcl package with entry file in 'entryFilePath' in 'vendorMode'.
func RunPkgInPath(opts *opt.CompileOptions) (string, error) {

	pkgPath, err := filepath.Abs(opts.PkgPath())
	if err != nil {
		return "", errors.InternalBug
	}

	kclPkg, err := pkg.LoadKclPkg(pkgPath)
	if err != nil {
		return "", errors.FailedToLoadPackage
	}

	kclPkg.SetVendorMode(opts.IsVendor())

	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return "", err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != (*reporter.KpmEvent)(nil) {
		return "", err
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

	// Calculate the absolute path of entry file described by '--input'.
	compiler := runner.NewCompilerWithOpts(opts)

	// Call the kcl compiler.
	compileResult, err := kclPkg.Compile(
		globalPkgPath,
		compiler,
	)

	if err != nil {
		return "", reporter.NewErrorEvent(reporter.CompileFailed, err, "failed to compile the kcl package")
	}

	return compileResult.GetRawYamlResult(), nil
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
