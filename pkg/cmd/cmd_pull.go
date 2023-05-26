// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"path/filepath"

	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/oci"
	"kusionstack.io/kpm/pkg/opt"
	"kusionstack.io/kpm/pkg/reporter"
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
			ociUrl := c.Args().Get(0)
			localPath := c.Args().Get(1)

			if len(ociUrl) == 0 {
				reporter.Report("kpm: oci url must be specified.")
				reporter.ExitWithReport("kpm: run 'kpm pull help' for more information.")
			}

			ociOpt, err := opt.ParseOciOptionFromOciUrl(ociUrl, tag)
			if err != nil {
				return err
			}

			absPullPath, err := filepath.Abs(localPath)
			if err != nil {
				return err
			}

			err = oci.Pull(absPullPath, ociOpt.Reg, ociOpt.Repo, ociOpt.Tag)
			if err != nil {
				return err
			}

			reporter.Report("kpm: the kcl package tar is pulled successfully.")
			return nil
		},
	}
}
