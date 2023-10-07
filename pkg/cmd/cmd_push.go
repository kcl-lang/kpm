// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// NewPushCmd new a Command for `kpm push`.
func NewPushCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "push",
		Usage:  "push kcl package to OCI registry.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  FLAG_TAR_PATH,
				Usage: "a kcl file as the compile entry file",
			},
			// '--vendor' will trigger the vendor mode
			// In the vendor mode, the package search path is the subdirectory 'vendor' in current package.
			// In the non-vendor mode, the package search path is the $KCL_PKG_PATH.
			&cli.BoolFlag{
				Name:  FLAG_VENDOR,
				Usage: "push in vendor mode",
			},
		},
		Action: func(c *cli.Context) error {
			return KpmPush(c, kpmcli)
		},
	}
}

func KpmPush(c *cli.Context, kpmcli *client.KpmClient) error {
	localTarPath := c.String(FLAG_TAR_PATH)
	ociUrl := c.Args().First()

	var err error

	if len(localTarPath) == 0 {
		// If the tar package to be pushed is not specified,
		// the current kcl package is packaged into tar and pushed.
		err = pushCurrentPackage(ociUrl, c.Bool(FLAG_VENDOR), kpmcli)
	} else {
		// Else push the tar package specified.
		err = pushTarPackage(ociUrl, localTarPath, c.Bool(FLAG_VENDOR), kpmcli)
	}

	if err != nil {
		return err
	}

	return nil
}

// genDefaultOciUrlForKclPkg will generate the default oci url from the current package.
func genDefaultOciUrlForKclPkg(pkg *pkg.KclPkg, kpmcli *client.KpmClient) (string, error) {

	urlPath := utils.JoinPath(kpmcli.GetSettings().DefaultOciRepo(), pkg.GetPkgName())

	u := &url.URL{
		Scheme: oci.OCI_SCHEME,
		Host:   kpmcli.GetSettings().DefaultOciRegistry(),
		Path:   urlPath,
	}

	return u.String(), nil
}

// pushCurrentPackage will push the current package to the oci registry.
func pushCurrentPackage(ociUrl string, vendorMode bool, kpmcli *client.KpmClient) error {
	pwd, err := os.Getwd()

	if err != nil {
		reporter.ReportEventToStderr(reporter.NewEvent(reporter.Bug, "internal bug: failed to load working directory"))
		return err
	}
	// 1. Load the current kcl packege.
	kclPkg, err := pkg.LoadKclPkg(pwd)

	if err != nil {
		reporter.ReportEventToStderr(reporter.NewEvent(reporter.FailedLoadKclMod, fmt.Sprintf("failed to load package in '%s'", pwd)))
		return err
	}

	// 2. push the package
	return pushPackage(ociUrl, kclPkg, vendorMode, kpmcli)
}

// pushTarPackage will push the kcl package in tarPath to the oci registry.
// If the tar in 'tarPath' is not a kcl package tar, pushTarPackage will return an error.
func pushTarPackage(ociUrl, localTarPath string, vendorMode bool, kpmcli *client.KpmClient) error {
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

	// 2. push the package
	return pushPackage(ociUrl, kclPkg, vendorMode, kpmcli)
}

// pushPackage will push the kcl package to the oci registry.
// 1. pushPackage will package the current kcl package into default tar path.
// 2. If the oci url is not specified, generate the default oci url from the current package.
// 3. Generate the OCI options from oci url and the version of current kcl package.
// 4. Push the package to the oci registry.
func pushPackage(ociUrl string, kclPkg *pkg.KclPkg, vendorMode bool, kpmcli *client.KpmClient) error {

	tarPath, err := kpmcli.PackagePkg(kclPkg, vendorMode)
	if err != nil {
		return err
	}

	// clean the tar path.
	defer func() {
		if kclPkg != nil && utils.DirExists(tarPath) {
			err = os.RemoveAll(tarPath)
			if err != nil {
				err = errors.InternalBug
			}
		}
	}()

	// 2. If the oci url is not specified, generate the default oci url from the current package.
	if len(ociUrl) == 0 {
		ociUrl, err = genDefaultOciUrlForKclPkg(kclPkg, kpmcli)
		if err != nil || len(ociUrl) == 0 {
			return reporter.NewErrorEvent(
				reporter.InvalidCmd,
				fmt.Errorf("failed to generate default oci url for current package"),
				"run 'kpm push help' for more information",
			)
		}
	}

	// 3. Generate the OCI options from oci url and the version of current kcl package.
	ociOpts, err := opt.ParseOciOptionFromOciUrl(ociUrl, kclPkg.GetPkgTag())
	if err != (*reporter.KpmEvent)(nil) {
		return reporter.NewErrorEvent(
			reporter.UnsupportOciUrlScheme,
			errors.InvalidOciUrl,
			"only support url scheme 'oci://'.",
		)
	}

	reporter.ReportMsgTo(fmt.Sprintf("kpm: package '%s' will be pushed", kclPkg.GetPkgName()), kpmcli.GetLogWriter())
	// 4. Push it.
	err = kpmcli.PushToOci(tarPath, ociOpts)
	if err != (*reporter.KpmEvent)(nil) {
		return err
	}

	return nil
}
