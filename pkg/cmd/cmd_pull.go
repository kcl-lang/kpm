// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"net/url"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

// NewPullCmd new a Command for `kpm pull`.
func NewPullCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "pull",
		Usage:  "pull kcl package from OCI registry.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  FLAG_TAG,
				Usage: "the tag for oci artifact",
			},
		},
		Action: func(c *cli.Context) error {
			tag := c.String(FLAG_TAG)
			ociUrlOrPkgName := c.Args().Get(0)
			localPath := c.Args().Get(1)

			if len(ociUrlOrPkgName) == 0 {
				reporter.Report("kpm: oci url or package name must be specified.")
				reporter.ExitWithReport("kpm: run 'kpm pull help' for more information.")
			}

			if len(tag) == 0 {
				reporter.Report("kpm: pulling '", ociUrlOrPkgName, "'.")
			} else {
				reporter.Report("kpm: pulling '", ociUrlOrPkgName, "' with tag '", tag, "'.")
			}

			ociOpt, err := opt.ParseOciOptionFromOciUrl(ociUrlOrPkgName, tag)

			if err == errors.IsOciRef {
				settings, err := settings.GetSettings()
				if err != nil {
					return err
				}

				urlpath, err := url.JoinPath(settings.DefaultOciRepo(), ociUrlOrPkgName)
				if err != nil {
					return err
				}

				ociOpt, err = opt.ParseOciRef(urlpath)
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

			absPullPath, err := filepath.Abs(localPath)
			if err != nil {
				return err
			}

			tmpDir, err := os.MkdirTemp("", "")
			if err != nil {
				return errors.InternalBug
			}
			// clean the temp dir.
			defer os.RemoveAll(tmpDir)

			localPath = ociOpt.AddStoragePathSuffix(tmpDir)

			// 2. Pull the tar.
			err = oci.Pull(localPath, ociOpt.Reg, ociOpt.Repo, ociOpt.Tag)

			if err != nil {
				return err
			}

			// 3. Get the (*.tar) file path.
			matches, err := filepath.Glob(filepath.Join(localPath, KCL_PKG_TAR))
			if err != nil || len(matches) != 1 {
				return errors.FailedPullFromOci
			}

			// 4. Untar the tar file.
			err = utils.UnTarDir(matches[0], ociOpt.AddStoragePathSuffix(absPullPath))
			if err != nil {
				return err
			}

			reporter.Report("kpm: the kcl package is pulled successfully.")
			return nil
		},
	}
}
