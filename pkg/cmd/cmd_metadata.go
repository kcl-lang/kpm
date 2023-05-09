// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/env"
	"kusionstack.io/kpm/pkg/errors"
	pkg "kusionstack.io/kpm/pkg/package"
)

// NewMetadataCmd new a Command for `kpm metadata`.
func NewMetadataCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "metadata",
		Usage:  "output the resolved dependencies of a package",
		Flags: []cli.Flag{
			// '--vendor' will trigger the vendor mode
			// In the vendor mode, the package search path is the subdirectory 'vendor' in current package.
			// In the non-vendor mode, the package search path is the $KCL_PKG_PATH.
			&cli.BoolFlag{
				Name:  FLAG_VENDOR,
				Usage: "get metadata in vendor mode",
			},
		},
		Action: func(c *cli.Context) error {
			pwd, err := os.Getwd()
			if err != nil {
				return errors.InternalBug
			}

			kclPkg, err := pkg.LoadKclPkg(pwd)
			if err != nil {
				return err
			}

			globalPkgPath, err := env.GetAbsPkgPath()
			if err != nil {
				return err
			}

			kclPkg.SetVendorMode(c.Bool(FLAG_VENDOR))

			err = kclPkg.ValidateKpmHome(globalPkgPath)
			if err != nil {
				return err
			}

			jsonStr, err := kclPkg.ResolveDepsMetadataInJsonStr(globalPkgPath)
			if err != nil {
				return err
			}

			err = kclPkg.UpdateModAndLockFile()
			if err != nil {
				return err
			}

			fmt.Println(jsonStr)

			return nil
		},
	}
}
