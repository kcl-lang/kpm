package api

import (
	"fmt"
	"os"
	"path/filepath"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kcl-go/pkg/spec/gpyrpc"
	"kcl-lang.io/kcl-go/pkg/tools/gen"
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
	kpmcli, err := client.NewKpmClient()
	if err != nil {
		return nil, err
	}
	kclPkg, err := kpmcli.LoadPkgFromPath(pkgPath)
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
	return kpmcli.ResolvePkgDepsMetadata(pkg.pkg, true)
}

// StoreModFile stores the kcl.mod file of the package to local file system.
func (pkg *KclPackage) StoreModFile() error {
	return pkg.pkg.ModFile.StoreModFile()
}

// StoreModLockFile stores the kcl.mod.lock file of the package to local file system.
func (pkg *KclPackage) StoreModLockFile() error {
	return pkg.pkg.LockDepsVersion()
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
func (pkg *KclPackage) GetFullSchemaTypeMappingWithFilters(kpmcli *client.KpmClient, filterFuncs []KclTypeFilterFunc) (map[string]map[string]*KclType, error) {
	schemaTypes := make(map[string]map[string]*KclType)
	err := filepath.Walk(pkg.GetPkgHomePath(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			filteredTypeMap := make(map[string]*KclType)

			depsMap, err := kpmcli.ResolveDepsIntoMap(pkg.pkg)
			if err != nil {
				return err
			}

			opts := kcl.NewOption()
			for depName, depPath := range depsMap {
				opts.Merge(kcl.WithExternalPkgs(fmt.Sprintf(constants.EXTERNAL_PKGS_ARG_PATTERN, depName, depPath)))
			}

			schemaTypeMap, err := kcl.GetFullSchemaTypeMapping([]string{path}, "", *opts)
			if err != nil && err.Error() != errors.NoKclFiles.Error() {
				return err
			}

			relPath, err := filepath.Rel(pkg.GetPkgHomePath(), path)
			if err != nil {
				return err
			}
			if len(schemaTypeMap) != 0 && schemaTypeMap != nil {
				for kName, kType := range schemaTypeMap {
					kTy := NewKclTypes(kName, relPath, kType)
					filterPassed := true
					for _, filterFunc := range filterFuncs {
						if !filterFunc(kTy) {
							filterPassed = false
							break
						}
					}
					if filterPassed {
						filteredTypeMap[kName] = kTy
					}
				}
				if len(filteredTypeMap) > 0 {
					schemaTypes[relPath] = filteredTypeMap
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
func (pkg *KclPackage) GetSchemaTypeMappingWithFilters(filterFuncs []KclTypeFilterFunc) (map[string]map[string]*KclType, error) {
	schemaTypes := make(map[string]map[string]*KclType)
	err := filepath.Walk(pkg.GetPkgHomePath(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			filteredTypeMap := make(map[string]*KclType)
			schemaTypeMap, err := kcl.GetFullSchemaTypeMapping([]string{path}, "")
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
					for _, filterFunc := range filterFuncs {
						if !filterFunc(kTy) {
							filterPassed = false
							break
						}
					}
					if filterPassed {
						filteredTypeMap[kName] = kTy
					}
				}
				if len(filteredTypeMap) > 0 {
					schemaTypes[relPath] = filteredTypeMap
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

// ExportSwaggerV2Spec extracts the swagger v2 representation of a kcl package
// with external dependencies.
func (pkg *KclPackage) ExportSwaggerV2Spec() (*gen.SwaggerV2Spec, error) {
	spec := &gen.SwaggerV2Spec{
		Swagger:     "2.0",
		Definitions: make(map[string]*gen.KclOpenAPIType),
		Paths:       map[string]interface{}{},
		Info: gen.SpecInfo{
			Title:   pkg.GetPkgName(),
			Version: pkg.GetVersion(),
		},
	}
	pkgMapping, err := pkg.GetAllSchemaTypeMapping()
	if err != nil {
		return spec, err
	}
	// package path -> package
	for packagePath, p := range pkgMapping {
		// schema name -> schema type
		for _, t := range p {
			id := gen.SchemaId(packagePath, t.KclType)
			spec.Definitions[id] = gen.GetKclOpenAPIType(packagePath, t.KclType, false)
		}
	}
	return spec, nil
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
