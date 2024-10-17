package downloader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/utils"
)

func TestCache(t *testing.T) {
	testDir := getTestDir("test_cache")
	defer func() {
		err := os.RemoveAll(filepath.Join(testDir, "oci"))
		assert.Equal(t, err, nil)
	}()
	ociCache := PkgCache{
		cacheDir: testDir,
	}

	oci := &Oci{
		Reg:  "ghcr.io",
		Repo: "kcl-lang/helloworld",
		Tag:  "0.1.3",
	}

	cachePath, err := oci.GenCachePath()
	if err != nil {
		t.Fatal(err)
	}
	cachePath = filepath.Join(testDir, cachePath)

	hash, err := utils.ShortHash(utils.JoinPath("ghcr.io", "kcl-lang"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, cachePath, filepath.Join(testDir, "oci", "cache", hash, "helloworld_0.1.3"))

	cachepath, err := ociCache.Find(Source{Oci: oci})
	assert.Equal(t, cachepath, "")
	assert.Equal(t, errors.Is(err, PkgCacheNotFound), true)

	ociCache.Update(Source{Oci: oci}, func(cachePath string) error {
		tarPath := filepath.Join(testDir, "helloworld_0.1.3.tar")
		destPath := filepath.Join(cachePath, "helloworld_0.1.3.tar")
		if !utils.DirExists(destPath) {
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		}
		if err := copy.Copy(tarPath, destPath); err != nil {
			return fmt.Errorf("failed to copy tar file to directory: %w", err)
		}

		srcPath, err := oci.GenSrcCachePath()
		if err != nil {
			return err
		}

		srcFullPath := filepath.Join(testDir, srcPath)
		if !utils.DirExists(srcFullPath) {
			if err := os.MkdirAll(srcFullPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		}

		err = utils.UnTarDir(destPath, srcFullPath)
		if err != nil {
			return err
		}

		return nil
	})

	cachepath, err = ociCache.Find(Source{Oci: oci})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cachepath, filepath.Join(testDir, "oci", "src", hash, "helloworld_0.1.3"))
}
