// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/opt"
	pkg "kusionstack.io/kpm/pkg/package"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/runner"
	"kusionstack.io/kpm/pkg/utils"
)

// NewRunCmd new a Command for `kpm run`.
func NewRunCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "run",
		Usage:  "compile kcl package.",
		Flags: []cli.Flag{
			// The kcl package tar executed.
			&cli.StringFlag{
				Name:  "tar",
				Usage: "The kcl package tar will be executed",
			},
			// The entry kcl file.
			&cli.StringFlag{
				Name:  "input",
				Usage: "a kcl file as the compile entry file",
			},
			// '--vendor' will trigger the vendor mode
			// In the vendor mode, the package search path is the subdirectory 'vendor' in current package.
			// In the non-vendor mode, the package search path is the $KPM_HOME.
			&cli.BoolFlag{
				Name:  "vendor",
				Usage: "run in vendor mode",
			},
		},
		Action: func(c *cli.Context) error {

			tarPath := c.String("tar")
			compileResult := ""

			// If a tar package specified by "--tar" to run
			if len(tarPath) != 0 {
				absTarPath, err := absTarPath(tarPath)
				if err != nil {
					return err
				}
				// Extract the tar package to a directory with the same name.
				// e.g.
				// 'xxx/xxx/xxx/test.tar' will be extracted to the directory 'xxx/xxx/xxx/test'.
				destDir := strings.TrimSuffix(absTarPath, filepath.Ext(absTarPath))
				err = utils.UnTarDir(absTarPath, destDir)
				if err != nil {
					return err
				}

				// The directory after extracting the tar package is taken as the root directory of the package,
				// and kclvm is called to compile the kcl program under the 'destDir'.
				// e.g.
				// if the tar path is 'xxx/xxx/xxx/test.tar',
				// the 'xxx/xxx/xxx/test' will be taken as the root path of the kcl package to compile.
				compileResult, compileErr := runPkgInPath(destDir, c.String("input"), c.Bool("vendor"))
				// After compiling the kcl program, clean up the contents extracted from the tar package.
				if utils.DirExists(destDir) {
					err = os.RemoveAll(destDir)
					if err != nil {
						return err
					}
				}
				if compileErr != nil {
					return compileErr
				}
				fmt.Print(compileResult)
			} else { // If no tar packages specified by "--tar" to run
				// kpm will take the current directory ($PWD) as the root of the kcl package and compile.
				pwd, err := os.Getwd()

				if err != nil {
					reporter.ExitWithReport("kpm: internal bug: failed to load working directory")
				}

				compileResult, err = runPkgInPath(pwd, c.String("input"), c.Bool("vendor"))
				if err != nil {
					return err
				}
				fmt.Print(compileResult)
			}
			return nil
		},
	}
}

// runPkgInPath will load the 'KclPkg' from path 'pkgPath'.
// And run the kcl package with entry file in 'entryFilePath' in 'vendorMode'.
func runPkgInPath(pkgPath, entryFilePath string, vendorMode bool) (string, error) {

	pkgPath, err := filepath.Abs(pkgPath)
	if err != nil {
		return "", errors.InternalBug
	}

	kclPkg, err := pkg.LoadKclPkg(pkgPath)
	if err != nil {
		return "", errors.FailedToLoadPackage
	}

	kclPkg.SetVendorMode(vendorMode)

	kpmHome, err := utils.GetAbsKpmHome()
	if err != nil {
		return "", err
	}

	err = kclPkg.ValidateKpmHome(kpmHome)
	if err != nil {
		return "", err
	}

	if len(entryFilePath) == 0 {
		reporter.Report("kpm: a compiler entry file need to specified by using '--input'")
		reporter.ExitWithReport("kpm: run 'kpm run help' for more information.")
	}

	// Calculate the absolute path of entry file described by '--input'.
	entryFilePath, err = getAbsInputPath(pkgPath, entryFilePath)
	if err != nil {
		return "", err
	}

	// Set the entry file into compile options.
	compileOpts := opt.NewKclvmOpts()
	compileOpts.EntryFiles = append(compileOpts.EntryFiles, entryFilePath)
	err = compileOpts.Validate()
	if err != nil {
		return "", err
	}

	kclvmCmd, err := runner.NewCompileCmd(compileOpts)

	if err != nil {
		return "", err
	}

	// Call the kclvm_cli.
	compileResult, err := kclPkg.CompileWithEntryFile(kpmHome, kclvmCmd)

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
	if !utils.DirExists(absTarPath) || filepath.Ext(absTarPath) != ".tar" {
		return "", errors.InvalidKclPacakgeTar
	}

	return absTarPath, nil
}

// getAbsInputPath will return the abs path of the file path described by '--input'.
// If the path exists after 'inputPath' is computed as a full path, it will be returned.
// If not, the kpm checks whether the full path of 'pkgPath/inputPath' exists,
// If the full path of 'pkgPath/inputPath' exists, it will be returned.
// If not, getAbsInputPath returns 'entry file not found' error.
func getAbsInputPath(pkgPath string, inputPath string) (string, error) {
	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return "", errors.InternalBug
	}

	if utils.DirExists(absInput) {
		return absInput, nil
	}

	absInput = filepath.Join(pkgPath, inputPath)
	if utils.DirExists(absInput) {
		return absInput, nil
	}

	return "", errors.EntryFileNotFound
}
