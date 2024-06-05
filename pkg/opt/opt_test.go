// Copyright 2023 The KCL Authors. All rights reserved.

package opt

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/settings"
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

func TestInitOptions(t *testing.T) {
	o1 := InitOptions{Name: "foo", InitPath: "bar", Version: "v0.0.1"}
	o2 := InitOptions{Name: "foo", InitPath: "bar", Version: "v0.0.2"}
	o3 := InitOptions{Name: "foo", InitPath: "bar", Version: "abc.0.3"}
	assert.Equal(t, o1.Validate(), nil)
	assert.Equal(t, o2.Validate(), nil)
	assert.Equal(t, o3.Validate(), errors.InvalidVersionFormat)
}

func TestNewRegistryOptionsFromRef(t *testing.T) {
	ref := "test:latest"
	settings := settings.GetSettings()
	opts, err := NewRegistryOptionsFrom(ref, settings)
	assert.Equal(t, err, nil)
	assert.Equal(t, opts.Oci.Tag, "latest")
	assert.Equal(t, opts.Oci.Repo, "kcl-lang/test")
	assert.Equal(t, opts.Oci.Reg, "ghcr.io")

	opts, err = NewRegistryOptionsFrom("oci://docker.io/kcllang/test1?tag=0.0.1", settings)
	assert.Equal(t, err, nil)
	assert.Equal(t, opts.Oci.Tag, "0.0.1")
	assert.Equal(t, opts.Oci.Repo, "/kcllang/test1")
	assert.Equal(t, opts.Oci.Reg, "docker.io")

	opts, err = NewRegistryOptionsFrom("ssh://github.com/kcl-lang/test1?tag=0.0.1", settings)
	assert.Equal(t, err, nil)
	assert.Equal(t, opts.Git.Tag, "0.0.1")
	assert.Equal(t, opts.Git.Url, "ssh://github.com/kcl-lang/test1")

	opts, err = NewRegistryOptionsFrom("http://github.com/kcl-lang/test1?commit=123456", settings)
	assert.Equal(t, err, nil)
	assert.Equal(t, opts.Git.Commit, "123456")
	assert.Equal(t, opts.Git.Url, "http://github.com/kcl-lang/test1")

	opts, err = NewRegistryOptionsFrom("https://github.com/kcl-lang/test1?branch=main", settings)
	assert.Equal(t, err, nil)
	assert.Equal(t, opts.Git.Branch, "main")
	assert.Equal(t, opts.Git.Url, "https://github.com/kcl-lang/test1")

	opts, err = NewRegistryOptionsFrom("git://github.com/kcl-lang/test1?branch=main", settings)
	assert.Equal(t, err, nil)
	assert.Equal(t, opts.Git.Branch, "main")
	assert.Equal(t, opts.Git.Url, "https://github.com/kcl-lang/test1")
}

func TestNewOciOptions(t *testing.T) {
	parsedUrl, err := url.Parse("oci://docker.io/kcllang/test1?tag=0.0.1")
	assert.Equal(t, err, nil)
	ociOptions := NewOciOptionsFromUrl(parsedUrl)
	assert.Equal(t, ociOptions.Tag, "0.0.1")
	assert.Equal(t, ociOptions.Repo, "/kcllang/test1")
	assert.Equal(t, ociOptions.Reg, "docker.io")

	parsedUrl, err = url.Parse("http://docker.io/kcllang/test1?tag=0.0.1")
	assert.Equal(t, err, nil)
	ociOptions = NewOciOptionsFromUrl(parsedUrl)
	assert.Equal(t, ociOptions.Tag, "0.0.1")
	assert.Equal(t, ociOptions.Repo, "/kcllang/test1")
	assert.Equal(t, ociOptions.Reg, "docker.io")

	parsedUrl, err = url.Parse("https://docker.io/kcllang/test1?tag=0.0.1")
	assert.Equal(t, err, nil)
	ociOptions = NewOciOptionsFromUrl(parsedUrl)
	assert.Equal(t, ociOptions.Tag, "0.0.1")
	assert.Equal(t, ociOptions.Repo, "/kcllang/test1")
	assert.Equal(t, ociOptions.Reg, "docker.io")
}
