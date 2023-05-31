// Copyright 2022 The KCL Authors. All rights reserved.

package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/cmd"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/settings"
	"kusionstack.io/kpm/pkg/version"
)

func main() {
	reporter.InitReporter()
	setting, err := settings.GetSettings()
	if err != nil {
		reporter.Fatal(err)
	}
	app := cli.NewApp()
	app.Name = "kpm"
	app.Usage = "kpm is a kcl package manager"
	app.Version = version.GetVersionInStr()
	app.UsageText = "kpm  <command> [arguments]..."
	app.Commands = []*cli.Command{
		cmd.NewInitCmd(),
		cmd.NewAddCmd(),
		cmd.NewPkgCmd(),
		cmd.NewMetadataCmd(),
		cmd.NewRunCmd(),
		cmd.NewLoginCmd(setting),
		cmd.NewLogoutCmd(setting),
		cmd.NewPushCmd(setting),
		cmd.NewPullCmd(),
	}
	err = app.Run(os.Args)
	if err != nil {
		reporter.Fatal(err)
	}
}
