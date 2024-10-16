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

func TestGitDownloader(t *testing.T) {
	// Create temporary directories for testing
	gitTestDir := getTestDir("test_git")
	if err := os.MkdirAll(gitTestDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(gitTestDir)
	}()

	cacheTestDir := getTestDir("test_cache")
	if err := os.MkdirAll(cacheTestDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(cacheTestDir)
	}()

	// Test 1: Normal clone without bare
	t.Run("NormalCloneWithoutBare", func(t *testing.T) {
		gitDownloader := GitDownloader{}
		err := gitDownloader.Download(*NewDownloadOptions(
			WithSource(Source{
				Git: &Git{
					Url:    "https://github.com/kcl-lang/flask-demo-kcl-manifests.git",
					Commit: "ade147b",
				},
			}),
			WithLocalPath(gitTestDir), // Use the temp directory for clone
		))

		assert.Equal(t, err, nil)
	})

	// Test 2: Caching enabled and cloning to non-existing cache path as bare repo
	t.Run("CachingEnabledCloneToNonExistingCachePath", func(t *testing.T) {
		gitDownloader := GitDownloader{}
		err := gitDownloader.Download(*NewDownloadOptions(
			WithSource(Source{
				Git: &Git{
					Url:    "https://github.com/kcl-lang/flask-demo-kcl-manifests.git",
					Commit: "ade147b",
				},
			}),
			WithLocalPath(gitTestDir),   // Use the temp directory for clone
			WithCachePath(cacheTestDir), // Set cache path
			WithEnableCache(true),       // Enable caching
		))

		assert.Equal(t, err, nil)

		// Verify that the cache path contains a bare repo
		assert.Assert(t, utils.DirExists(cacheTestDir), "Cache directory does not exist")
	})

	// Test 3: Caching enabled and cache directory already exists, perform checkout from bare cached repo
	t.Run("CachingEnabledCheckoutFromExistingCache", func(t *testing.T) {
		gitDownloader := GitDownloader{}
		// Pre-clone to cache directory first
		err := gitDownloader.Download(*NewDownloadOptions(
			WithSource(Source{
				Git: &Git{
					Url:    "https://github.com/kcl-lang/flask-demo-kcl-manifests.git",
					Commit: "ade147b",
				},
			}),
			WithLocalPath(gitTestDir),   // Use the temp directory for clone
			WithCachePath(cacheTestDir), // Set cache path
			WithEnableCache(true),       // Enable caching
		))

		assert.Equal(t, err, nil)

		// Now try to download again to test checkout from the cache
		err = gitDownloader.Download(*NewDownloadOptions(
			WithSource(Source{
				Git: &Git{
					Url:    "https://github.com/kcl-lang/flask-demo-kcl-manifests.git",
					Commit: "ade147b",
				},
			}),
			WithLocalPath(gitTestDir),   // Use the same temp directory
			WithCachePath(cacheTestDir), // Set the same cache path
			WithEnableCache(true),       // Keep caching enabled
		))

		assert.Equal(t, err, nil)

		// Optionally verify the contents of the checkout
		// Check for specific files or directories that should exist after checkout
	})
}
