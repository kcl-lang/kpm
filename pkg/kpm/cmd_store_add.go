package kpm

import (
	"github.com/urfave/cli/v2"
)

func NewStoreAddCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "add",
		Usage:  "Add packages to the global store",
		Flags: []cli.Flag{&cli.BoolFlag{
			Name:  "git",
			Usage: "add git pkg",
		}},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowAppHelpAndExit(c, 0)
			}
			//Add packages to the global store
			ps := c.Args().Slice()[c.Args().Len()-1]
			if c.Bool("git") {
				ps = "git:" + ps
			} else {
				ps = "registry:" + ps
			}
			pkgStruct, err := GetRequirePkgStruct(ps)
			if err != nil {
				return err
			}
			rb := RequireBase{
				RequirePkgStruct: *pkgStruct,
			}
			err = kpmC.Get(&rb)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
