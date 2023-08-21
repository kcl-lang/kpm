package api

import (
	"os"
	"path/filepath"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kcl-go/pkg/spec/gpyrpc"
	pkg "kcl-lang.io/kpm/pkg/package"
)

// The KCL Package
type KclPackage struct {
	pkg *pkg.KclPkg
}

// GetKclPackage returns the kcl package infomation.
//
// 'pkgPath' is the root path of the package where the 'kcl.mod' is located in.
//
// 'kcl_pkg_path' is the path of dependencies download by kpm.
func GetKclPackage(pkgPath string) (*KclPackage, error) {
	kclPkg, err := pkg.LoadKclPkg(pkgPath)
	if err != nil {
		return nil, err
	}

	return &KclPackage{
		pkg: kclPkg,
	}, nil
}

// UpdateDependencyInPath updates the dependencies in the path.
//
// 'pkg_path' is the path of dependencies download by kpm.
func (pkg *KclPackage) UpdateDependencyInPath(pkg_path string) error {
	return pkg.pkg.ResolveDepsMetadata(pkg_path, true)
}

// GetPkgName returns the name of the package.
func (pkg *KclPackage) GetPkgName() string {
	return pkg.pkg.GetPkgName()
}

// GetPkgVersion returns the version of the package.
func (pkg *KclPackage) GetVersion() string {
	return pkg.pkg.GetPkgTag()
}

// GetPkgEdition returns the kcl compiler edition of the package.
func (pkg *KclPackage) GetEdition() string {
	return pkg.pkg.GetPkgEdition()
}

// GetDependencies returns the dependencies of the package.
func (pkg *KclPackage) GetDependencies() pkg.Dependencies {
	return pkg.pkg.Dependencies
}

// GetPkgHomePath returns the home path of the package.
func (pkg *KclPackage) GetPkgHomePath() string {
	return pkg.pkg.HomePath
}

// GetPkgProfile returns the profile of the package.
func (pkg *KclPackage) GetPkgProfile() pkg.Profile {
	return pkg.pkg.GetPkgProfile()
}

// GetAllSchemaTypeMapping returns all the schema types of the package.
//
// It will return a map of schema types, the key is the relative path to the package home path.
//
// And, the value is a map of schema types, the key is the schema name, the value is the schema type.
func (pkg *KclPackage) GetAllSchemaTypeMapping() (map[string]map[string]*gpyrpc.KclType, error) {
	return pkg.GetSchemaTypeMappingNamed("")
}

// GetSchemaTypeMappingNamed returns the schema type filtered by schema name.
//
// If 'schemaName' is empty, it will return all the schema types.
func (pkg *KclPackage) GetSchemaTypeMappingNamed(schemaName string) (map[string]map[string]*gpyrpc.KclType, error) {
	schemaTypes := make(map[string]map[string]*gpyrpc.KclType)
	err := filepath.Walk(pkg.GetPkgHomePath(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			schemaTypeMap, err := kcl.GetSchemaTypeMapping(path, "", schemaName)
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(pkg.GetPkgHomePath(), path)
			if err != nil {
				return err
			}

			schemaTypes[relPath] = schemaTypeMap
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return schemaTypes, nil
}

// GetAllSchemaType returns all the schema types of the package.
//
// It will return a map of schema types, the key is the relative path to the package home path.
//
// And, the value is a slice of schema types.
func (pkg *KclPackage) GetAllSchemaType() (map[string][]*gpyrpc.KclType, error) {
	return pkg.GetSchemaTypeNamed("")
}

// GetSchemaTypeNamed returns the schema type filtered by schema name.
//
// If 'schemaName' is empty, it will return all the schema types.
func (pkg *KclPackage) GetSchemaTypeNamed(schemaName string) (map[string][]*gpyrpc.KclType, error) {
	schemaTypes := make(map[string][]*gpyrpc.KclType)
	err := filepath.Walk(pkg.GetPkgHomePath(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			schemaType, err := kcl.GetSchemaType(path, "", schemaName)
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(pkg.GetPkgHomePath(), path)
			if err != nil {
				return err
			}

			schemaTypes[relPath] = schemaType
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return schemaTypes, nil
}
