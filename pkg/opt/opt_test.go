// Copyright 2023 The KCL Authors. All rights reserved.

package opt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKclvmOptions2Args(t *testing.T) {
	kclvmOptions := NewKclvmOpts()
	err := kclvmOptions.Validate()
	assert.NotEqual(t, err, nil)

	kclvmOptions.EntryFiles = append(kclvmOptions.EntryFiles, "test_entry_file")
	err = kclvmOptions.Validate()
	assert.Equal(t, err, nil)

	args := kclvmOptions.Args()
	assert.Equal(t, len(args), 1)
	assert.Equal(t, args[0], "test_entry_file")

	kclvmOptions.Deps["test_pkg_name"] = "test_pkg_path"
	depArgs := kclvmOptions.PkgPathMapArgs()
	assert.Equal(t, len(depArgs), 2)
	assert.Equal(t, depArgs[0], "-E")
	assert.Equal(t, depArgs[1], "test_pkg_name=test_pkg_path")

	args = kclvmOptions.Args()
	assert.Equal(t, len(args), 3)
	assert.Equal(t, args[0], "test_entry_file")
	assert.Equal(t, args[1], "-E")
	assert.Equal(t, args[2], "test_pkg_name=test_pkg_path")
}
