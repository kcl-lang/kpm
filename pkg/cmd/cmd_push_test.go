// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	pkg "kcl-lang.io/kpm/pkg/package"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func TestGenDefaultOciUrlForKclPkg(t *testing.T) {
	pkgPath := getTestDir("test_gen_oci_url")
	kclPkg, err := pkg.LoadKclPkg(pkgPath)
	assert.Equal(t, err, nil)
	url, err := genDefaultOciUrlForKclPkg(kclPkg)
	assert.Equal(t, err, nil)
	assert.Equal(t, url, "oci://ghcr.io/kcl-lang/test_gen_oci_url")
}
