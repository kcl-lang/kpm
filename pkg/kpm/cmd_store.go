package kpm

import (
	"github.com/urfave/cli/v2"
)

func NewStoreCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "store",
		Usage:  "Reads and performs actions on kpm store that is on the current filesystem",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowAppHelpAndExit(c, 0)
			}
			app := cli.NewApp()
			app.Name = "store"
			app.Usage = "Reads and performs actions on kpm store that is on the current filesystem"
			app.Version = "v0.0.1-alpha.2"
			app.UsageText = CliHelp
			app.Commands = []*cli.Command{
				NewStoreAddCmd(),
				NewStoreAddFileCmd(),
			}
			//Add a parameter that ensures that the parameter is associated with "os. Args" are consistent in number。
			nargs := make([]string, c.NArg())
			nargs = nargs[:1]
			nargs = append(nargs, c.Args().Slice()...)
			err := app.Run(nargs)
			if err != nil {
				return err
			}
			return nil
		},
	}
}
