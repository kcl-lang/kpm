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

	schemas, err := pkg.GetAllSchemaTypes()
	assert.Equal(t, err, nil)
	assert.Equal(t, len(schemas), 3)
	assert.Equal(t, schemas["SchemaInMainK"].Type, "schema")
	assert.Equal(t, schemas["SchemaInMainK"].SchemaName, "SchemaInMainK")
	assert.Equal(t, schemas["SchemaInMainK"].Properties["msg"].Type, "str")
	assert.Equal(t, schemas["schema_in_main_k"].Type, "schema")
	assert.Equal(t, schemas["schema_in_main_k"].SchemaName, "SchemaInMainK")
	assert.Equal(t, schemas["schema_in_main_k"].Properties["msg"].Type, "str")
	assert.Equal(t, schemas["schema_in_sub_k"].Type, "schema")
	assert.Equal(t, schemas["schema_in_sub_k"].SchemaName, "SchemaInSubK")
	assert.Equal(t, schemas["schema_in_sub_k"].Properties["msg"].Type, "str")
}
