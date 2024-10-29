package api

import (
	"fmt"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/client"
)

func TestAllSchemaTypeMappings(t *testing.T) {
	t.Run("TestPackageApi", func(t *testing.T) {
		testPackageApi(t)
		fmt.Println("TestPackageApi completed")
	})

	t.Run("TestGetAllSchemaTypesMappingNamed", func(t *testing.T) {
		testGetAllSchemaTypesMappingNamed(t)
		fmt.Println("TestGetAllSchemaTypesMappingNamed completed")
	})

	t.Run("TestGetSchemaTypeMappingWithFilters", func(t *testing.T) {
		testGetSchemaTypeMappingWithFilters(t)
		fmt.Println("TestGetSchemaTypeMappingWithFilters completed")
	})
}

func testPackageApi(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_kpm_package"), "kcl_pkg")
	kcl_pkg_path, err := GetKclPkgPath()
	assert.Equal(t, err, nil)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	pkg, err := GetKclPackage(pkg_path)
	assert.Equal(t, err, nil)
	err = kpmcli.ResolvePkgDepsMetadata(pkg.pkg, true)
	assert.Equal(t, err, nil)
	assert.Equal(t, pkg.GetPkgName(), "kcl_pkg")
	assert.Equal(t, pkg.GetVersion(), "0.0.1")
	assert.Equal(t, pkg.GetEdition(), "0.0.1")
	assert.Equal(t, pkg.GetDependencies().Deps.Len(), 1)

	dep, _ := pkg.GetDependencies().Deps.Get("helloworld")
	assert.Equal(t, dep.Name, "helloworld")
	assert.Equal(t, dep.FullName, "helloworld_0.1.3")
	assert.Equal(t, dep.Version, "0.1.3")
	assert.Equal(t, dep.Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Oci.Repo, "kcl-lang/helloworld")
	assert.Equal(t, dep.Source.Oci.Tag, "0.1.3")

	assert.Equal(t, dep.GetLocalFullPath(""), filepath.Join(kcl_pkg_path, "helloworld_0.1.3"))

	schemas, err := pkg.GetAllSchemaTypeMapping()
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 3)
	assert.Equal(t, len(schemas["."]), 2)
	assert.Equal(t, len(schemas[filepath.Join("sub")]), 1)
	assert.Equal(t, len(schemas[filepath.Join("sub", "sub1")]), 2)

	// All schema types under the root path
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaInMainK"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaInMainK"].SchemaName, "SchemaInMainK")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")

	// All schema types under the root_path/sub path
	assert.Equal(t, schemas[filepath.Join("sub")]["SchemaInSubK"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub")]["SchemaInSubK"].SchemaName, "SchemaInSubK")

	// All schema types under the root_path/sub/sub1 path
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaInSubSub1K"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaInSubSub1K"].SchemaName, "SchemaInSubSub1K")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")
}

func TestApiGetDependenciesInModFile(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_get_mod_deps"), "kcl_pkg")
	pkg, err := GetKclPackage(pkg_path)
	assert.Equal(t, err, nil)
	dep, _ := pkg.GetDependenciesInModFile().Deps.Get("k8s")
	assert.Equal(t, dep.Name, "k8s")
	assert.Equal(t, dep.FullName, "k8s_1.27")
	assert.Equal(t, dep.Version, "1.27")
	assert.Equal(t, dep.Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Oci.Repo, "kcl-lang/k8s")
	assert.Equal(t, dep.Source.Oci.Tag, "1.27")
}

func testGetAllSchemaTypesMappingNamed(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_kpm_package"), "kcl_pkg")
	pkg, err := GetKclPackage(pkg_path)
	assert.Equal(t, err, nil)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)

	err = kpmcli.ResolvePkgDepsMetadata(pkg.pkg, true)
	assert.Equal(t, err, nil)

	schemas, err := pkg.GetSchemaTypeMappingNamed("SchemaWithSameName")
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 2)
	assert.Equal(t, len(schemas["."]), 1)
	assert.Equal(t, len(schemas[filepath.Join("sub", "sub1")]), 1)

	// // All schema types under the root path
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")

	// // All schema types under the root_path/sub/sub1 path
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")
}

