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

func TestPullWithInsecureSkipTLSverify(t *testing.T) {
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

func TestInsecureSkipTLSverifyOCIRegistry(t *testing.T) {
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
