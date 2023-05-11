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
			// '--update' will trigger the auto-update mode
			// In the auto-update mode, `kpm metadata` will automatically check the local package, update and download the package.
			// In the non-auto-update mode, `kpm metadata`` will only return the metadata of the existing packages.
			&cli.BoolFlag{
				Name:  FLAG_UPDATE,
				Usage: "check the local package and update and download the local package.",
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

			autoUpdate := c.Bool(FLAG_UPDATE)
			jsonStr, err := kclPkg.ResolveDepsMetadataInJsonStr(globalPkgPath, autoUpdate)
			if err != nil {
				return err
			}

			if autoUpdate {
				err = kclPkg.UpdateModAndLockFile()
				if err != nil {
					return err
				}
			}

			fmt.Println(jsonStr)

			return nil
		},
	}
}
