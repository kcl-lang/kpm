package downloader

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/features"
	"kcl-lang.io/kpm/pkg/git"
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

func makeDir(t *testing.T, path string) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Fatal(err)
	}
}

func testOciDownloader(t *testing.T) {
	path_oci := getTestDir("test_oci")
	makeDir(t, path_oci)

	defer func() {
		_ = os.RemoveAll(path_oci)
	}()

	ociDownloader := OciDownloader{
		Platform: "linux/amd64",
	}

	err := ociDownloader.Download(NewDownloadOptions(
		WithSource(Source{
			Oci: &Oci{
				Reg:  "ghcr.io",
				Repo: "zong-zhe/helloworld",
				Tag:  "0.0.3",
			},
		}),
		WithLocalPath(path_oci),
	))

	assert.Equal(t, err, nil)
}

func testGitDownloader(t *testing.T) {
	features.Enable(features.SupportNewStorage)
	path_git := getTestDir("test_git_bare_repo")
	makeDir(t, path_git)

	defer func() {
		_ = os.RemoveAll(path_git)
	}()

	gitDownloader := GitDownloader{}
	gitSource := Source{
		Git: &Git{
			Url:    "https://github.com/kcl-lang/flask-demo-kcl-manifests.git",
			Commit: "ade147b",
		},
	}
	gitHash, err := gitSource.Hash()
	assert.Equal(t, err, nil)

	err = gitDownloader.Download(NewDownloadOptions(
		WithSource(gitSource),
		WithLocalPath(filepath.Join(path_git, "git", "src", gitHash)),
		WithCachePath(filepath.Join(path_git, "git", "cache", gitHash)),
		WithEnableCache(true),
	))

	fmt.Printf("err: %v\n", err)
	assert.Equal(t, err, nil)
	assert.Equal(t, git.IsGitBareRepo(filepath.Join(path_git, "git", "cache", gitHash)), true)
	assert.Equal(t, utils.DirExists(filepath.Join(path_git, "git", "src", gitHash)), true)
	assert.Equal(t, utils.DirExists(filepath.Join(path_git, "git", "src", gitHash, "kcl.mod")), true)
}

func testDepDownloaderWhenPackageCacheFolderExistsButEmpty(t *testing.T) {
	path_tmp := getTestDir("test_dep_downloader")
	makeDir(t, path_tmp)
	defer func() {
		_ = os.RemoveAll(path_tmp)
	}()

	path_local := filepath.Join(path_tmp, "package_local")
	makeDir(t, path_local)
	path_cache := filepath.Join(path_tmp, "package_cache")
	makeDir(t, path_cache)

	depDownloader := DepDownloader{
		OciDownloader: &OciDownloader{
			Platform: "linux/amd64",
		},
	}

	err := depDownloader.Download(NewDownloadOptions(
		WithSource(Source{
			Oci: &Oci{
				Reg:  "ghcr.io",
				Repo: "zong-zhe/helloworld",
				Tag:  "0.0.3",
			},
		}),
		WithLocalPath(path_local),
		WithCachePath(path_cache),
		WithEnableCache(true),
	))

	assert.Equal(t, err, nil)
	existFile, err := utils.Exists(path_local + "/kcl.mod")
	assert.NilError(t, err)
	assert.Check(t, existFile)
}

func TestWithGlobalLock(t *testing.T) {
	test.RunTestWithGlobalLock(t, "TestOciDownloader", testOciDownloader)
	test.RunTestWithGlobalLock(t, "TestGitDownloader", testGitDownloader)
	test.RunTestWithGlobalLock(t, "TestDepDownloaderWhenPackageCacheFolderExistsButEmpty",
		testDepDownloaderWhenPackageCacheFolderExistsButEmpty)
}
