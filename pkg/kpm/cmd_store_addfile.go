package kpm

import (
	"github.com/KusionStack/kpm/pkg/go-oneutils/GlobalStore"
	"github.com/urfave/cli/v2"
)

func NewStoreAddFileCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "addfile",
		Usage:  "Add the current working directory to the global store",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowAppHelpAndExit(c, 0)
			}
			//Add the current working directory to the global store
			fim, err := kpmC.GitStore.AddDir(kpmC.WorkDir)
			if err != nil {
				return err
			}
			GlobalStore.ReleaseFileInfoMap(fim)
			return nil
		},
	}
}