func testGetSchemaTypeMappingWithFilters(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_kpm_package"), "kcl_pkg")
	pkg, err := GetKclPackage(pkg_path)
	assert.Equal(t, err, nil)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	err = kpmcli.ResolvePkgDepsMetadata(pkg.pkg, true)
	assert.Equal(t, err, nil)

	filterFunc := func(kt *KclType) bool {
		return kt.Type != "schema"
	}
	schemas, err := pkg.GetSchemaTypeMappingWithFilters([]KclTypeFilterFunc{filterFunc})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 0)

	filterFunc = func(kt *KclType) bool {
		return kt.SchemaName == "SchemaWithSameName"
	}
	schemas, err = pkg.GetSchemaTypeMappingWithFilters([]KclTypeFilterFunc{filterFunc})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 2)
	assert.Equal(t, len(schemas[filepath.Join(".")]), 3)
	assert.Equal(t, len(schemas[filepath.Join("sub", "sub1")]), 1)
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_with_same_name"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_with_same_name"].SchemaName, "SchemaWithSameName")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_with_same_name_in_sub"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["schema_with_same_name_in_sub"].SchemaName, "SchemaWithSameName")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")

	filterFunc = func(kt *KclType) bool {
		return kt.SchemaName == "SchemaWithSameName" && kt.Name == "SchemaWithSameName"
	}
	schemas, err = pkg.GetSchemaTypeMappingWithFilters([]KclTypeFilterFunc{filterFunc})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 2)
	assert.Equal(t, len(schemas[filepath.Join(".")]), 1)
	assert.Equal(t, len(schemas[filepath.Join("sub", "sub1")]), 1)
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].Name, "SchemaWithSameName")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].RelPath, ".")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].SchemaName, "SchemaWithSameName")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaWithSameName"].Name, "SchemaWithSameName")
	assert.Equal(t, schemas[filepath.Join("sub", "sub1")]["SchemaWithSameName"].RelPath, filepath.Join("sub", "sub1"))
}

func TestGetFullSchemaTypeMappingWithFilters(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_kpm_package"), "get_schema_ty", "aaa")
	pkg, err := GetKclPackage(pkg_path)
	assert.Equal(t, err, nil)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	err = kpmcli.ResolvePkgDepsMetadata(pkg.pkg, true)
	assert.Equal(t, err, nil)

	filterFunc := func(kt *KclType) bool {
		return kt.Type == "schema"
	}

	schemas, err := pkg.GetFullSchemaTypeMappingWithFilters(kpmcli, []KclTypeFilterFunc{filterFunc})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 1)

	fmt.Println(schemas)

	assert.Equal(t, schemas[filepath.Join(".")]["a"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["a"].SchemaName, "B")
}

func TestGetSchemaTypeUnderEmptyDir(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_kpm_package"), "no_kcl_files")
	pkg, err := GetKclPackage(pkg_path)
	assert.Equal(t, err, nil)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	err = kpmcli.ResolvePkgDepsMetadata(pkg.pkg, true)
	assert.Equal(t, err, nil)
	schemas, err := pkg.GetSchemaTypeMappingNamed("SchemaInMain")
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 1)
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaInMain"].Type, "schema")
	assert.Equal(t, schemas[filepath.Join(".")]["SchemaInMain"].SchemaName, "SchemaInMain")
}

func TestExportSwaggerV2Spec(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_kpm_package"), "export_swagger", "aaa")
	pkg, err := GetKclPackage(pkg_path)
	assert.Equal(t, err, nil)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	err = kpmcli.ResolvePkgDepsMetadata(pkg.pkg, true)
	assert.Equal(t, err, nil)
	spec, err := pkg.ExportSwaggerV2Spec()
	assert.Equal(t, err, nil)
	assert.Equal(t, len(spec.Definitions), 1)
}
