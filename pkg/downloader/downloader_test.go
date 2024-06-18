package downloader

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/utils"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func TestOciDownloader(t *testing.T) {
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

	err := ociDownloader.Download(*NewDownloadOptions(
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
	assert.Equal(t, true, utils.DirExists(filepath.Join(path_oci, "artifact.tgz")))

	path_git := getTestDir("test_git")
	if err := os.MkdirAll(path_oci, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = os.RemoveAll(path_git)
	}()

	gitDownloader := GitDownloader{}

	err = gitDownloader.Download(*NewDownloadOptions(
		WithSource(Source{
			Git: &Git{
				Url:    "https://github.com/kcl-lang/flask-demo-kcl-manifests.git",
				Commit: "ade147b",
			},
		}),
		WithLocalPath(path_git),
	))

	assert.Equal(t, err, nil)
}
