package visitor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/settings"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func TestVisitPkgDir(t *testing.T) {
	pkgDir := getTestDir("test_visit_dir")
	pVisitor := PkgVisitor{}
	source, err := downloader.NewSourceFromStr(pkgDir)
	if err != nil {
		t.Fatal(err)
	}

	err = pVisitor.Visit(source, func(pkg *pkg.KclPkg) error {
		assert.Equal(t, pkg.GetPkgName(), "test_visit_dir")
		assert.Equal(t, pkg.GetPkgVersion(), "0.0.1")
		return nil
	})
	assert.NilError(t, err)
}

func TestVisitPkgTar(t *testing.T) {
	pkgTar := filepath.Join(getTestDir("test_visit_tar"), "test_visit_tar-0.0.1.tar")
	pVisitor := PkgVisitor{}
	source, err := downloader.NewSourceFromStr(pkgTar)
	if err != nil {
		t.Fatal(err)
	}

	err = pVisitor.Visit(source, func(pkg *pkg.KclPkg) error {
		assert.Equal(t, pkg.GetPkgName(), "test_visit_tar")
		assert.Equal(t, pkg.GetPkgVersion(), "0.0.1")
		return nil
	})
	assert.NilError(t, err)
}

func TestVisitPkgRemote(t *testing.T) {
	var buf bytes.Buffer
	remotePkgVisitor := RemoteVisitor{
		PkgVisitor: &PkgVisitor{
			LogWriter: &buf,
			Settings:  settings.GetSettings(),
		},
		Downloader: &downloader.DepDownloader{},
	}

	tests := []struct {
		sourceStr       string
		expectedPkgName string
		expectedPkgVer  string
		expectedLog     string
	}{
		{
			sourceStr:       "oci://ghcr.io/kcl-lang/helloworld?tag=0.1.2",
			expectedPkgName: "helloworld",
			expectedPkgVer:  "0.1.2",
			expectedLog:     "downloading 'kcl-lang/helloworld:0.1.2' from 'ghcr.io/kcl-lang/helloworld:0.1.2'\n",
		},
		{
			sourceStr:       "git://github.com/kcl-lang/flask-demo-kcl-manifests.git?branch=main",
			expectedPkgName: "flask_manifests",
			expectedPkgVer:  "0.0.1",
			expectedLog:     "cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with branch 'main'\n",
		},
	}

	for _, tt := range tests {
		buf.Reset()
		source, err := downloader.NewSourceFromStr(tt.sourceStr)
		if err != nil {
			t.Fatal(err)
		}

		err = remotePkgVisitor.Visit(source, func(pkg *pkg.KclPkg) error {
			assert.Equal(t, pkg.GetPkgName(), tt.expectedPkgName)
			assert.Equal(t, pkg.GetPkgVersion(), tt.expectedPkgVer)
			return nil
		})
		assert.Equal(t, buf.String(), tt.expectedLog)
		assert.NilError(t, err)
	}
}

// TestVisitPkgRemoteHostless verifies that a host-less OCI dependency
// (`repo = "..."` in kcl.mod, no registry host) resolves the registry host
// from KPM_REG at runtime instead of failing with "hostName is empty".
func TestVisitPkgRemoteHostless(t *testing.T) {
	oldReg := os.Getenv("KPM_REG")
	err := os.Setenv("KPM_REG", "ghcr.io")
	assert.NilError(t, err)
	defer func() {
		_ = os.Setenv("KPM_REG", oldReg)
	}()

	var buf bytes.Buffer
	remotePkgVisitor := RemoteVisitor{
		PkgVisitor: &PkgVisitor{
			LogWriter: &buf,
			Settings:  settings.GetSettings(),
		},
		Downloader: &downloader.DepDownloader{},
	}

	source, err := downloader.NewSourceFromStr("oci:///kcl-lang/helloworld?tag=0.1.2")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, source.Oci.Reg, "")
	assert.Equal(t, source.Oci.RegFromEnv, true)

	err = remotePkgVisitor.Visit(source, func(pkg *pkg.KclPkg) error {
		assert.Equal(t, pkg.GetPkgName(), "helloworld")
		assert.Equal(t, pkg.GetPkgVersion(), "0.1.2")
		return nil
	})
	assert.NilError(t, err)
}

// TestVisitPkgRemoteHostlessNoRegistryConfigured verifies that a host-less OCI
// dependency (no host in the URI) legitimately fails when there is no KPM_REG
// env var and no default registry configured, instead of silently resolving
// to some unintended host.
func TestVisitPkgRemoteHostlessNoRegistryConfigured(t *testing.T) {
	oldReg := os.Getenv("KPM_REG")
	err := os.Unsetenv("KPM_REG")
	assert.NilError(t, err)
	defer func() {
		_ = os.Setenv("KPM_REG", oldReg)
	}()

	// Start from the real settings (valid CredentialsFile path) but with no
	// default registry configured, simulating a fresh environment with no
	// KPM_REG and no `kpm.json` default.
	noRegSettings := *settings.GetSettings()
	noRegSettings.Conf.DefaultOciRegistry = ""

	var buf bytes.Buffer
	remotePkgVisitor := RemoteVisitor{
		PkgVisitor: &PkgVisitor{
			LogWriter: &buf,
			Settings:  &noRegSettings,
		},
		Downloader: &downloader.DepDownloader{},
	}

	source, err := downloader.NewSourceFromStr("oci:///kcl-lang/helloworld?tag=0.1.2")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, source.Oci.Reg, "")
	assert.Equal(t, source.Oci.RegFromEnv, true)

	err = remotePkgVisitor.Visit(source, func(pkg *pkg.KclPkg) error { return nil })
	assert.Error(t, err, "hostName is empty")
}

func TestVisitedSpace(t *testing.T) {
	var buf bytes.Buffer
	remotePkgVisitor := RemoteVisitor{
		PkgVisitor: &PkgVisitor{
			LogWriter: &buf,
			Settings:  settings.GetSettings(),
		},
		VisitedSpace: getTestDir("test_visited_space"),
		Downloader:   &downloader.DepDownloader{},
	}

	source, err := downloader.NewSourceFromStr("oci://ghcr.io/kcl-lang/helloworld?tag=0.1.2")
	if err != nil {
		t.Fatal(err)
	}

	err = remotePkgVisitor.Visit(source, func(pkg *pkg.KclPkg) error {
		assert.Equal(t, pkg.GetPkgName(), "helloworld")
		assert.Equal(t, pkg.GetPkgVersion(), "0.1.2")
		assert.Equal(t, pkg.HomePath, source.LocalPath((remotePkgVisitor.VisitedSpace)))
		return nil
	})
	assert.NilError(t, err)
}

func TestVisitedPkgWithDefaultVersion(t *testing.T) {
	var buf bytes.Buffer
	remotePkgVisitor := RemoteVisitor{
		PkgVisitor: &PkgVisitor{
			LogWriter: &buf,
			Settings:  settings.GetSettings(),
		},
		Downloader: &downloader.DepDownloader{},
	}

	buf.Reset()
	source, err := downloader.NewSourceFromStr("oci://ghcr.io/kcl-lang/helloworld")
	if err != nil {
		t.Fatal(err)
	}

	source.ModSpec = &downloader.ModSpec{
		Name: "subhelloworld",
	}

	err = remotePkgVisitor.Visit(source, func(pkg *pkg.KclPkg) error { return nil })
	assert.Equal(t, source.ModSpec.Version, "0.0.1")
	assert.Equal(t, source.Oci.Tag, "0.1.4")
	assert.NilError(t, err)
}
