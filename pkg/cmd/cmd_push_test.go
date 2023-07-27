// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	pkg "kcl-lang.io/kpm/pkg/package"
)

func TestGenDefaultOciUrlForKclPkg(t *testing.T) {
	pkgPath := getTestDir("test_gen_oci_url")
	kclPkg, err := pkg.LoadKclPkg(pkgPath)
	assert.Equal(t, err, nil)
	url, err := genDefaultOciUrlForKclPkg(kclPkg)
	assert.Equal(t, err, nil)
	assert.Equal(t, url, "oci://ghcr.io/kcl-lang/test_gen_oci_url")
}
