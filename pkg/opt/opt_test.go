// Copyright 2023 The KCL Authors. All rights reserved.

package opt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kcl-go/pkg/kcl"
)

func TestWorkDirAsPkgPath(t *testing.T) {
	opts := DefaultCompileOptions()
	assert.Equal(t, opts.PkgPath(), "")
	opts.Merge(kcl.WithWorkDir("test_work_dir"))
	assert.Equal(t, opts.PkgPath(), "test_work_dir")
	opts.ExtendEntries([]string{"file1.k", "file2.k"})
	opts.ExtendEntries([]string{"file3.k", "file4.k"})
	assert.Equal(t, opts.Entries(), []string{"file1.k", "file2.k", "file3.k", "file4.k"})
	opts.SetEntries([]string{"override.k"})
	assert.Equal(t, opts.Entries(), []string{"override.k"})
}
