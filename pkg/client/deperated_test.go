package client

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/git"
	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/utils"
)

func TestRunWithNoSumCheck(t *testing.T) {
	pkgPath := getTestDir("test_run_no_sum_check")

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	opts := opt.DefaultCompileOptions()
	opts.SetNoSumCheck(true)
	opts.SetPkgPath(pkgPath)

	_, err = kpmcli.CompileWithOpts(opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), false)

	opts = opt.DefaultCompileOptions()
	opts.SetPkgPath(pkgPath)
	opts.SetNoSumCheck(false)
	_, err = kpmcli.CompileWithOpts(opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), true)

	defer func() {
		_ = os.Remove(filepath.Join(pkgPath, "kcl.mod.lock"))
	}()
}

func testRunWithGitPackage(t *testing.T) {
	pkgPath := getTestDir("test_run_git_package")

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	opts := opt.DefaultCompileOptions()
	opts.SetPkgPath(pkgPath)

	compileResult, err := kpmcli.CompileWithOpts(opts)
	assert.Equal(t, err, nil)
	expectedCompileResult := `{"apiVersion": "v1", "kind": "Pod", "metadata": {"name": "web-app"}, "spec": {"containers": [{"image": "nginx", "name": "main-container", "ports": [{"containerPort": 80}]}]}}`
	assert.Equal(t, expectedCompileResult, compileResult.GetRawJsonResult())

	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), true)

	defer func() {
		_ = os.Remove(filepath.Join(pkgPath, "kcl.mod.lock"))
	}()
}

func testRunWithOciDownloader(t *testing.T) {
	kpmCli, err := NewKpmClient()
	path := getTestDir("test_oci_downloader")
	assert.Equal(t, err, nil)

	kpmCli.DepDownloader = downloader.NewOciDownloader("linux/amd64")

	var buf bytes.Buffer
	writer := io.MultiWriter(&buf, os.Stdout)

	res, err := kpmCli.RunWithOpts(
		opt.WithEntries([]string{filepath.Join(path, "run_pkg", "pkg", "main.k")}),
		opt.WithKclOption(kcl.WithWorkDir(filepath.Join(path, "run_pkg", "pkg"))),
		opt.WithNoSumCheck(true),
		opt.WithLogWriter(writer),
	)
	assert.Equal(t, err, nil)
	strings.Contains(buf.String(), "downloading 'zong-zhe/helloworld:0.0.3' from 'ghcr.io/zong-zhe/helloworld:0.0.3'")
	assert.Equal(t, res.GetRawYamlResult(), "The_first_kcl_program: Hello World!")
}

func testRunGit(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	testPath := getTestDir("test_run_git")

	opts := opt.DefaultCompileOptions()
	gitOpts := git.NewCloneOptions("https://github.com/kcl-lang/flask-demo-kcl-manifests.git", "", "", "main", filepath.Join(testPath, "flask-demo-kcl-manifests"), nil)
	defer func() {
		_ = os.RemoveAll(filepath.Join(testPath, "flask-demo-kcl-manifests"))
	}()

	result, err := kpmcli.CompileGitPkg(gitOpts, opts)
	assert.Equal(t, err, nil)

	resultStr := result.GetRawYamlResult()

	expectedFilePath := filepath.Join(testPath, "expect.yaml")
	bytes, err := os.ReadFile(expectedFilePath)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.RmNewline(string(bytes)), utils.RmNewline(string(resultStr)))
}

func testRunOciWithSettingsFile(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kpmcli.SetLogWriter(nil)
	opts := opt.DefaultCompileOptions()
	opts.SetEntries([]string{})
	opts.Merge(kcl.WithSettings(filepath.Join(".", "test_data", "test_run_oci_with_settings", "kcl.yaml")))
	opts.SetHasSettingsYaml(true)
	_, err = kpmcli.CompileOciPkg("oci://ghcr.io/kcl-lang/helloworld", "", opts)
	assert.Equal(t, err, nil)
}

// TODO: failed because of https://github.com/kcl-lang/kcl/issues/1660, re-enable this test after the issue is fixed
// func TestRunGitWithLocalDep(t *testing.T) {
// 	kpmcli, err := NewKpmClient()
// 	assert.Equal(t, err, nil)

// 	testPath := getTestDir("test_run_git_with_local_dep")
// 	defer func() {
// 		_ = os.RemoveAll(filepath.Join(testPath, "catalog"))
// 	}()

// 	testCases := []struct {
// 		ref        string
// 		expectFile string
// 	}{
// 		{"8308200", "expect1.yaml"},
// 		{"0b3f5ab", "expect2.yaml"},
// 	}

// 	for _, tc := range testCases {

// 		expectPath := filepath.Join(testPath, tc.expectFile)
// 		opts := opt.DefaultCompileOptions()
// 		gitOpts := git.NewCloneOptions("https://github.com/kcl-lang/flask-demo-kcl-manifests.git", tc.ref, "", "", "", nil)

// 		result, err := kpmcli.CompileGitPkg(gitOpts, opts)
// 		assert.Equal(t, err, nil)

// 		fileBytes, err := os.ReadFile(expectPath)
// 		assert.Equal(t, err, nil)

// 		var expectObj map[string]interface{}
// 		err = yaml.Unmarshal(fileBytes, &expectObj)
// 		assert.Equal(t, err, nil)

// 		var gotObj map[string]interface{}
// 		err = yaml.Unmarshal([]byte(result.GetRawJsonResult()), &gotObj)
// 		assert.Equal(t, err, nil)

// 		assert.Equal(t, gotObj, expectObj)
// 	}
// }
