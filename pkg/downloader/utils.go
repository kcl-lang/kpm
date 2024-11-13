package downloader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/hashicorp/go-version"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/utils"
)

func loadModSpecFromKclMod(kclModPath string) (*ModSpec, error) {
	type Package struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	}
	type ModFile struct {
		Package Package `toml:"package"`
	}

	var modFile ModFile
	_, err := toml.DecodeFile(kclModPath, &modFile)
	if err != nil {
		return nil, err
	}

	return &ModSpec{
		Name:    modFile.Package.Name,
		Version: modFile.Package.Version,
	}, nil
}

// MatchesPackageName checks whether the package name in the kcl.mod file under 'kclModPath' is equal to 'targetPackage'.
func matchesPackageSpec(kclModPath string, modSpec *ModSpec) bool {
	type Package struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	}
	type ModFile struct {
		Package Package `toml:"package"`
	}

	var modFile ModFile
	_, err := toml.DecodeFile(kclModPath, &modFile)
	if err != nil {
		fmt.Printf("Error parsing kcl.mod file: %v\n", err)
		return false
	}

	return modFile.Package.Name == modSpec.Name && modFile.Package.Version == modSpec.Version
}

func FindPackageByModSpec(root string, modSpec *ModSpec) (string, error) {
	var result string
	var modVersion string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			kclModPath := filepath.Join(path, constants.KCL_MOD)
			if _, err := os.Stat(kclModPath); err == nil {
				// If the package name and version are specified,
				// we can directly check if the kcl.mod file matches the package.
				if matchesPackageSpec(kclModPath, modSpec) {
					result = path
					return filepath.SkipAll
				} else if modSpec.Version == "" {
					// If the package name specified, but version are not specified,
					if utils.MatchesPackageName(kclModPath, modSpec.Name) {
						// load the version from the kcl.mod file
						tmpSpec, err := loadModSpecFromKclMod(kclModPath)
						if err != nil {
							return err
						}
						// Remember the local path with the highest version
						tmpVer, err := version.NewSemver(tmpSpec.Version)
						if err != nil {
							return err
						}
						if modVersion != "" {
							modVer, err := version.NewSemver(modVersion)
							if err != nil {
								return err
							}
							if tmpVer.GreaterThan(modVer) {
								modVersion = tmpSpec.Version
								result = path
							}
						} else {
							modVersion = tmpSpec.Version
							result = path
						}
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}
	if result == "" {
		return "", fmt.Errorf("kcl.mod with package '%s:%s' not found", modSpec.Name, modSpec.Version)
	}
	return result, nil
}
