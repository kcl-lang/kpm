// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"os"

	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/oci"
	"kusionstack.io/kpm/pkg/opt"
	pkg "kusionstack.io/kpm/pkg/package"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/settings"
	"kusionstack.io/kpm/pkg/utils"
)

// NewPushCmd new a Command for `kpm push`.
func NewPushCmd(settings *settings.Settings) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "push",
		Usage:  "push kcl package to OCI registry.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  FLAG_TAR_PATH,
				Usage: "a kcl file as the compile entry file",
			},
		},
		Action: func(c *cli.Context) error {

			localTarPath := c.String(FLAG_TAR_PATH)
			ociUrl := c.Args().First()
			if len(ociUrl) == 0 {
				reporter.Report("kpm: oci url must be specified.")
				reporter.ExitWithReport("kpm: run 'kpm push help' for more information.")
			}

			var tarPath string
			var err error

			// clean the kcl package tar.
			defer func() {
				if len(tarPath) != 0 && utils.DirExists(tarPath) {
					err = os.Remove(tarPath)
					if err != nil {
						err = errors.InternalBug
					}
				}
			}()

			if len(localTarPath) == 0 {
				// If the tar package to be pushed is not specified,
				// the current kcl package is packaged into tar and pushed.
				tarPath, err = pushCurrentPackage(ociUrl, settings)
			} else {
				// Else push the tar package specified.
				err = pushTarPackage(ociUrl, localTarPath, settings)
			}

			if err != nil {
				return err
			}

			return nil
		},
	}
}

// pushCurrentPackage will push the current package to the oci registry.
func pushCurrentPackage(ociUrl string, settings *settings.Settings) (string, error) {
	pwd, err := os.Getwd()

	if err != nil {
		reporter.ExitWithReport("kpm: internal bug: failed to load working directory")
	}
	// 1. Load the current kcl packege.
	kclPkg, err := pkg.LoadKclPkg(pwd)

	if err != nil {
		reporter.ExitWithReport("kpm: failed to load package in " + pwd + ".")
		return "", err
	}

	reporter.Report("kpm: the current package '" + kclPkg.GetPkgName() + "' will be pushed.")

	// 2. Package the current kcl package into default tar path.
	tarPath, err := kclPkg.PackageCurrentPkg()
	if err != nil {
		return tarPath, err
	}

	// 3. Generate the OCI options from oci url and the version of current kcl package.
	ociOpts, err := opt.ParseOciOptionFromOciUrl(ociUrl, kclPkg.GetPkgTagForOci())
	if err != nil {
		return tarPath, err
	}

	// 4. Push it.
	err = oci.Push(tarPath, ociOpts.Reg, ociOpts.Repo, ociOpts.Tag, settings)
	if err != nil {
		return tarPath, err
	}

	return tarPath, nil
}

// pushTarPackage will push the kcl package in tarPath to the oci registry.
// If the tar in 'tarPath' is not a kcl package tar, pushTarPackage will return an error.
func pushTarPackage(ociUrl, localTarPath string, settings *settings.Settings) error {
	var kclPkg *pkg.KclPkg
	var err error

	// clean the temp dir used to untar kcl package tar file.
	defer func() {
		if kclPkg != nil && utils.DirExists(kclPkg.HomePath) {
			err = os.RemoveAll(kclPkg.HomePath)
			if err != nil {
				err = errors.InternalBug
			}
		}
	}()

	// 1. load the kcl package from the tar path.
	kclPkg, err = pkg.LoadKclPkgFromTar(localTarPath)
	if err != nil {
		return err
	}

	// 2. Generate the OCI options from oci url and the version of current kcl package.
	ociOpts, err := opt.ParseOciOptionFromOciUrl(ociUrl, kclPkg.GetPkgTagForOci())
	if err != nil {
		return err
	}

	// 3. Push it.
	err = oci.Push(localTarPath, ociOpts.Reg, ociOpts.Repo, ociOpts.Tag, settings)
	if err != nil {
		return err
	}

	return nil
}
