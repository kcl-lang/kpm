package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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

func load(t *testing.T, pkgPath string) *pkg.KclPkg {
	pkg, err := Load(
		WithPkgPath(pkgPath),
		WithSettings(settings.GetSettings()),
	)
	if err != nil {
		t.Fatal(err)
	}

	if pkg == nil {
		t.Fatal("pkg is nil")
	}
	return pkg
}

// Test the section [package] loaded from kcl.mod
func TestLoadPackage(t *testing.T) {
	pkgPath := getTestDir("test_load_0")
	pkg := load(t, pkgPath)

	assert.Equal(t, pkg.HomePath, pkgPath)
	assert.Equal(t, pkg.ModFile.Pkg.Name, "test_load_0")
	assert.Equal(t, pkg.ModFile.Pkg.Version, "0.0.1")
	assert.ElementsMatch(t, pkg.ModFile.Pkg.Include, []string{"src/**/include.k"})
	assert.ElementsMatch(t, pkg.ModFile.Pkg.Exclude, []string{"src/**/excluded.k"})
	assert.Equal(t, pkg.ModFile.Pkg.Description, "A test package for the loader")
	assert.Equal(t, pkg.ModFile.Dependencies.Deps.Len(), 0)
}

// Test load the dependency
// 'helloworld = 0.1.2'
func TestLoadDefaultDep(t *testing.T) {
	pkgPath := getTestDir("test_load_1")
	pkg := load(t, pkgPath)

	assert.Equal(t, pkg.Dependencies.Deps.Len(), 1)
	assert.Equal(t, pkg.ModFile.Dependencies.Deps.Len(), 1)

	// test dep from kcl.mod
	dep, _ := pkg.Dependencies.Deps.Get("helloworld")
	assert.Equal(t, dep.Name, "helloworld")
	assert.Equal(t, dep.FullName, "helloworld_0.1.2")
	assert.Equal(t, dep.Version, "0.1.2")
	assert.Equal(t, dep.Source.Registry.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Registry.Oci.Repo, "kcl-lang/helloworld")

	// test dep from kcl.mod.lock
	dep, _ = pkg.Dependencies.Deps.Get("helloworld")
	assert.Equal(t, dep.Name, "helloworld")
	assert.Equal(t, dep.FullName, "helloworld_0.1.2")
	assert.Equal(t, dep.Version, "0.1.2")
	assert.Equal(t, dep.Source.Registry.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Registry.Oci.Repo, "kcl-lang/helloworld")
}

// Test load the dependency
// 'helloworld = { oci = "oci://ghcr.io/kcl-lang/helloworld", tag = "0.1.1" }'
func TestLoadOciDep(t *testing.T) {
	pkgPath := getTestDir("test_load_2")
	pkg := load(t, pkgPath)

	assert.Equal(t, pkg.Dependencies.Deps.Len(), 1)
	assert.Equal(t, pkg.ModFile.Dependencies.Deps.Len(), 1)

	// test dep from kcl.mod
	dep, _ := pkg.Dependencies.Deps.Get("helloworld")
	assert.Equal(t, dep.Name, "helloworld")
	assert.Equal(t, dep.FullName, "helloworld_0.1.1")
	assert.Equal(t, dep.Version, "0.1.1")
	assert.Equal(t, dep.Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Oci.Repo, "kcl-lang/helloworld")

	// test dep from kcl.mod.lock
	dep, _ = pkg.Dependencies.Deps.Get("helloworld")
	assert.Equal(t, dep.Name, "helloworld")
	assert.Equal(t, dep.FullName, "helloworld_0.1.1")
	assert.Equal(t, dep.Version, "0.1.1")
	assert.Equal(t, dep.Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Oci.Repo, "kcl-lang/helloworld")
}

// Test load the dependency
// flask-demo-kcl-manifests = { git = "https://github.com/kcl-lang/flask-demo-kcl-manifests.git", commit = "ade147b" }
func TestLoadGitDep(t *testing.T) {
	pkgPath := getTestDir("test_load_3")
	pkg := load(t, pkgPath)

	assert.Equal(t, pkg.Dependencies.Deps.Len(), 1)
	assert.Equal(t, pkg.ModFile.Dependencies.Deps.Len(), 1)

	// test dep from kcl.mod
	dep, _ := pkg.ModFile.Dependencies.Deps.Get("flask-demo-kcl-manifests")
	assert.Equal(t, dep.Name, "flask-demo-kcl-manifests")
	assert.Equal(t, dep.FullName, "flask-demo-kcl-manifests_ade147b")
	assert.Equal(t, dep.Version, "ade147b")
	assert.Equal(t, dep.Source.Git.Url, "https://github.com/kcl-lang/flask-demo-kcl-manifests.git")
	assert.Equal(t, dep.Source.Git.Commit, "ade147b")

	// test dep from kcl.mod.lock
	dep, _ = pkg.Dependencies.Deps.Get("flask-demo-kcl-manifests")
	assert.Equal(t, dep.Name, "flask-demo-kcl-manifests")
	assert.Equal(t, dep.FullName, "flask_manifests_0.0.1")
	assert.Equal(t, dep.Version, "0.0.1")
	assert.Equal(t, dep.Source.Git.Url, "https://github.com/kcl-lang/flask-demo-kcl-manifests.git")
	assert.Equal(t, dep.Source.Git.Commit, "ade147b")
}
