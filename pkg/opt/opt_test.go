// Copyright 2023 The KCL Authors. All rights reserved.

package opt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOciOptionFromString(t *testing.T) {
	oci_ref_with_tag := "test_oci_repo:test_oci_tag"
	ociOption, err := ParseOciOptionFromString(oci_ref_with_tag, "test_tag")
	assert.Equal(t, err, nil)
	assert.Equal(t, ociOption.PkgName, "")
	assert.Equal(t, ociOption.Reg, "ghcr.io")
	assert.Equal(t, ociOption.Repo, "test_oci_repo")
	assert.Equal(t, ociOption.Tag, "test_oci_tag")

	oci_ref_without_tag := "test_oci_repo:test_oci_tag"
	ociOption, err = ParseOciOptionFromString(oci_ref_without_tag, "test_tag")
	assert.Equal(t, err, nil)
	assert.Equal(t, ociOption.PkgName, "")
	assert.Equal(t, ociOption.Reg, "ghcr.io")
	assert.Equal(t, ociOption.Repo, "test_oci_repo")
	assert.Equal(t, ociOption.Tag, "test_oci_tag")

	oci_url_with_tag := "oci://test_reg/test_oci_repo"
	ociOption, err = ParseOciOptionFromString(oci_url_with_tag, "test_tag")
	assert.Equal(t, err, nil)
	assert.Equal(t, ociOption.PkgName, "")
	assert.Equal(t, ociOption.Reg, "test_reg")
	assert.Equal(t, ociOption.Repo, "/test_oci_repo")
	assert.Equal(t, ociOption.Tag, "test_tag")
}
