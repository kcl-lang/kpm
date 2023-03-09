package pkg

import (
	"os"

	modfile "kusionstack.io/kpm/pkg/mod"
	"kusionstack.io/kpm/pkg/opt"
	"kusionstack.io/kpm/pkg/reporter"
)

type KclPkg struct {
	modFile  modfile.ModFile
	HomePath string
}

func NewKclPkg(opt *opt.InitOptions) KclPkg {
	return KclPkg{
		modFile:  *modfile.NewModFile(opt),
		HomePath: opt.InitPath,
	}
}

// Load the kcl package from directory containing kcl.mod and kcl.mod.lock file.
func LoadKclPkg(pkgPath string) (*KclPkg, error) {
	modFile, err := modfile.LoadModFile(pkgPath)
	if err != nil {
		return nil, err
	}
	return &KclPkg{
		modFile:  *modFile,
		HomePath: pkgPath,
	}, nil
}

// InitEmptyModule inits an empty kcl module and create a default kcl.modfile.
func (kclPkg KclPkg) InitEmptyPkg() error {
	err := createFileIfNotExist(kclPkg.modFile.GetModFilePath(), "kcl.mod", kclPkg.modFile.Store)
	if err != nil {
		return err
	}

	err = createFileIfNotExist(kclPkg.modFile.GetModLockFilePath(), "kcl.mod.lock", kclPkg.modFile.StoreLockFile)
	if err != nil {
		return err
	}

	return nil
}

func createFileIfNotExist(filePath string, fileName string, storeFunc func() error) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		reporter.Report("kpm: creating new "+fileName+":", filePath)
		err := storeFunc()
		if err != nil {
			reporter.Report("kpm: failed to create "+fileName+",", err)
			return err
		}
	} else {
		reporter.Report("kpm: '" + filePath + "' already exists")
		return err
	}
	return nil
}
