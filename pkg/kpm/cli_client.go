package kpm

import (
	"encoding/json"
	"errors"
	"github.com/KusionStack/kpm/pkg/go-oneutils/ExecCmd"
	"github.com/KusionStack/kpm/pkg/go-oneutils/GlobalStore"
	"github.com/KusionStack/kpm/pkg/go-oneutils/PathHandle"
	"os"
)

type CliClient struct {
	GitStore         *GlobalStore.FileStore
	RegistryStore    *GlobalStore.FileStore
	WorkDir          string
	Root             string
	RegistryAddr     string
	RegistryAddrPath string
	KclVmVersion     string
	NestedMode       bool
}

func (c CliClient) Get(rb *RequireBase) error {
	var store *GlobalStore.FileStore
	if rb.Type == "git" {
		store = c.GitStore
	} else {
		store = c.RegistryStore
	}
	exist, err := store.DirIsExist(rb.Name + "@" + string(rb.Version))
	if err != nil {
		return err
	}

	//找不到，开始查找元文件
	metadata, err := LoadLocalMetadata(rb.Name, rb.Version, store)
	if err != nil {
		//找不到元文件,下载
		err = c.PkgDownload(rb)
		if err != nil {
			return err
		}
		return err
	}
	if exist {
		//找到包
		println("found", rb.GetPkgString())
		if rb.Integrity == "" {
			rb.Integrity = metadata.Integrity
		} else {
			if rb.Integrity != metadata.Integrity {
				e := errors.New("the package integrity check failed")

				return e
			}
		}
		return nil
	}
	println("not found pkg", rb.GetPkgString())
	//找到元文件
	if c.NestedMode {
		err = c.Build(rb)
	} else {
		err = metadata.Build(store)
	}

	println("building pkg", rb.GetPkgString())
	if err != nil {
		return err
	}
	//下载成功则得到元数据，开始检查hash文件是否缺失

	return nil
}

// PkgDownload 下载包
func (c CliClient) PkgDownload(rb *RequireBase) error {
	println("downloading pkg", rb.GetPkgString())
	if rb.Type == "git" {
		//git版本
		err := PathHandle.RunInTempDir(func(tmppath string) error {
			err := ExecCmd.Run(tmppath, "git", "clone", "--branch", string(rb.Version), "https://"+rb.Name)
			if err != nil {
				return err
			}
			t2 := tmppath + PathHandle.Separator + rb.GetShortName()
			metadata, err := NewMetadata(rb.Name, t2, string(rb.Version), c.GitStore)
			if err != nil {
				return err
			}
			err = metadata.Save(c.GitStore)
			if err != nil {
				return err
			}
			if c.NestedMode {
				err = c.Build(rb)
			} else {
				err = metadata.Build(c.GitStore)
			}
			if err != nil {
				return err
			}
			rb.Integrity = metadata.Integrity
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		//仓库版本
	}
	return nil
}

func (c CliClient) LoadKpmFileStruct(rb *RequireBase) (*KpmFile, error) {
	var store *GlobalStore.FileStore
	if rb.Type == "git" {
		store = kpmC.GitStore
	} else {
		store = kpmC.RegistryStore
	}
	path, err := store.GetDirPath(rb.Name + "@" + string(rb.Version))
	if err != nil {
		return nil, err
	}
	filebytes, err := os.ReadFile(path + PathHandle.Separator + "kpm.json")
	if err != nil {
		return nil, err
	}
	kf := KpmFile{}
	err = json.Unmarshal(filebytes, &kf)
	if err != nil {
		return nil, err
	}
	return &kf, nil
}

func (c CliClient) LoadKpmFileStructInWorkdir() (*KpmFile, error) {
	filebytes, err := os.ReadFile(c.WorkDir + PathHandle.Separator + "kpm.json")
	if err != nil {
		return nil, err
	}
	kf := KpmFile{}
	err = json.Unmarshal(filebytes, &kf)
	if err != nil {
		return nil, err
	}
	return &kf, nil
}

func (c CliClient) SaveKpmFileInWorkdir(kf *KpmFile) error {
	marshal, err := json.Marshal(&kf)
	if err != nil {
		return err
	}
	err = os.WriteFile(c.WorkDir+PathHandle.Separator+"kpm.json", marshal, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (c CliClient) Build(rb *RequireBase) error {
	//先构建，再读文件，最后链接依赖
	var store *GlobalStore.FileStore
	if rb.Type == "git" {
		store = kpmC.GitStore
	} else {
		store = kpmC.RegistryStore
	}
	md, err := LoadLocalMetadata(rb.Name, rb.Version, store)
	if err != nil {
		return err
	}
	err = store.BuildDir(md.Files, md.Name+"@"+string(md.Version))
	if err != nil {
		return err
	}
	fileStruct, err := c.LoadKpmFileStruct(rb)
	if err != nil {
		if err == os.ErrNotExist {
			return nil
		}
		return err
	}
	//_ = fileStruct
	for s, base := range fileStruct.Direct {
		var rbstore *GlobalStore.FileStore
		if rb.Type == "git" {
			rbstore = kpmC.GitStore
		} else {
			rbstore = kpmC.RegistryStore
		}
		path, err := store.GetDirPath(rb.Name)
		if err != nil {
			return err
		}
		err = rbstore.Link(base.Name+"@"+string(base.Version), path+"@"+string(rb.Version)+PathHandle.Separator+"external"+PathHandle.Separator+s)
		if err != nil {
			return err
		}
	}
	return nil
}
