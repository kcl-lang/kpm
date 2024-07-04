package client

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/downloader"
)

func TestPull(t *testing.T) {
	pulledPath := getTestDir("test_pull")

	kpmcli, err := NewKpmClient()
	assert.NilError(t, err)

	var buf bytes.Buffer
	kpmcli.SetLogWriter(&buf)

	kPkg, err := kpmcli.Pull(
		WithLocalPath(pulledPath),
		WithPullSource(&downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Tag:  "0.0.1",
			},
		}),
	)

	pkgPath := filepath.Join(pulledPath, "oci", "ghcr.io", "kcl-lang", "helloworld", "0.0.1")
	assert.NilError(t, err)
	assert.Equal(t, kPkg.GetPkgName(), "helloworld")
	assert.Equal(t, kPkg.GetPkgVersion(), "0.0.1")
	assert.Equal(t, kPkg.HomePath, pkgPath)

	kPkg, err = kpmcli.Pull(
		WithLocalPath(pulledPath),
		WithPullSourceUrl("oci://ghcr.io/kcl-lang/helloworld?tag=0.1.0"),
	)
	pkgPath = filepath.Join(pulledPath, "oci", "ghcr.io", "kcl-lang", "helloworld", "0.1.0")
	assert.NilError(t, err)
	assert.Equal(t, kPkg.GetPkgName(), "helloworld")
	assert.Equal(t, kPkg.GetPkgVersion(), "0.1.0")
	assert.Equal(t, kPkg.HomePath, pkgPath)

	defer func() {
		err = os.RemoveAll(filepath.Join(pulledPath, "oci"))
		assert.NilError(t, err)
	}()
}
