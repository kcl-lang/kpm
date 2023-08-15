// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/api"
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
			return KpmPull(c)
		},
	}
}

func KpmPull(c *cli.Context) error {
	tag := c.String(FLAG_TAG)
	ociUrlOrPkgName := c.Args().Get(0)
	localPath := c.Args().Get(1)

	if len(ociUrlOrPkgName) == 0 {
		return reporter.NewErrorEvent(
			reporter.UnKnownPullWhat,
			errors.FailedPull,
			"oci url or package name must be specified.",
		)
	}

	if len(tag) == 0 {
		reporter.ReportEventToStdout(
			reporter.NewEvent(
				reporter.PullingStarted,
				fmt.Sprintf("start to pull '%s'.", ociUrlOrPkgName),
			),
		)
	} else {
		reporter.ReportEventToStdout(
			reporter.NewEvent(
				reporter.PullingStarted,
				fmt.Sprintf("start to pull '%s' with tag '%s'.", ociUrlOrPkgName, tag),
			),
		)
	}

	ociOpt, event := opt.ParseOciOptionFromOciUrl(ociUrlOrPkgName, tag)
	var err error
	if event != nil && (event.Type() == reporter.IsNotUrl || event.Type() == reporter.UrlSchemeNotOci) {
		settings := settings.GetSettings()
		if settings.ErrorEvent != nil {
			return settings.ErrorEvent
		}

		urlpath := utils.JoinPath(settings.DefaultOciRepo(), ociUrlOrPkgName)

		ociOpt, err = opt.ParseOciRef(urlpath)
		if err != nil {
			return err
		}
	} else if event != nil {
		return event
	}

	absPullPath, err := filepath.Abs(localPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err)
	}

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err, fmt.Sprintf("failed to create temp dir '%s'.", tmpDir))
	}

	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	localPath = ociOpt.AddStoragePathSuffix(tmpDir)

	// 2. Pull the tar.
	err = oci.Pull(localPath, ociOpt.Reg, ociOpt.Repo, ociOpt.Tag)

	if err != (*reporter.KpmEvent)(nil) {
		return err
	}

	// 3. Get the (*.tar) file path.
	tarPath := filepath.Join(localPath, api.KCL_PKG_TAR)
	matches, err := filepath.Glob(tarPath)
	if err != nil || len(matches) != 1 {
		if err == nil {
			err = errors.InvalidPkg
		}

		return reporter.NewErrorEvent(
			reporter.InvalidKclPkg,
			err,
			fmt.Sprintf("failed to find the kcl package tar from '%s'.", tarPath),
		)
	}

	// 4. Untar the tar file.
	storagePath := ociOpt.AddStoragePathSuffix(absPullPath)
	err = utils.UnTarDir(matches[0], storagePath)
	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedUntarKclPkg,
			err,
			fmt.Sprintf("failed to untar the kcl package tar from '%s' into '%s'.", matches[0], storagePath),
		)
	}

	reporter.ReportEventToStdout(
		reporter.NewEvent(reporter.PullingFinished, fmt.Sprintf("pulled '%s' in '%s' successfully.", ociUrlOrPkgName, storagePath)),
	)
	return nil
}
