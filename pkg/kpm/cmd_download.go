package kpm

import (
	"github.com/KusionStack/kpm/pkg/go-oneutils/GlobalStore"
	"github.com/KusionStack/kpm/pkg/go-oneutils/PathHandle"
	"github.com/urfave/cli/v2"
	"os"
)

func NewDownloadCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "download",
		Usage:  "download dependencies pkg to local cache and link to workspace",
		Action: func(c *cli.Context) error {
			if c.NArg() != 0 {
				cli.ShowAppHelpAndExit(c, 0)
			}
			println("download...")
			kf, err := kpmC.LoadKpmFileStructInWorkdir()
			if err != nil {
				return err
			}
			globalWriterFlag := false
			for ps, integrity := range kf.Indirect {
				pkgStruct, err := GetRequirePkgStruct(ps)
				if err != nil {
					return err
				}
				rb := RequireBase{
					RequirePkgStruct: *pkgStruct,
					Integrity:        integrity,
				}
				writerFlag := rb.Integrity == ""
				err = kpmC.Get(&rb)
				if err != nil {

					return err
				}
				if !kpmC.NestedMode {
					//平铺模式
					var tmpstore *GlobalStore.FileStore
					if rb.Type == "git" {
						tmpstore = kpmC.GitStore
					} else {
						tmpstore = kpmC.RegistryStore
					}
					tmptargetPath := kpmC.WorkDir + PathHandle.Separator + "external" + PathHandle.Separator + PathHandle.URLToLocalDirPath(rb.Name) + "@" + string(rb.Version)
					_ = os.Remove(tmptargetPath)
					err = tmpstore.Link(rb.Name+"@"+string(rb.Version), tmptargetPath)
					if err != nil {
						return err
					}
				}
				if writerFlag {
					globalWriterFlag = true
					kf.Indirect[ps] = rb.Integrity
				}
			}
			for rbn, rb := range kf.Direct {
				writerFlag := rb.Integrity == ""
				err = kpmC.Get(&rb)
				if err != nil {
					return err
				}
				var rbstore *GlobalStore.FileStore
				if rb.Type == "git" {
					rbstore = kpmC.GitStore
				} else {
					rbstore = kpmC.RegistryStore
				}
				var targetPath string
				if kpmC.NestedMode {
					targetPath = kpmC.WorkDir + PathHandle.Separator + "external" + PathHandle.Separator + rbn
				} else {
					//待开发
					targetPath = kpmC.WorkDir + PathHandle.Separator + "external" + PathHandle.Separator + PathHandle.URLToLocalDirPath(rb.Name) + "@" + string(rb.Version)
				}
				_ = os.Remove(targetPath)
				err = rbstore.Link(rb.Name+"@"+string(rb.Version), targetPath)
				if err != nil {
					return err
				}
				if writerFlag {
					globalWriterFlag = true
					kf.Direct[rbn] = rb
				}
			}

			if globalWriterFlag {
				err = kpmC.SaveKpmFileInWorkdir(kf)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
}
