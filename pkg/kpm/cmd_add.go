package kpm

import (
	"errors"
	"github.com/KusionStack/kpm/pkg/go-oneutils/GlobalStore"
	"github.com/KusionStack/kpm/pkg/go-oneutils/PathHandle"
	"github.com/KusionStack/kpm/pkg/go-oneutils/Semver"
	"github.com/urfave/cli/v2"
	"os"
)

func NewAddCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "add",
		Usage:  "add dependencies pkg",
		Flags: []cli.Flag{&cli.BoolFlag{
			Name:  "git",
			Usage: "add git pkg",
		},
		//&cli.StringFlag{
		//	Name:  "",
		//	Usage: "",
		//},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowAppHelpAndExit(c, 0)
			}
			println("add...")
			kf, err := kpmC.LoadKpmFileStructInWorkdir()
			if err != nil {
				return err
			}
			ps := c.Args().Slice()[c.Args().Len()-1]
			var rbstore *GlobalStore.FileStore
			if c.Bool("git") {
				ps = "git:" + ps
				rbstore = kpmC.GitStore
			} else {
				ps = "registry:" + ps
				rbstore = kpmC.RegistryStore
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
			if kf.Direct == nil {
				kf.Direct = make(DirectRequire, 16)
			}
			if kf.Indirect == nil {
				kf.Indirect = make(IndirectRequire, 16)
			}
			shortname := rb.GetShortName()
			_, ok := kf.Direct[shortname]
			if ok {
				e := errors.New("this package already exists")
				return e
			}
			kf.Direct[shortname] = rb
			dkf, err := kpmC.LoadKpmFileStruct(&rb)
			if err == nil {
				//找到文件
				kfv, err := Semver.NewFromString(kf.KclvmMinVersion)
				if err != nil {
					return err
				}
				dkfv, err := Semver.NewFromString(dkf.KclvmMinVersion)
				if err != nil {
					return err
				}
				if kfv.Cmp(dkfv) == -1 {
					e := errors.New("the KclvmMinVersion of the added dependency " + shortname + " is greater than the KclvmMinVersion of the workspace")

					return e
				}
				for k, v := range dkf.Indirect {
					kf.Indirect[k] = v
					//
					dpkgStruct, err := GetRequirePkgStruct(ps)
					if err != nil {
						return err
					}
					drb := RequireBase{
						RequirePkgStruct: *dpkgStruct,
					}
					err = kpmC.Get(&drb)
					if err != nil {
						return err
					}
					if !kpmC.NestedMode {
						//平铺模式
						var tmpstore *GlobalStore.FileStore
						if drb.Type == "git" {
							tmpstore = kpmC.GitStore
						} else {
							tmpstore = kpmC.RegistryStore
						}
						tmptargetPath := kpmC.WorkDir + PathHandle.Separator + "external" + PathHandle.Separator + PathHandle.URLToLocalDirPath(drb.Name) + "@" + string(drb.Version)
						_ = os.Remove(tmptargetPath)
						err = tmpstore.Link(drb.Name+"@"+string(drb.Version), tmptargetPath)
						if err != nil {
							return err
						}
					}
				}
				for _, v := range dkf.Direct {
					kf.Indirect[v.GetPkgString()] = v.Integrity
					err = kpmC.Get(&v)
					if err != nil {
						return err
					}
					if !kpmC.NestedMode {
						//平铺模式
						var tmpstore *GlobalStore.FileStore
						if v.Type == "git" {
							tmpstore = kpmC.GitStore
						} else {
							tmpstore = kpmC.RegistryStore
						}
						tmptargetPath := kpmC.WorkDir + PathHandle.Separator + "external" + PathHandle.Separator + PathHandle.URLToLocalDirPath(v.Name) + "@" + string(v.Version)
						_ = os.Remove(tmptargetPath)
						err = tmpstore.Link(v.Name+"@"+string(v.Version), tmptargetPath)
						if err != nil {
							return err
						}
					}
				}
			}
			var targetPath string
			if kpmC.NestedMode {
				targetPath = kpmC.WorkDir + PathHandle.Separator + "external" + PathHandle.Separator + shortname
			} else {
				//待开发
				targetPath = kpmC.WorkDir + PathHandle.Separator + "external" + PathHandle.Separator + PathHandle.URLToLocalDirPath(rb.Name) + "@" + string(rb.Version)
			}
			_ = os.Remove(targetPath)
			err = rbstore.Link(rb.Name+"@"+string(rb.Version), targetPath)
			if err != nil {
				return err
			}
			println("add", shortname, "success")
			//保存
			err = kpmC.SaveKpmFileInWorkdir(kf)
			if err != nil {
				return err
			}
			return nil
		},
	}
}
