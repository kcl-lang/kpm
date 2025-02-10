package api

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

// TestGetKclPkgPath tests the retrieval of KCL_PKG_PATH
func TestGetKclPkgPath(t *testing.T) {
	// Backup original environment variable
	originalKclPkgPath, isSet := os.LookupEnv("KCL_PKG_PATH")
	if isSet {
		defer os.Setenv("KCL_PKG_PATH", originalKclPkgPath) // Restore after test
	} else {
		defer os.Unsetenv("KCL_PKG_PATH")
	}

	// Case 1: When KCL_PKG_PATH is set
	customPath := filepath.Join(os.TempDir(), "custom_kcl_path")
	err := os.Setenv("KCL_PKG_PATH", customPath)
	assert.Equal(t, err, nil)

	path, err := GetKclPkgPath()
	assert.Equal(t, err, nil)
	assert.Equal(t, path, customPath)
	fmt.Printf("Test Case 1: Expected %v, Got %v\n", customPath, path)

	// Case 2: When KCL_PKG_PATH is not set (should return default path)
	os.Unsetenv("KCL_PKG_PATH")
	homeDir, err := os.UserHomeDir()
	assert.Equal(t, err, nil)
	expectedDefaultPath := filepath.Join(homeDir, ".kcl", "kpm")

	path, err = GetKclPkgPath()
	assert.Equal(t, err, nil)
	assert.Equal(t, path, expectedDefaultPath)
	fmt.Printf("Test Case 2: Expected %v, Got %v\n", expectedDefaultPath, path)
}
