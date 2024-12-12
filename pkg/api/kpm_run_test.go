package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/test"
	"kcl-lang.io/kpm/pkg/utils"
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

	path, err = getAbsInputPath(pkgPath, "test_input_outside")
	assert.Equal(t, err, nil)
	assert.Equal(t, path, filepath.Join(pkgPath, "test_input_outside"))

	path, err = getAbsInputPath(pkgPath, "path_not_exist")
	assert.NotEqual(t, err, nil)
	assert.Equal(t, path, "")
}

func TestRunPkgInPath(t *testing.T) {
	pkgPath := getTestDir("test_run_pkg_in_path")
	opts := opt.DefaultCompileOptions()
	opts.AddEntry(filepath.Join(pkgPath, "test_kcl", "main.k"))
	opts.SetPkgPath(filepath.Join(pkgPath, "test_kcl"))
	result, err := RunPkgInPath(opts)
	assert.Equal(t, err, nil)
	expected, _ := os.ReadFile(filepath.Join(pkgPath, "expected"))
	assert.Equal(t, utils.RmNewline(string(result)), utils.RmNewline(string(expected)))
}

func TestRunPkgInPathInvalidPath(t *testing.T) {
	pkgPath := getTestDir("test_run_pkg_in_path")
	opts := opt.DefaultCompileOptions()
	opts.AddEntry(filepath.Join(pkgPath, "test_kcl", "not_exist.k"))
	opts.SetPkgPath(filepath.Join(pkgPath, "test_kcl"))
	result, err := RunPkgInPath(opts)
	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), fmt.Sprintf("failed to compile the kcl package\nCannot find the kcl file, please check the file path %s\n", filepath.Join(pkgPath, "test_kcl", "not_exist.k")))
	assert.Equal(t, result, "")
}

func TestRunPkgInPathInvalidPkg(t *testing.T) {
	pkgPath := getTestDir("test_run_pkg_in_path")
	opts := opt.DefaultCompileOptions()
	opts.SetPkgPath(pkgPath)
	opts.Merge(kcl.WithKFilenames(filepath.Join(pkgPath, "invalid_pkg", "not_exist.k")))
	result, err := RunPkgInPath(opts)
	assert.NotEqual(t, err, nil)
	assert.Equal(t, true, strings.Contains(err.Error(), fmt.Sprintf("could not load 'kcl.mod' in '%s'\n", pkgPath)))
	assert.Equal(t, result, "")
}

func TestRunTar(t *testing.T) {
	pkgPath := getTestDir("test_run_tar_in_path")
	tarPath, _ := filepath.Abs(filepath.Join(pkgPath, "test.tar"))
	untarPath := filepath.Join(pkgPath, "test")
	expectPath := filepath.Join(pkgPath, "expected")

	if utils.DirExists(untarPath) {
		os.RemoveAll(untarPath)
	}

	expectedResult, _ := os.ReadFile(expectPath)
	opts := opt.DefaultCompileOptions()
	opts.SetVendor(true)
	gotResult, err := RunTar(tarPath, opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.RmNewline(string(expectedResult)), utils.RmNewline(gotResult))
	assert.Equal(t, utils.DirExists(untarPath), true)

	if utils.DirExists(untarPath) {
		os.RemoveAll(untarPath)
	}
}

func TestRunWithWorkdir(t *testing.T) {
	pkgPath := getTestDir(filepath.Join("test_work_dir", "dev"))
	opts := opt.DefaultCompileOptions()
	opts.SetPkgPath(pkgPath)
	result, err := RunPkgInPath(opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, result, "base: base\nmain: main")
}

func TestRunWithSettingsOpts(t *testing.T) {
	pkgPath := getTestDir("test_settings")
	opts := opt.DefaultCompileOptions()
	opts.Merge(kcl.WithSettings(filepath.Join(pkgPath, "kcl.yaml")))
	opts.SetHasSettingsYaml(true)
	_, err := RunWithOpt(opts)
	assert.Equal(t, err, nil)
}

