package downloader

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	pkg "kcl-lang.io/kpm/pkg/package"
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
	ociTestDir := getTestDir("oci_test_dir")
	if err := os.MkdirAll(ociTestDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = os.RemoveAll(ociTestDir)
	}()

	downloader := OciDownloader{
		Platform: "linux/amd64",
	}

	options := NewDownloadOptions(
		WithSource(pkg.Source{
			Oci: &pkg.Oci{
				Reg:  "ghcr.io",
				Repo: "zong-zhe/helloworld",
				Tag:  "0.0.3",
			},
		})
		WithLocalPath(ociTestDir),
	)

	err := downloader.Download(*options)

	assert.Equal(t, err, nil)
	assert.Equal(t, true, utils.DirExists(filepath.Join(ociTestDir, "artifact.tgz")))

	gitTestDir := getTestDir("git_test_dir")
	if err := os.MkdirAll(gitTestDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = os.RemoveAll(gitTestDir)
	}()

	gitDownloader := GitDownloader{}

	err = gitDownloader.Download(*NewDownloadOptions(
		WithSource(pkg.Source{
			Git: &pkg.Git{
				Url:    "https://github.com/kcl-lang/flask-demo-kcl-manifests.git",
				Commit: "ade147b",
			},
		}),
		WithLocalPath(gitTestDir),
	))

	assert.Equal(t, err, nil)
	assert.Equal(t, false, utils.DirExists(filepath.Join(gitTestDir, "some_expected_file")))
}

