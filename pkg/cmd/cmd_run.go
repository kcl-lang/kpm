// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/env"
	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/oci"
	"kusionstack.io/kpm/pkg/opt"
	pkg "kusionstack.io/kpm/pkg/package"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/runner"
	"kusionstack.io/kpm/pkg/settings"
	"kusionstack.io/kpm/pkg/utils"
)

const FLAG_INPUT = "input"
const FLAG_VENDOR = "vendor"
const FLAG_KCL = "kcl_args"
const FLAG_TAG = "tag"

// NewRunCmd new a Command for `kpm run`.
func NewRunCmd(settings *settings.Settings) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "run",
		Usage:  "compile kcl package.",
		Flags: []cli.Flag{
			// The entry kcl file.
			&cli.StringFlag{
				Name:  FLAG_INPUT,
				Usage: "a kcl file as the compile entry file",
			},
			&cli.StringFlag{
				Name:  FLAG_TAG,
				Usage: "the tag for oci artifact",
			},
			// '--vendor' will trigger the vendor mode
			// In the vendor mode, the package search path is the subdirectory 'vendor' in current package.
			// In the non-vendor mode, the package search path is the $KCL_PKG_PATH.
			&cli.BoolFlag{
				Name:  FLAG_VENDOR,
				Usage: "run in vendor mode",
			},

			// '--kcl' will pass the arguments to kclvm_cli.
			&cli.StringFlag{
				Name:  FLAG_KCL,
				Value: "",
				Usage: "Arguments for kclvm_cli",
			},
		},
		Action: func(c *cli.Context) error {
			pkgWillBeCompiled := c.Args().First()
			// 'kpm run' compile the current package undor '$pwd'.
			if len(pkgWillBeCompiled) == 0 {
				compileResult, err := runPkg(c.String(FLAG_INPUT), c.Bool(FLAG_VENDOR), c.String(FLAG_KCL))
				if err != nil {
					return err
				}
				fmt.Print(compileResult)
			} else {
				// 'kpm run <package source>' compile the kcl package from the <package source>.
				compileResult, err := runTar(pkgWillBeCompiled, c.String(FLAG_INPUT), c.Bool(FLAG_VENDOR), c.String(FLAG_KCL))
				if err == errors.InvalidKclPacakgeTar {
					compileResult, err = runOci(pkgWillBeCompiled, c.String(FLAG_TAG), c.String(FLAG_INPUT), c.Bool(FLAG_VENDOR), settings, c.String(FLAG_KCL))
					if err != nil {
						return err
					}
				} else if err != nil {
					return err
				}
				fmt.Print(compileResult)
			}
			return nil
		},
	}
}

// runTar will compile the kcl package from a kcl package tar.
func runTar(tarPath, entryFile string, vendorMode bool, kclArgs string) (string, error) {
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

	// The directory after extracting the tar package is taken as the root directory of the package,
	// and kclvm is called to compile the kcl program under the 'destDir'.
	// e.g.
	// if the tar path is 'xxx/xxx/xxx/test.tar',
	// the 'xxx/xxx/xxx/test' will be taken as the root path of the kcl package to compile.
	compileResult, compileErr := runPkgInPath(destDir, entryFile, vendorMode, kclArgs)

	if compileErr != nil {
		return "", compileErr
	}
	return compileResult, nil
}

// runOci will compile the kcl package from an OCI reference.
func runOci(ociRef, version, entryFile string, vendorMode bool, settings *settings.Settings, kclArgs string) (string, error) {
	ociOpts, err := opt.ParseOciOptionFromString(ociRef, version)

	if err != nil {
		return "", err
	}

	pwd, err := os.Getwd()

	if err != nil {
		return "", errors.InternalBug
	}

	localPath := ociOpts.AddStoragePathSuffix(pwd)
	localTarPath, err := oci.Pull(localPath, ociOpts.Reg, ociOpts.Repo, ociOpts.Tag, settings)

	if err != nil {
		return "", err
	}

	return runTar(localTarPath, entryFile, vendorMode, kclArgs)
}

// runPkg will compile current kcl package.
func runPkg(entryFile string, vendorMode bool, kclArgs string) (string, error) {

	// If no tar packages specified by "--tar" to run
	// kpm will take the current directory ($PWD) as the root of the kcl package and compile.
	pwd, err := os.Getwd()

	if err != nil {
		reporter.ExitWithReport("kpm: internal bug: failed to load working directory")
	}

	compileResult, err := runPkgInPath(pwd, entryFile, vendorMode, kclArgs)
	if err != nil {
		return "", err
	}

	return compileResult, nil
}

// runPkgInPath will load the 'KclPkg' from path 'pkgPath'.
// And run the kcl package with entry file in 'entryFilePath' in 'vendorMode'.
func runPkgInPath(pkgPath, entryFilePath string, vendorMode bool, kclArgs string) (string, error) {

	pkgPath, err := filepath.Abs(pkgPath)
	if err != nil {
		return "", errors.InternalBug
	}

	kclPkg, err := pkg.LoadKclPkg(pkgPath)
	if err != nil {
		return "", errors.FailedToLoadPackage
	}

	kclPkg.SetVendorMode(vendorMode)

	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return "", err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != nil {
		return "", err
	}

	// Calculate the absolute path of entry file described by '--input'.
	entryFilePath, err = getAbsInputPath(pkgPath, entryFilePath)
	if err != nil {
		return "", err
	}

	// Set the entry file into compile options.
	compileOpts := opt.NewKclvmOpts()
	compileOpts.EntryFiles = append(compileOpts.EntryFiles, entryFilePath)
	compileOpts.KclvmCliArgs = kclArgs

	err = compileOpts.Validate()
	if err != nil {
		return "", err
	}

	kclvmCmd, err := runner.NewCompileCmd(compileOpts)

	if err != nil {
		return "", err
	}

	// Call the kclvm_cli.
	compileResult, err := kclPkg.CompileWithEntryFile(globalPkgPath, kclvmCmd)

	if err != nil {
		return "", err
	}

	return compileResult, nil
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
