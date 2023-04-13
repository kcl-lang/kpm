// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
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
			pwd, err := os.Getwd()

			if err != nil {
				reporter.ExitWithReport("kpm: internal bug: failed to load working directory")
			}

			kclPkg, err := pkg.LoadKclPkg(pwd)
			if err != nil {
				reporter.ExitWithReport("kpm: failed to package pkg from ", pwd)
				return err
			}

			kclPkg.SetVendorMode(c.Bool("vendor"))

			kpmHome, err := utils.GetAbsKpmHome()
			if err != nil {
				return err
			}

			err = kclPkg.ValidateKpmHome(kpmHome)
			if err != nil {
				return err
			}

			entryFile := c.String("input")
			if len(entryFile) == 0 {
				reporter.Report("kpm: a compiler entry file need to specified by using '--input'")
				reporter.ExitWithReport("kpm: run 'kpm run help' for more information.")
			}

			// Set the entry file into compile options.
			compileOpts := opt.NewKclvmOpts()
			compileOpts.EntryFiles = append(compileOpts.EntryFiles, entryFile)
			err = compileOpts.Validate()
			if err != nil {
				return err
			}

			kclvmCmd, err := runner.NewCompileCmd(compileOpts)

			if err != nil {
				return err
			}

			// Call the kclvm_cli.
			compileResult, err := kclPkg.CompileWithEntryFile(kpmHome, kclvmCmd)

			if err != nil {
				return err
			}

			fmt.Print(compileResult)

			return nil
		},
	}
}
