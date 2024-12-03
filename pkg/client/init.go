package client

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// InitOptions contains the options for initializing a kcl package.
type InitOptions struct {
	ModPath    string
	ModName    string
	ModVersion string
	WorkDir    string
}

type InitOption func(*InitOptions) error

func WithInitWorkDir(workDir string) InitOption {
	return func(opts *InitOptions) error {
		opts.WorkDir = workDir
		return nil
	}
}

func WithInitModVersion(modVersion string) InitOption {
	return func(opts *InitOptions) error {
		opts.ModVersion = modVersion
		return nil
	}
}

func WithInitModPath(modPath string) InitOption {
	return func(opts *InitOptions) error {
		opts.ModPath = modPath
		return nil
	}
}

func WithInitModName(modName string) InitOption {
	return func(opts *InitOptions) error {
		opts.ModName = modName
		return nil
	}
}

func (c *KpmClient) Init(options ...InitOption) error {
	opts := &InitOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return err
		}
	}

	modPath := opts.ModPath
	modName := opts.ModName
	modVer := opts.ModVersion

	if modVer != "" {
		_, err := version.NewVersion(modVer)
		if err != nil {
			return err
		}
	}

	workDir, err := filepath.Abs(opts.WorkDir)
	if err != nil {
		return err
	}

	if !filepath.IsAbs(modPath) {
		modPath = filepath.Join(workDir, modPath)
	}

	if len(modName) == 0 {
		modName = filepath.Base(modPath)
	} else {
		modPath = filepath.Join(modPath, modName)
	}

	if !utils.DirExists(modPath) {
		err := os.MkdirAll(modPath, os.ModePerm)
		if err != nil {
			return err
		}
	}

	kclPkg := pkg.NewKclPkg(&opt.InitOptions{
		InitPath: modPath,
		Name:     modName,
		Version:  modVer,
	})

	return c.InitEmptyPkg(&kclPkg)
}

// createIfNotExist will create a file if it does not exist.
func (c *KpmClient) createIfNotExist(filepath string, storeFunc func() error) error {
	reporter.ReportMsgTo(fmt.Sprintf("creating new :%s", filepath), c.GetLogWriter())
	err := utils.CreateFileIfNotExist(
		filepath,
		storeFunc,
	)
	if err != nil {
		if errEvent, ok := err.(*reporter.KpmEvent); ok {
			if errEvent.Type() != reporter.FileExists {
				return err
			} else {
				reporter.ReportMsgTo(fmt.Sprintf("'%s' already exists", filepath), c.GetLogWriter())
			}
		} else {
			return err
		}
	}

	return nil
}

// InitEmptyPkg will initialize an empty kcl package.
func (c *KpmClient) InitEmptyPkg(kclPkg *pkg.KclPkg) error {
	err := c.createIfNotExist(kclPkg.ModFile.GetModFilePath(), kclPkg.ModFile.StoreModFile)
	if err != nil {
		return err
	}

	err = c.createIfNotExist(kclPkg.ModFile.GetModLockFilePath(), kclPkg.LockDepsVersion)
	if err != nil {
		return err
	}

	err = c.createIfNotExist(filepath.Join(kclPkg.ModFile.HomePath, constants.DEFAULT_KCL_FILE_NAME), kclPkg.CreateDefaultMain)
	if err != nil {
		return err
	}

	return nil
}