func TestRunWithSettingsOptsAndFile(t *testing.T) {
	pkgPath := getTestDir("test_settings")
	opts := opt.DefaultCompileOptions()
	opts.Merge(kcl.WithSettings(filepath.Join(pkgPath, "kcl.yaml")))
	opts.SetHasSettingsYaml(true)
	opts.SetEntries([]string{filepath.Join(pkgPath, "test.k")})
	_, err := RunWithOpt(opts)
	assert.Equal(t, err, nil)
}

func TestRunTarPkg(t *testing.T) {
	pkgPath := getTestDir("test_run_tar_in_path")
	tarPath, _ := filepath.Abs(filepath.Join(pkgPath, "test.tar"))
	untarPath := filepath.Join(pkgPath, "test")
	expectPath := filepath.Join(pkgPath, "expected")
	expectPathJson := filepath.Join(pkgPath, "expected.json")

	if utils.DirExists(untarPath) {
		os.RemoveAll(untarPath)
	}

	expectedResult, _ := os.ReadFile(expectPath)
	expectedResultJson, _ := os.ReadFile(expectPathJson)
	opts := opt.DefaultCompileOptions()
	opts.SetVendor(true)
	gotResult, err := RunTarPkg(tarPath, opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.RmNewline(string(expectedResult)), utils.RmNewline(gotResult.GetRawYamlResult()))
	assert.Equal(t, utils.RmNewline(string(expectedResultJson)), utils.RmNewline(gotResult.GetRawJsonResult()))
	assert.Equal(t, utils.DirExists(untarPath), true)

	if utils.DirExists(untarPath) {
		os.RemoveAll(untarPath)
	}
}

func testRunWithNoSumCheck(t *testing.T) {
	pkgPath := getTestDir("test_run_with_nosumcheck")
	opts := opt.DefaultCompileOptions()
	opts.SetPkgPath(pkgPath)
	opts.SetNoSumCheck(true)
	_, err := RunPkgInPath(opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), false)

	opts = opt.DefaultCompileOptions()
	opts.SetPkgPath(pkgPath)
	opts.SetNoSumCheck(false)
	_, err = RunPkgInPath(opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), true)
	defer func() {
		_ = os.Remove(filepath.Join(pkgPath, "kcl.mod.lock"))
	}()
}

func TestStoreModAndModLockFile(t *testing.T) {
	testPath := getTestDir("store_mod_and_lock")

	testDep := pkg.Dependency{
		Name:          "dep1",
		FullName:      "dep1_0.0.1",
		Version:       "0.0.1",
		Sum:           "sLr3e6W4RPrXYyswdOSiKqkHes1QHX2tk6SwxAPDqqo=",
		LocalFullPath: filepath.Join(testPath, "dep1_0.0.1"),
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/dep1",
				Tag:  "0.0.1",
			},
		},
	}

	deps := orderedmap.NewOrderedMap[string, pkg.Dependency]()
	deps.Set("dep1", testDep)

	testModFile := pkg.ModFile{
		Pkg: pkg.Package{
			Name:    "test",
			Version: "0.0.1",
			Edition: "0.0.1",
		},
		HomePath: filepath.Join(testPath, "dep1_0.0.1"),
		Dependencies: pkg.Dependencies{
			Deps: deps,
		},
	}

	testPkg := pkg.KclPkg{
		ModFile:      testModFile,
		HomePath:     filepath.Join(testPath, "dep1_0.0.1"),
		Dependencies: testModFile.Dependencies,
	}

	testPackage := KclPackage{
		pkg: &testPkg,
	}

	err := testPackage.StoreModFile()
	assert.Equal(t, err, nil)

	err = testPackage.StoreModLockFile()
	assert.Equal(t, err, nil)
}

func TestRunWithLock(t *testing.T) {
	test.RunTestWithGlobalLock(t, "TestRunWithNoSumCheck", testRunWithNoSumCheck)
}
