package api

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/utils"
)

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

func TestRunWithOpts(t *testing.T) {
	pkgPath := getTestDir("test_run_pkg_in_path")
	opts := opt.DefaultCompileOptions()
	opts.AddEntry(filepath.Join(pkgPath, "test_kcl", "main.k"))
	opts.SetPkgPath(filepath.Join(pkgPath, "test_kcl"))
	result, err := RunPkgWithOpt(opts)
	fmt.Printf("err: %v\n", err)
	assert.Equal(t, err, nil)
	expected, _ := os.ReadFile(filepath.Join(pkgPath, "expected"))
	assert.Equal(t, utils.RmNewline(string(result.GetRawYamlResult())), utils.RmNewline(string(expected)))
	expectedJson, _ := os.ReadFile(filepath.Join(pkgPath, "expected.json"))
	assert.Equal(t, utils.RmNewline(string(result.GetRawJsonResult())), utils.RmNewline(string(expectedJson)))
}

func TestRunPkgWithOpts(t *testing.T) {
	pkgPath := getTestDir("test_run_pkg_in_path")

	result, err := RunWithOpts(
		opt.WithNoSumCheck(false),
		opt.WithEntries([]string{filepath.Join(pkgPath, "test_kcl", "main.k")}),
		opt.WithKclOption(kcl.WithWorkDir(filepath.Join(pkgPath, "test_kcl"))),
	)

	assert.Equal(t, err, nil)
	expected, _ := os.ReadFile(filepath.Join(pkgPath, "expected"))
	assert.Equal(t, utils.RmNewline(string(result.GetRawYamlResult())), utils.RmNewline(string(expected)))
	expectedJson, _ := os.ReadFile(filepath.Join(pkgPath, "expected.json"))
	assert.Equal(t, utils.RmNewline(string(result.GetRawJsonResult())), utils.RmNewline(string(expectedJson)))
}

func TestRunWithOptsAndNoSumCheck(t *testing.T) {
	pkgPath := filepath.Join(getTestDir("test_run_pkg_in_path"), "test_run_no_sum_check")
	testCases := []string{"dep_git_commit", "dep_git_tag", "dep_oci"}

	for _, testCase := range testCases {

		pathMainK := filepath.Join(pkgPath, testCase, "main.k")
		workDir := filepath.Join(pkgPath, testCase)
		modLock := filepath.Join(workDir, "kcl.mod.lock")
		expected, err := os.ReadFile(filepath.Join(pkgPath, testCase, "expected"))
		assert.Equal(t, err, nil)
		fmt.Printf("testCase: %v\n", testCase)
		res, err := RunWithOpts(
			opt.WithNoSumCheck(true),
			opt.WithEntries([]string{pathMainK}),
			opt.WithKclOption(kcl.WithWorkDir(workDir)),
		)
		fmt.Printf("err: %v\n", err)
		assert.Equal(t, err, nil)
		assert.Equal(t, utils.DirExists(modLock), false)
		assert.Equal(t, utils.RmNewline(res.GetRawYamlResult()), utils.RmNewline(string(expected)))
		assert.Equal(t, err, nil)
	}
}

func TestRunWithOptsWithNoLog(t *testing.T) {
	pkgPath := filepath.Join(getTestDir("test_run_pkg_in_path"), "test_run_with_no_log")

	defer func() {
		_ = os.Remove(filepath.Join(pkgPath, "kcl.mod.lock"))
	}()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	pathMainK := filepath.Join(pkgPath, "main.k")

	_, err := RunWithOpts(
		opt.WithLogWriter(nil),
		opt.WithEntries([]string{pathMainK}),
		opt.WithKclOption(kcl.WithWorkDir(pkgPath)),
	)
	assert.Equal(t, err, nil)
	os.Stdout = old
	w.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	assert.Equal(t, err, nil)

	assert.Equal(t, buf.String(), "")
}
