// Copyright 2023 The KCL Authors. All rights reserved.
// Deprecated: The entire contents of this file will be deprecated. 
// Please use the kcl cli - https://github.com/kcl-lang/cli.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	reporter "kcl-lang.io/kpm/pkg/reporter"
)

// NewInitCmd new a Command for `kpm init`.
func NewInitCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "init",
		Usage:  "initialize new module in current directory",
		Action: func(c *cli.Context) error {
			pwd, err := os.Getwd()

			if err != nil {
				return reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, failed to load working directory.")
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

			if err != (*reporter.KpmEvent)(nil) {
				return err
			}

			err = kpmcli.InitEmptyPkg(&kclPkg)
			if err != nil {
				return err
			}

			reporter.ReportMsgTo(fmt.Sprintf("package '%s' init finished", pkgName), kpmcli.GetLogWriter())
			return nil
		},
	}
}
