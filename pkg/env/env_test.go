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
}

func TestSkipChecksumCheck(t *testing.T) {
	defer os.Unsetenv(KPM_NO_SUM)

	// Test exact matches
	os.Setenv(KPM_NO_SUM, "crossplane,k8s")
	assert.Equal(t, SkipChecksumCheck("crossplane"), true)
	assert.Equal(t, SkipChecksumCheck("k8s"), true)
	assert.Equal(t, SkipChecksumCheck("json_merge_patch"), false)

	// Test wildcard
	os.Setenv(KPM_NO_SUM, "*")
	assert.Equal(t, SkipChecksumCheck("anything"), true)
	assert.Equal(t, SkipChecksumCheck("crossplane"), true)

	// Test prefix wildcard
	os.Setenv(KPM_NO_SUM, "k8s-*")
	assert.Equal(t, SkipChecksumCheck("k8s-utils"), true)
	assert.Equal(t, SkipChecksumCheck("crossplane"), false)

	// Test empty environment variable
	os.Setenv(KPM_NO_SUM, "")
	assert.Equal(t, SkipChecksumCheck("crossplane"), false)
}
