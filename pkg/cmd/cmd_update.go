// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"os"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/reporter"
)

// NewUpdateCmd new a Command for `kpm update`.
func NewUpdateCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "update",
		Usage:  "Update dependencies listed in kcl.mod.lock based on kcl.mod",
		Action: func(c *cli.Context) error {
			return KpmUpdate(c, kpmcli)
		},
	}
}

func KpmUpdate(c *cli.Context, kpmcli *client.KpmClient) error {
	// acquire the lock of the package cache.
	err := kpmcli.AcquirePackageCacheLock()
	if err != nil {
		return err
	}

	defer func() {
		// release the lock of the package cache after the function returns.
		releaseErr := kpmcli.ReleasePackageCacheLock()
		if releaseErr != nil && err == nil {
			err = releaseErr
		}
	}()

	input_paths := c.Args().Slice()

	pkg_paths := []string{}
	if len(input_paths) == 0 {
		pwd, err := os.Getwd()
		if err != nil {
			return errors.InternalBug
		}
		pkg_paths = append(pkg_paths, pwd)
	} else {
		pkg_paths = input_paths
	}

	for _, pkg_path := range pkg_paths {
		kclPkg, err := kpmcli.LoadPkgFromPath(pkg_path)
		if err != nil {
			return err
		}

		globalPkgPath, err := env.GetAbsPkgPath()
		if err != nil {
			return err
		}

		err = kclPkg.ValidateKpmHome(globalPkgPath)
		if err != (*reporter.KpmEvent)(nil) {
			return err
		}

		_, err = kpmcli.ResolveDepsMetadataInJsonStr(kclPkg, true)
		if err != nil {
			return err
		}

		err = kclPkg.UpdateModAndLockFile()
		if err != nil {
			return err
		}
	}
	return nil
}
