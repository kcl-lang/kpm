package api

import (
	"os"
	"path/filepath"
	"strings"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/opt"
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

// CompileWithOpt will compile the kcl program without kcl package.
// Deprecated: This method will not be maintained in the future. Use RunWithOpts instead.
func RunWithOpt(opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	// The entries will override the entries in the settings file.
	if opts.HasSettingsYaml() && len(opts.KFilenameList) > 0 && len(opts.Entries()) > 0 {
		opts.KFilenameList = []string{}
	}
	if len(opts.Entries()) > 0 {
		for _, entry := range opts.Entries() {
			if filepath.IsAbs(entry) {
				opts.Merge(kcl.WithKFilenames(entry))
			} else {
				opts.Merge(kcl.WithKFilenames(filepath.Join(opts.PkgPath(), entry)))
			}
		}
	} else if !opts.HasSettingsYaml() && len(opts.KFilenameList) == 0 {
		// If no entry, no kcl files and no settings files.
		opts.Merge(kcl.WithKFilenames(opts.PkgPath()))
	}
	opts.Merge(kcl.WithWorkDir(opts.PkgPath()))
	return kcl.RunWithOpts(*opts.Option)
}

// RunWithOpts will compile the kcl package with the compile options.
func RunWithOpts(opts ...opt.Option) (*kcl.KCLResultList, error) {
	mergedOpts := opt.DefaultCompileOptions()
	for _, opt := range opts {
		opt(mergedOpts)
	}
	return runPkgWithOpt(mergedOpts)
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
// Deprecated: This method will not be maintained in the future. Use RunWithOpts instead.
func RunPkgWithOpt(opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	kpmcli, err := client.NewKpmClient()
	kpmcli.SetNoSumCheck(opts.NoSumCheck())
	if err != nil {
		return nil, err
	}
	return run(kpmcli, opts)
}

func runPkgWithOpt(opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	kpmcli, err := client.NewKpmClient()
	kpmcli.SetNoSumCheck(opts.NoSumCheck())
	if err != nil {
		return nil, err
	}
	return run(kpmcli, opts)
}

// RunCurrentPkg will compile the current kcl package.
func RunCurrentPkg(opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	pwd, err := os.Getwd()
	opts.SetPkgPath(pwd)

	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, failed to load working directory.")
	}

	return RunPkgWithOpt(opts)
}

// RunTarPkg will compile the kcl package from a kcl package tar.
func RunTarPkg(tarPath string, opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	absTarPath, err := utils.AbsTarPath(tarPath)
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
	kpmcli, err := client.NewKpmClient()
	kpmcli.SetNoSumCheck(opts.NoSumCheck())
	if err != nil {
		return nil, err
	}
	// The directory after extracting the tar package is taken as the root directory of the package,
	// and kclvm is called to compile the kcl program under the 'destDir'.
	// e.g.
	// if the tar path is 'xxx/xxx/xxx/test.tar',
	// the 'xxx/xxx/xxx/test' will be taken as the root path of the kcl package to compile.
	return run(kpmcli, opts)
}

// RunOciPkg will compile the kcl package from an OCI reference.
func RunOciPkg(ociRef, version string, opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	kpmcli, err := client.NewKpmClient()
	if err != nil {
		return nil, err
	}
	kpmcli.SetNoSumCheck(opts.NoSumCheck())
	ociOpts, err := kpmcli.ParseOciOptionFromString(ociRef, version)

	if err != nil {
		return nil, err
	}

	// 1. Create the temporary directory to pull the tar.
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, please contact us to fix it.")
	}
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	localPath := ociOpts.SanitizePathWithSuffix(tmpDir)

	// 2. Pull the tar.
	err = oci.Pull(localPath, ociOpts.Reg, ociOpts.Repo, ociOpts.Tag, kpmcli.GetSettings())

	if err != (*reporter.KpmEvent)(nil) {
		return nil, err
	}

	// 3.Get the (*.tar) file path.
	matches, err := filepath.Glob(filepath.Join(localPath, constants.KCL_PKG_TAR))
	if err != nil || len(matches) != 1 {
		if err != nil {
			return nil, reporter.NewErrorEvent(reporter.FailedGetPkg, err, "failed to pull kcl package")
		} else {
			return nil, errors.FailedPull
		}
	}

	// 4. Untar the tar file.
	absTarPath, err := utils.AbsTarPath(matches[0])
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
	return run(kpmcli, opts)
}

// 'run' will compile the kcl package from the compile options by kpm client.
func run(kpmcli *client.KpmClient, opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	pkgPath, err := filepath.Abs(opts.PkgPath())
	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, please contact us to fix it.")
	}

	kclPkg, err := kpmcli.LoadPkgFromPath(pkgPath)
	if err != nil {
		return nil, err
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
	} else if kclPkg.HasProfile() {
		opts.Merge(*kclPkg.GetKclOpts())
	} else if !opts.HasSettingsYaml() {
		// no entry
		opts.Merge(kcl.WithKFilenames(opts.PkgPath()))
	}
	opts.Merge(kcl.WithWorkDir(opts.PkgPath()))

	// Calculate the absolute path of entry file described by '--input'.
	compiler := runner.NewCompilerWithOpts(opts)

	kpmcli.SetLogWriter(opts.LogWriter())

	// Call the kcl compiler.
	compileResult, err := kpmcli.Compile(kclPkg, compiler)

	if err != nil {
		return nil, reporter.NewErrorEvent(reporter.CompileFailed, err, "failed to compile the kcl package")
	}

	return compileResult, nil
}
