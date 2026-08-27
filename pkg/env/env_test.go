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

func TestGetKpmLogLevel(t *testing.T) {
	defer os.Unsetenv(KPM_LOG_LEVEL)

	// Empty env → empty string (parsing/default handled in reporter.ParseLogLevel).
	os.Unsetenv(KPM_LOG_LEVEL)
	assert.Equal(t, GetKpmLogLevel(), "")

	os.Setenv(KPM_LOG_LEVEL, "DEBUG")
	assert.Equal(t, GetKpmLogLevel(), "debug")

	os.Setenv(KPM_LOG_LEVEL, "  Info  ")
	assert.Equal(t, GetKpmLogLevel(), "info")

	os.Setenv(KPM_LOG_LEVEL, "error")
	assert.Equal(t, GetKpmLogLevel(), "error")
}

func TestIsRunNoCache(t *testing.T) {
	// Snapshot and restore so we don't leak the env var across tests.
	prev, hadPrev := os.LookupEnv(KPM_RUN_NO_CACHE)
	defer func() {
		if hadPrev {
			_ = os.Setenv(KPM_RUN_NO_CACHE, prev)
		} else {
			_ = os.Unsetenv(KPM_RUN_NO_CACHE)
		}
	}()

	cases := []struct {
		name string
		set  string
		want bool
	}{
		{"unset means cache-on", "", false},
		{"empty string means cache-on", " ", false},
		{"literal 1", "1", true},
		{"literal 0 (falsy)", "0", false},
		{"true (lowercase)", "true", true},
		{"TRUE (uppercase)", "TRUE", true},
		{"yes", "yes", true},
		{"on", "on", true},
		{"no (falsy)", "no", false},
		{"off (falsy)", "off", false},
		{"random text", "nope", false},
		{"padded truthy", "  1  ", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.set == "" {
				_ = os.Unsetenv(KPM_RUN_NO_CACHE)
			} else {
				_ = os.Setenv(KPM_RUN_NO_CACHE, c.set)
			}
			if got := IsRunNoCache(); got != c.want {
				t.Errorf("IsRunNoCache() with %q = %v, want %v", c.set, got, c.want)
			}
		})
	}
}
