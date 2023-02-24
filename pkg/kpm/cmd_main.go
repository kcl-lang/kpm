package kpm

import (
	"github.com/urfave/cli/v2"
)

func CLI(args ...string) error {
	app := cli.NewApp()
	app.Name = "kpm"
	app.Usage = "kpm is a kcl package manager"
	app.Version = "v0.0.1-alpha.1"
	app.UsageText = CliHelp
	app.Commands = []*cli.Command{
		NewInitCmd(),
		NewAddCmd(),
		NewDelCmd(),
		NewDownloadCmd(),
		NewStoreCmd(),
	}
	err := Setup()
	if err != nil {
		return err
	}
	err = app.Run(args)
	if err != nil {
		return err
	}
	return nil
}
