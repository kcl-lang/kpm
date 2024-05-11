// Copyright 2023 The KCL Authors. All rights reserved.
// Deprecated: The entire contents of this file will be deprecated.
// Please use the kcl cli - https://github.com/kcl-lang/cli.

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/client"
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
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	url, err := genDefaultOciUrlForKclPkg(kclPkg, kpmcli)
	assert.Equal(t, err, nil)
	assert.Equal(t, url, "oci://ghcr.io/kcl-lang/test_gen_oci_url")
}
