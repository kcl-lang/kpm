package api

import (
	"fmt"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/opt"
)

func TestPackageApi(t *testing.T) {
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

	dep, _ := pkg.GetDependencies().Deps.Get("k8s")
	assert.Equal(t, dep.Name, "k8s")
	assert.Equal(t, dep.FullName, "k8s_1.27")
	assert.Equal(t, dep.Version, "1.27")
	assert.Equal(t, dep.Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Oci.Repo, "kcl-lang/k8s")
	assert.Equal(t, dep.Source.Oci.Tag, "1.27")

	assert.Equal(t, dep.GetLocalFullPath(""), filepath.Join(kcl_pkg_path, "k8s_1.27"))

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
	assert.Equal(t, dep.Source.Registry.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Registry.Oci.Repo, "kcl-lang/k8s")
	assert.Equal(t, dep.Source.Registry.Oci.Tag, "1.27")
}

func TestGetAllSchemaTypesMappingNamed(t *testing.T) {
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

func TestGetSchemaTypeMappingWithFilters(t *testing.T) {
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

func TestGetEntries(t *testing.T) {
	testPath := getTestDir("test_get_entries")
	pkgPath := filepath.Join(testPath, "no_entries")
	pkg, err := GetKclPackage(pkgPath)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(pkg.GetPkgProfile().GetEntries()), 0)

	pkgPath = filepath.Join(testPath, "with_path_entries")
	pkg, err = GetKclPackage(pkgPath)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(pkg.GetPkgProfile().GetEntries()), 1)

	res, err := RunWithOpts(
		opt.WithEntries(pkg.GetPkgProfile().GetEntries()),
		opt.WithKclOption(kcl.WithWorkDir(pkgPath)),
	)

	assert.Equal(t, err, nil)
	assert.Equal(t, res.GetRawYamlResult(), "sub: test in sub")
	assert.Equal(t, res.GetRawJsonResult(), "{\"sub\": \"test in sub\"}")
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

func TestXXX(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_kpm_package"), "export_swagger", "aaa")
	pkg, _ := GetKclPackage(pkg_path)
	// tys, _ := pkg.GetAllSchemaTypeMapping()
	kpmcli, _ := client.NewKpmClient()

	tys, _ := pkg.GetFullSchemaTypeMappingWithFilters(
		kpmcli,
		[]KclTypeFilterFunc{},
	)
	for k, v := range tys {
		fmt.Println("key:"+k, fmt.Sprintf("val: %s", v))
	}
}
