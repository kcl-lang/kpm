package api

import (
	"fmt"
	"os"
	"path/filepath"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kcl-go/pkg/spec/gpyrpc"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/errors"
	pkg "kcl-lang.io/kpm/pkg/package"
)

// The KCL Package
type KclPackage struct {
	pkg *pkg.KclPkg
}

// The KCL Type

// An additional field 'Name' is added to the original 'KclType'.
//
// 'Name' is the name of the kcl type.
//
// 'RelPath' is the relative path to the package home path.
type KclType struct {
	Name    string
	RelPath string
	*gpyrpc.KclType
}

// NewKclTypes returns a new KclType.
func NewKclTypes(name, path string, tys *gpyrpc.KclType) *KclType {
	return &KclType{
		Name:    name,
		RelPath: path,
		KclType: tys,
	}
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
	kpmcli, err := client.NewKpmClient()
	if err != nil {
		return err
	}
	return kpmcli.ResolvePkgDepsMetadata(pkg.pkg, &pkg.pkg.Dependencies, true)
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

// GetDependenciesInModFile returns the mod file of the package.
func (pkg *KclPackage) GetDependenciesInModFile() *pkg.Dependencies {
	return &pkg.pkg.ModFile.Dependencies
}

// GetPkgHomePath returns the home path of the package.
func (pkg *KclPackage) GetPkgHomePath() string {
	return pkg.pkg.HomePath
}

// GetPkgProfile returns the profile of the package.
func (pkg *KclPackage) GetPkgProfile() *pkg.Profile {
	return pkg.pkg.GetPkgProfile()
}

// GetAllSchemaTypeMapping returns all the schema types of the package.
//
// It will return a map of schema types, the key is the relative path to the package home path.
//
// And, the value is a map of schema types, the key is the schema name, the value is the schema type.
func (pkg *KclPackage) GetAllSchemaTypeMapping() (map[string]map[string]*KclType, error) {
	cli, err := client.NewKpmClient()
	if err != nil {
		return nil, err
	}
	return pkg.GetFullSchemaTypeMappingWithFilters(cli, []KclTypeFilterFunc{IsSchemaType})
}

// GetSchemaTypeMappingNamed returns the schema type filtered by schema name.
//
// If 'schemaName' is empty, it will return all the schema types.
func (pkg *KclPackage) GetSchemaTypeMappingNamed(schemaName string) (map[string]map[string]*KclType, error) {
	namedFilterFunc := func(kt *KclType) bool {
		return IsSchemaNamed(kt, schemaName)
	}
	return pkg.GetSchemaTypeMappingWithFilters([]KclTypeFilterFunc{IsSchemaType, namedFilterFunc})
}

// GetFullSchemaTypeMappingWithFilters returns the full schema type filtered by the filter functions.
func (pkg *KclPackage) GetFullSchemaTypeMappingWithFilters(kpmcli *client.KpmClient, fileterFuncs []KclTypeFilterFunc) (map[string]map[string]*KclType, error) {
	schemaTypes := make(map[string]map[string]*KclType)
	err := filepath.Walk(pkg.GetPkgHomePath(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			fileteredKtypeMap := make(map[string]*KclType)

			depsMap, err := kpmcli.ResolveDepsIntoMap(pkg.pkg)
			if err != nil {
				return err
			}

			opts := kcl.NewOption()
			for depName, depPath := range depsMap {
				opts.Merge(kcl.WithExternalPkgs(fmt.Sprintf(constants.EXTERNAL_PKGS_ARG_PATTERN, depName, depPath)))
			}

			schemaTypeList, err := kcl.GetFullSchemaType([]string{path}, "", *opts)
			if err != nil && err.Error() != errors.NoKclFiles.Error() {
				return err
			}

			schemaTypeMap := make(map[string]*gpyrpc.KclType)
			for _, schemaType := range schemaTypeList {
				schemaTypeMap[schemaType.SchemaName] = schemaType
			}

			relPath, err := filepath.Rel(pkg.GetPkgHomePath(), path)
			if err != nil {
				return err
			}
			if len(schemaTypeMap) != 0 && schemaTypeMap != nil {
				for kName, kType := range schemaTypeMap {
					kTy := NewKclTypes(kName, relPath, kType)
					filterPassed := true
					for _, filterFunc := range fileterFuncs {
						if !filterFunc(kTy) {
							filterPassed = false
							break
						}
					}
					if filterPassed {
						fileteredKtypeMap[kName] = kTy
					}
				}
				if len(fileteredKtypeMap) > 0 {
					schemaTypes[relPath] = fileteredKtypeMap
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return schemaTypes, nil
}

// GetSchemaTypeMappingWithFilters returns the schema type filtered by the filter functions.
func (pkg *KclPackage) GetSchemaTypeMappingWithFilters(fileterFuncs []KclTypeFilterFunc) (map[string]map[string]*KclType, error) {
	schemaTypes := make(map[string]map[string]*KclType)
	err := filepath.Walk(pkg.GetPkgHomePath(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			fileteredKtypeMap := make(map[string]*KclType)
			schemaTypeMap, err := kcl.GetSchemaTypeMapping(path, "", "")
			if err != nil && err.Error() != errors.NoKclFiles.Error() {
				return err
			}

			relPath, err := filepath.Rel(pkg.GetPkgHomePath(), path)
			if err != nil {
				return err
			}
			if schemaTypeMap != nil {
				for kName, kType := range schemaTypeMap {
					kTy := NewKclTypes(kName, relPath, kType)
					filterPassed := true
					for _, filterFunc := range fileterFuncs {
						if !filterFunc(kTy) {
							filterPassed = false
							break
						}
					}
					if filterPassed {
						fileteredKtypeMap[kName] = kTy
					}
				}
				if len(fileteredKtypeMap) > 0 {
					schemaTypes[relPath] = fileteredKtypeMap
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return schemaTypes, nil
}

type KclTypeFilterFunc func(kt *KclType) bool

// IsSchema returns true if the type is schema.
func IsSchema(kt *KclType) bool {
	return kt.Type == "schema"
}

// IsSchemaType returns true if the type is schema type.
func IsSchemaType(kt *KclType) bool {
	return IsSchema(kt) && kt.SchemaName == kt.Name
}

// IsSchemaInstance returns true if the type is schema instance.
func IsSchemaInstance(kt *KclType) bool {
	return IsSchema(kt) && kt.SchemaName != kt.Name
}

// IsSchemaNamed returns true if the type is schema and the name is equal to the given name.
func IsSchemaNamed(kt *KclType, name string) bool {
	return IsSchema(kt) && kt.Name == name
}
