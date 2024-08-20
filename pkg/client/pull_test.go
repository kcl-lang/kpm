package client

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/downloader"
)

func TestPull(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test_pull")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	kpmcli, err := NewKpmClient()
	assert.NilError(t, err)

	var buf bytes.Buffer
	kpmcli.SetLogWriter(&buf)

	kPkg, err := kpmcli.Pull(
		WithLocalPath(tmpDir),
		WithPullSource(&downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Tag:  "0.0.1",
			},
		}),
	)

	pkgPath := filepath.Join(tmpDir, "oci", "ghcr.io", "kcl-lang", "helloworld", "0.0.1")
	assert.NilError(t, err)
	assert.Equal(t, kPkg.GetPkgName(), "helloworld")
	assert.Equal(t, kPkg.GetPkgVersion(), "0.0.1")
	assert.Equal(t, kPkg.HomePath, pkgPath)

	kPkg, err = kpmcli.Pull(
		WithLocalPath(tmpDir),
		WithPullSourceUrl("oci://ghcr.io/kcl-lang/helloworld?tag=0.1.0"),
	)
	pkgPath = filepath.Join(tmpDir, "oci", "ghcr.io", "kcl-lang", "helloworld", "0.1.0")
	assert.NilError(t, err)
	assert.Equal(t, kPkg.GetPkgName(), "helloworld")
	assert.Equal(t, kPkg.GetPkgVersion(), "0.1.0")
	assert.Equal(t, kPkg.HomePath, pkgPath)

	// Handle cleanup within the same filesystem
	err = os.RemoveAll(filepath.Join(tmpDir, "oci"))
	if err != nil {
		t.Errorf("Failed to remove directory: %v", err)
	}
}
