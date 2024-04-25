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
	path := getTestDir("test_oci")

	defer func() {
		_ = os.RemoveAll(path)
	}()

	ociDownloader := OciDownloader{
		Platform: "linux/amd64",
	}

	err := ociDownloader.Download(*NewDownloadOptions(
		WithSource(pkg.Source{
			Oci: &pkg.Oci{
				Reg:  "ghcr.io",
				Repo: "zong-zhe/helloworld",
				Tag:  "0.0.3",
			},
		}),
		WithLocalPath(path),
	))

	assert.Equal(t, err, nil)
	assert.Equal(t, true, utils.DirExists(filepath.Join(path, "artifact.tgz")))
}
