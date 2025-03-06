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

func testOciDownloader(t *testing.T) {
	testCases := []struct {
		name   string
		source Source
	}{
		{
			name: "Download using Digest",
			source: Source{
				Oci: &Oci{
					Reg:    "ghcr.io",
					Repo:   "kcl-lang/k8s",
					Digest: "sha256:e1317f84cb0c6188054332983e09779e5d86fbadc8702d5b4b2959ef4753367b",
				},
			},
		},
		{
			name: "Download using Tag",
			source: Source{
				Oci: &Oci{
					Reg:  "ghcr.io",
					Repo: "zong-zhe/helloworld",
					Tag:  "0.0.3",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path_oci := getTestDir("test_oci")
			if err := os.MkdirAll(path_oci, os.ModePerm); err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = os.RemoveAll(path_oci)
			}()

			ociDownloader := OciDownloader{
				Platform: "linux/amd64",
			}

			err := ociDownloader.Download(NewDownloadOptions(
				WithSource(tc.source),
				WithLocalPath(path_oci),
			))

			assert.Equal(t, err, nil)
		})
	}
}

func testGitDownloader(t *testing.T) {
	features.Enable(features.SupportNewStorage)
	path_git := getTestDir("test_git_bare_repo")
	if err := os.MkdirAll(path_git, os.ModePerm); err != nil {
		t.Fatal(err)
	}

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

func TestWithGlobalLock(t *testing.T) {
	test.RunTestWithGlobalLock(t, "TestOciDownloader", testOciDownloader)
	test.RunTestWithGlobalLock(t, "TestGitDownloader", testGitDownloader)
}
