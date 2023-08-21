package api

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestPackageApi(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_kpm_package"), "kcl_pkg")
	kcl_pkg_path, err := GetKclPkgPath()

	assert.Equal(t, err, nil)
	pkg, err := GetKclPackage(pkg_path)
	assert.Equal(t, err, nil)
	err = pkg.pkg.ResolveDepsMetadata(kcl_pkg_path, true)
	assert.Equal(t, err, nil)

	assert.Equal(t, err, nil)
	assert.Equal(t, pkg.GetPkgName(), "kcl_pkg")
	assert.Equal(t, pkg.GetVersion(), "0.0.1")
	assert.Equal(t, pkg.GetEdition(), "0.0.1")
	assert.Equal(t, len(pkg.GetDependencies().Deps), 1)

	dep := pkg.GetDependencies().Deps["k8s"]
	assert.Equal(t, dep.Name, "k8s")
	assert.Equal(t, dep.FullName, "k8s_1.27")
	assert.Equal(t, dep.Version, "1.27")
	assert.Equal(t, dep.Sum, "xnYM1FWHAy3m+KcQMQb2rjZouTxumqYt6FGZpu2T4yM=")
	assert.Equal(t, dep.Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Oci.Repo, "kcl-lang/k8s")
	assert.Equal(t, dep.Source.Oci.Tag, "1.27")

	assert.Equal(t, dep.GetLocalFullPath(), filepath.Join(kcl_pkg_path, "k8s_1.27"))

	schemas, err := pkg.GetAllSchemaTypeMapping()
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 3)
	assert.Equal(t, len(schemas["."]), 6)
	assert.Equal(t, len(schemas[filepath.Join("sub")]), 1)
	assert.Equal(t, len(schemas[filepath.Join("sub", "sub1")]), 2)

	// All schema types under the root path
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaInMainK"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaInMainK"].SchemaName, "SchemaInMainK")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_in_main_k"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_in_main_k"].SchemaName, "SchemaInMainK")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_in_sub_k"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_in_sub_k"].SchemaName, "SchemaInSubK")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_with_same_name"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_with_same_name"].SchemaName, "SchemaWithSameName")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_with_same_name_in_sub"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_with_same_name_in_sub"].SchemaName, "SchemaWithSameName")

	// All schema types under the root_path/sub path
	assert.Equal(t, schemas[filepath.Join("sub")]["SchemaInSubK"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub")]["SchemaInSubK"].SchemaName, "SchemaInSubK")

	// All schema types under the root_path/sub/sub1 path
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaInSubSub1K"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaInSubSub1K"].SchemaName, "SchemaInSubSub1K")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")
}

func TestGetAllSchemaTypesMappingNamed(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_kpm_package"), "kcl_pkg")
	kcl_pkg_path, err := GetKclPkgPath()

	assert.Equal(t, err, nil)
	pkg, err := GetKclPackage(pkg_path)
	assert.Equal(t, err, nil)
	err = pkg.pkg.ResolveDepsMetadata(kcl_pkg_path, true)
	assert.Equal(t, err, nil)

	schemas, err := pkg.GetSchemaTypeMappingNamed("SchemaWithSameName")
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 3)
	assert.Equal(t, len(schemas["."]), 1)
	assert.Equal(t, len(schemas[filepath.Join("sub")]), 0)
	assert.Equal(t, len(schemas[filepath.Join("sub", "sub1")]), 1)

	// // All schema types under the root path
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")

	// // All schema types under the root_path/sub/sub1 path
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")
}

func TestGetAllSchemaTypes(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_kpm_package"), "kcl_pkg")
	kcl_pkg_path, err := GetKclPkgPath()

	assert.Equal(t, err, nil)
	pkg, err := GetKclPackage(pkg_path)
	assert.Equal(t, err, nil)
	err = pkg.pkg.ResolveDepsMetadata(kcl_pkg_path, true)
	assert.Equal(t, err, nil)

	schemas, err := pkg.GetAllSchemaType()
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 3)
	assert.Equal(t, len(schemas["."]), 6)
	assert.Equal(t, len(schemas[filepath.Join("sub")]), 1)
	assert.Equal(t, len(schemas[filepath.Join("sub", "sub1")]), 2)

	// All schema types under the root path
	assert.Equal(t, schemas[filepath.Join(".")][0].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")][0].SchemaName, "SchemaInMainK")
	assert.Equal(t, schemas[filepath.Join(".")][1].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")][1].SchemaName, "SchemaWithSameName")
	assert.Equal(t, schemas[filepath.Join(".")][2].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")][2].SchemaName, "SchemaInMainK")
	assert.Equal(t, schemas[filepath.Join(".")][3].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")][3].SchemaName, "SchemaInSubK")
	assert.Equal(t, schemas[filepath.Join(".")][4].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")][4].SchemaName, "SchemaWithSameName")
	assert.Equal(t, schemas[filepath.Join(".")][5].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")][5].SchemaName, "SchemaWithSameName")

	// All schema types under the root_path/sub path
	assert.Equal(t, schemas[filepath.Join("sub")][0].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub")][0].SchemaName, "SchemaInSubK")

	// All schema types under the root_path/sub/sub1 path
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")][0].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")][0].SchemaName, "SchemaInSubSub1K")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")][1].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")][1].SchemaName, "SchemaWithSameName")
}

func TestGetAllSchemaTypesNamed(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_kpm_package"), "kcl_pkg")
	kcl_pkg_path, err := GetKclPkgPath()

	assert.Equal(t, err, nil)
	pkg, err := GetKclPackage(pkg_path)
	assert.Equal(t, err, nil)
	err = pkg.pkg.ResolveDepsMetadata(kcl_pkg_path, true)
	assert.Equal(t, err, nil)

	schemas, err := pkg.GetSchemaTypeNamed("SchemaWithSameName")
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 3)
	assert.Equal(t, len(schemas["."]), 1)
	assert.Equal(t, len(schemas[filepath.Join("sub")]), 0)
	assert.Equal(t, len(schemas[filepath.Join("sub", "sub1")]), 1)

	// // All schema types under the root path
	assert.Equal(t, schemas[filepath.Join(".")][0].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")][0].SchemaName, "SchemaWithSameName")

	// // All schema types under the root_path/sub/sub1 path
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")][0].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")][0].SchemaName, "SchemaWithSameName")
}
