package kpm

import (
	"errors"
	"github.com/KusionStack/kpm/pkg/go-oneutils/PathHandle"
	"github.com/urfave/cli/v2"
	"io/fs"
	"os"
)

func NewDelCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "del",
		Usage:  "del  dependencies pkg",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowAppHelpAndExit(c, 0)
			}
			println("del...")
			kf, err := kpmC.LoadKpmFileStructInWorkdir()
			if err != nil {
				return err
			}
			shortname := c.Args().Slice()[c.Args().Len()-1]

			rb, ok := kf.Direct[shortname]
			if !ok {
				e := errors.New("this package does not exists")
				return e
			}
			delete(kf.Direct, shortname)
			var targetPath string
			if kpmC.NestedMode {
				targetPath = kpmC.WorkDir + PathHandle.Separator + "external" + PathHandle.Separator + shortname
			} else {
				//待开发
				targetPath = kpmC.WorkDir + PathHandle.Separator + "external" + PathHandle.Separator + PathHandle.URLToLocalDirPath(rb.Name) + "@" + string(rb.Version)
			}
			err = os.Remove(targetPath)
			if err != nil {
				if !(errors.Is(err, os.ErrNotExist) || errors.Is(err, &fs.PathError{Op: "remove", Path: targetPath})) {
					return err
				}
			}
			println("delete", shortname, "success")
			err = kpmC.SaveKpmFileInWorkdir(kf)
			if err != nil {
				return err
			}
			return nil
		},
	}
}
