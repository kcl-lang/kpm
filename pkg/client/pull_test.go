package client

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/downloader"
)

func testPull(t *testing.T) {
	pulledPath := getTestDir("test_pull")
	defer func() {
		err := os.RemoveAll(filepath.Join(pulledPath, "oci"))
		assert.NilError(t, err)
	}()

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
	err = os.RemoveAll(filepath.Join(pulledPath, "oci"))
	assert.NilError(t, err)

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

func testPullWithInsecureSkipTLSverify(t *testing.T) {
	pulledPath := getTestDir("test_pull")

	kpmcli, err := NewKpmClient()
	kpmcli.SetInsecureSkipTLSverify(true)
	assert.NilError(t, err)

	var buf bytes.Buffer
	kpmcli.SetLogWriter(&buf)

	kPkg, err := kpmcli.Pull(
		WithLocalPath(pulledPath),
		WithPullSourceUrl("oci://ghcr.io/kcl-lang/helloworld?tag=0.1.0"),
	)
	pkgPath := filepath.Join(pulledPath, "oci", "ghcr.io", "kcl-lang", "helloworld", "0.1.0")
	assert.NilError(t, err)
	assert.Equal(t, kPkg.GetPkgName(), "helloworld")
	assert.Equal(t, kPkg.GetPkgVersion(), "0.1.0")
	assert.Equal(t, kPkg.HomePath, pkgPath)

	defer func() {
		err = os.RemoveAll(filepath.Join(pulledPath, "oci"))
		assert.NilError(t, err)
	}()
}

func testInsecureSkipTLSverifyOCIRegistry(t *testing.T) {
	var buf bytes.Buffer

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		buf.WriteString("Called Success\n")
		fmt.Fprintln(w, "Hello, client")
	})

	mux.HandleFunc("/subpath", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from subpath")
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	fmt.Printf("ts.URL: %v\n", ts.URL)
	turl, err := url.Parse(ts.URL)
	assert.NilError(t, err)

	turl.Scheme = "oci"
	turl.Path = filepath.Join(turl.Path, "subpath")
	kpmcli, err := NewKpmClient()
	assert.NilError(t, err)
	_, _ = kpmcli.Pull(
		WithLocalPath("test"),
		WithPullSourceUrl(turl.String()),
	)

	assert.Equal(t, buf.String(), "")

	kpmcli.SetInsecureSkipTLSverify(true)
	_, _ = kpmcli.Pull(
		WithLocalPath("test"),
		WithPullSourceUrl(turl.String()),
	)

	assert.Equal(t, buf.String(), "Called Success\n")
}

func testPullWithModSpec(t *testing.T) {
	pulledPath := getTestDir("test_pull_with_modspec")
	defer func() {
		err := os.RemoveAll(filepath.Join(pulledPath, "oci"))
		assert.NilError(t, err)
	}()

	kpmcli, err := NewKpmClient()
	assert.NilError(t, err)

	var buf bytes.Buffer
	kpmcli.SetLogWriter(&buf)

	kPkg, err := kpmcli.Pull(
		WithLocalPath(pulledPath),
		WithPullSource(&downloader.Source{
			ModSpec: &downloader.ModSpec{
				Name:    "subhelloworld",
				Version: "0.0.1",
			},
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Tag:  "0.1.4",
			},
		}),
	)

	pkgPath := filepath.Join(pulledPath, "oci", "ghcr.io", "kcl-lang", "helloworld", "0.1.4", "subhelloworld", "0.0.1")
	assert.NilError(t, err)
	assert.Equal(t, kPkg.GetPkgName(), "subhelloworld")
	assert.Equal(t, kPkg.GetPkgVersion(), "0.0.1")
	assert.Equal(t, kPkg.HomePath, pkgPath)
	err = os.RemoveAll(filepath.Join(pulledPath, "oci"))
	assert.NilError(t, err)

	kPkg, err = kpmcli.Pull(
		WithLocalPath(pulledPath),
		WithPullSourceUrl("oci://ghcr.io/kcl-lang/helloworld?tag=0.1.4&mod=subhelloworld:0.0.1"),
	)
	pkgPath = filepath.Join(pulledPath, "oci", "ghcr.io", "kcl-lang", "helloworld", "0.1.4", "subhelloworld", "0.0.1")
	assert.NilError(t, err)
	assert.Equal(t, kPkg.GetPkgName(), "subhelloworld")
	assert.Equal(t, kPkg.GetPkgVersion(), "0.0.1")
	assert.Equal(t, kPkg.HomePath, pkgPath)
	err = os.RemoveAll(filepath.Join(pulledPath, "oci"))
	assert.NilError(t, err)

	_, err = kpmcli.Pull(
		WithLocalPath(pulledPath),
		WithPullSourceUrl("oci://ghcr.io/kcl-lang/helloworld?tag=0.1.4&mod=subhelloworld:0.0.2"),
	)
	assert.Equal(t, err.Error(), "version mismatch: 0.0.1 != 0.0.2, version 0.0.2 not found")
}

func testPullWithOnlySpec(t *testing.T) {
	pulledPath := getTestDir("test_pull_with_only_modspec")
	defer func() {
		err := os.RemoveAll(filepath.Join(pulledPath, "oci"))
		assert.NilError(t, err)
	}()

	kpmcli, err := NewKpmClient()
	assert.NilError(t, err)

	var buf bytes.Buffer
	kpmcli.SetLogWriter(&buf)

	kPkg, err := kpmcli.Pull(
		WithLocalPath(pulledPath),
		WithPullSource(&downloader.Source{
			ModSpec: &downloader.ModSpec{
				Name:    "helloworld",
				Version: "0.1.4",
			},
		}),
	)

	pkgPath := filepath.Join(pulledPath, "oci", "ghcr.io", "kcl-lang", "helloworld", "0.1.4", "helloworld", "0.1.4")
	assert.NilError(t, err)
	assert.Equal(t, kPkg.GetPkgName(), "helloworld")
	assert.Equal(t, kPkg.GetPkgVersion(), "0.1.4")
	assert.Equal(t, kPkg.HomePath, pkgPath)
	err = os.RemoveAll(filepath.Join(pulledPath, "oci"))
	assert.NilError(t, err)
}
