package api

import (
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

// GetAllSchemaTypes returns all the schema types of the package.
func (pkg *KclPackage) GetAllSchemaTypes() (map[string]*gpyrpc.KclType, error) {
	return kcl.GetSchemaTypeMapping(pkg.GetPkgHomePath(), "", "")
}

// GetSchemaType returns the schema type filtered by schema name.
func (pkg *KclPackage) GetSchemaType(schemaName string) ([]*gpyrpc.KclType, error) {
	return kcl.GetSchemaType(pkg.GetPkgHomePath(), "", schemaName)
}
