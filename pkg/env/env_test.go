package env

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGetAbsPkgPath(t *testing.T) {
	// Test absolute directory
	os.Setenv(PKG_PATH, ".")
	got, err := GetAbsPkgPath()
	expect, _ := filepath.Abs(".")
	assert.Equal(t, err, nil)
	assert.Equal(t, got, expect)

	// Test sub directory
	os.Setenv(PKG_PATH, "test_subdir")
	got, err = GetAbsPkgPath()
	assert.Equal(t, got, filepath.Join(expect, "test_subdir"))
	assert.Equal(t, err, nil)

	// Test default path
	os.Setenv(PKG_PATH, "")
	got, err = GetAbsPkgPath()
	homeDir, _ := os.UserHomeDir()
	assert.Equal(t, got, filepath.Join(homeDir, ".kcl/kpm"))
	assert.Equal(t, err, nil)
}
