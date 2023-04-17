package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kusionstack.io/kpm/pkg/errors"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func TestGetAbsInputPath(t *testing.T) {
	pkgPath := getTestDir("test_abs_input")
	path, err := getAbsInputPath(filepath.Join(pkgPath, "test_pkg_path"), "test_input")
	assert.Equal(t, err, nil)
	assert.Equal(t, path, filepath.Join(filepath.Join(pkgPath, "test_pkg_path"), "test_input"))

	path, err = getAbsInputPath(pkgPath, filepath.Join("test_pkg_path", "test_input"))
	assert.Equal(t, err, nil)
	assert.Equal(t, path, filepath.Join(filepath.Join(pkgPath, "test_pkg_path"), "test_input"))

	path, err = getAbsInputPath(pkgPath, filepath.Join(pkgPath, "test_input_outside"))
	assert.Equal(t, err, nil)
	assert.Equal(t, path, filepath.Join(pkgPath, "test_input_outside"))

	path, err = getAbsInputPath(pkgPath, "path_not_exist")
	assert.NotEqual(t, err, nil)
	assert.Equal(t, path, "")
}

func TestAbsTarPath(t *testing.T) {
	pkgPath := getTestDir("test_check_tar_path")
	expectAbsTarPath, _ := filepath.Abs(filepath.Join(pkgPath, "test.tar"))

	abs, err := absTarPath(filepath.Join(pkgPath, "test.tar"))
	assert.Equal(t, err, nil)
	assert.Equal(t, abs, expectAbsTarPath)

	abs, err = absTarPath(filepath.Join(pkgPath, "no_exist.tar"))
	assert.NotEqual(t, err, nil)
	assert.Equal(t, abs, "")

	abs, err = absTarPath(filepath.Join(pkgPath, "invalid_tar"))
	assert.NotEqual(t, err, nil)
	assert.Equal(t, abs, "")
}

func TestRunPkgInPath(t *testing.T) {
	pkgPath := getTestDir("test_run_pkg_in_path")
	result, err := runPkgInPath(filepath.Join(pkgPath, "test_kcl"), "main.k", false)
	assert.Equal(t, err, nil)
	expected, _ := ioutil.ReadFile(filepath.Join(pkgPath, "expected"))
	assert.Equal(t, string(result), string(expected))
}

func TestRunPkgInPathInvalidPath(t *testing.T) {
	pkgPath := getTestDir("test_run_pkg_in_path")
	result, err := runPkgInPath(filepath.Join(pkgPath, "test_kcl"), "not_exist.k", false)
	assert.NotEqual(t, err, nil)
	assert.Equal(t, err, errors.EntryFileNotFound)
	assert.Equal(t, result, "")
}

func TestRunPkgInPathInvalidPkg(t *testing.T) {
	pkgPath := getTestDir("test_run_pkg_in_path")
	result, err := runPkgInPath(filepath.Join(pkgPath, "invalid_pkg"), "not_exist.k", false)
	assert.NotEqual(t, err, nil)
	assert.Equal(t, err, errors.FailedToLoadPackage)
	assert.Equal(t, result, "")
}
