// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	reporter "kcl-lang.io/kpm/pkg/reporter"
)

// NewInitCmd new a Command for `kpm init`.
func NewInitCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "init",
		Usage:  "initialize new module in current directory",
		Action: func(c *cli.Context) error {
			pwd, err := os.Getwd()

			if err != nil {
				reporter.Fatal("kpm: internal bugs, please contact us to fix it")
			}

			var pkgName string
			var pkgRootPath string
			// 1. If no package name is given, the current directory name is used as the package name.
			if c.NArg() == 0 {
				pkgName = filepath.Base(pwd)
				pkgRootPath = pwd
			} else {
				// 2. If the package name is given, create a new directory for the package.
				pkgName = c.Args().First()
				pkgRootPath = filepath.Join(pwd, pkgName)
				err = os.MkdirAll(pkgRootPath, 0755)
				if err != nil {
					return err
				}
			}

			initOpts := opt.InitOptions{
				Name:     pkgName,
				InitPath: pkgRootPath,
			}

			err = initOpts.Validate()
			if err != nil {
				return err
			}

			kclPkg := pkg.NewKclPkg(&initOpts)

			globalPkgPath, err := env.GetAbsPkgPath()

			if err != nil {
				return err
			}

			err = kclPkg.ValidateKpmHome(globalPkgPath)

			if err != nil {
				return err
			}

			err = kclPkg.InitEmptyPkg()

			if err == nil {
				reporter.Report("kpm: package '", pkgName, "' init finished")
			} else {
				reporter.ExitWithReport(err)
			}

			return err
		},
	}
}
