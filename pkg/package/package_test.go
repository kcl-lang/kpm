package pkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/runner"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func initTestDir(subDir string) string {
	testDir := getTestDir(subDir)
	// clean the test data
	_ = os.RemoveAll(testDir)
	_ = os.Mkdir(testDir, 0755)

	return testDir
}

func TestLoadKclPkg(t *testing.T) {
	testDir := initTestDir("test_init_empty_mod")
	kclPkg, err := LoadKclPkg(testDir)
	if err == nil && kclPkg != nil {
		t.Errorf("Failed to 'LoadKclPkg'.")
	}

	mfile := NewModFile(&opt.InitOptions{Name: "test_name", InitPath: testDir})
	_ = mfile.StoreModFile()

	kclPkg, err = LoadKclPkg(testDir)
	if err != nil {
		t.Errorf("Failed to 'LoadKclPkg'.")
	}
	assert.Equal(t, kclPkg.ModFile.Pkg.Name, "test_name")
	assert.Equal(t, kclPkg.ModFile.Pkg.Version, "0.0.1")
	assert.Equal(t, kclPkg.ModFile.Pkg.Edition, runner.GetKclVersion())
	assert.Equal(t, kclPkg.ModFile.Dependencies.Deps.Len(), 0)
	assert.Equal(t, kclPkg.Dependencies.Deps.Len(), 0)
}

func TestCheck(t *testing.T) {
	testDir := getTestDir("test_check")
	dep := Dependency{
		FullName: "test_full_name",
		Sum:      "",
	}

	testFullDir := filepath.Join(testDir, "test_full_name")

	assert.Equal(t, check(dep, testFullDir), false)
	dep.Sum = "sdfsldk"
	assert.Equal(t, check(dep, testFullDir), false)
	dep.Sum = "okQqHgQaR1il7vOPuZPPVostthK5nUJkZAZVgXMqU3Q="
	assert.Equal(t, check(dep, testFullDir), true)
}

func TestGetPkgName(t *testing.T) {
	kclPkg := KclPkg{
		ModFile: ModFile{
			Pkg: Package{
				Name: "test",
			},
		},
	}
	assert.Equal(t, kclPkg.GetPkgName(), "test")
}

func TestValidateKpmHome(t *testing.T) {
	kclPkg := NewKclPkg(&opt.InitOptions{
		Name:     "test_name",
		InitPath: "test_home_path",
	})
	oldValue := os.Getenv(env.PKG_PATH)
	os.Setenv(env.PKG_PATH, "test_home_path")
	err := kclPkg.ValidateKpmHome(os.Getenv(env.PKG_PATH))
	assert.Equal(t, err.Error(), "environment variable KCL_PKG_PATH cannot be set to the same path as the current KCL package.\n")
	assert.Equal(t, err.Type(), reporter.InvalidKpmHomeInCurrentPkg)
	os.Setenv(env.PKG_PATH, oldValue)
}

func TestLoadKclPkgFromTar(t *testing.T) {
	testDir := getTestDir("load_kcl_tar")
	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "kcl1-v0.0.3")), false)

	kclPkg, err := LoadKclPkgFromTar(filepath.Join(testDir, "kcl1-v0.0.3.tar"))
	assert.Equal(t, err, nil)
	assert.Equal(t, kclPkg.HomePath, filepath.Join(testDir, "kcl1-v0.0.3"))
	assert.Equal(t, kclPkg.ModFile.Pkg.Name, "kcl1")
	assert.Equal(t, kclPkg.ModFile.Pkg.Edition, "0.0.1")
	assert.Equal(t, kclPkg.ModFile.Pkg.Version, "0.0.3")

	assert.Equal(t, kclPkg.ModFile.Deps.Len(), 2)
	assert.Equal(t, kclPkg.ModFile.Deps.GetOrDefault("konfig", TestPkgDependency).Name, "konfig")
	assert.Equal(t, kclPkg.ModFile.Deps.GetOrDefault("konfig", TestPkgDependency).FullName, "konfig_v0.0.1")
	assert.Equal(t, kclPkg.ModFile.Deps.GetOrDefault("konfig", TestPkgDependency).Git.Url, "https://github.com/awesome-kusion/konfig.git")
	assert.Equal(t, kclPkg.ModFile.Deps.GetOrDefault("konfig", TestPkgDependency).Git.Tag, "v0.0.1")

	assert.Equal(t, kclPkg.ModFile.Deps.GetOrDefault("oci_konfig", TestPkgDependency).Name, "oci_konfig")
	assert.Equal(t, kclPkg.ModFile.Deps.GetOrDefault("oci_konfig", TestPkgDependency).FullName, "oci_konfig_0.0.1")
	assert.Equal(t, kclPkg.ModFile.Deps.GetOrDefault("oci_konfig", TestPkgDependency).ModSpec.Version, "0.0.1")

	assert.Equal(t, kclPkg.Deps.Len(), 2)
	assert.Equal(t, kclPkg.Deps.GetOrDefault("konfig", TestPkgDependency).Name, "konfig")
	assert.Equal(t, kclPkg.Deps.GetOrDefault("konfig", TestPkgDependency).FullName, "konfig_v0.0.1")
	assert.Equal(t, kclPkg.Deps.GetOrDefault("konfig", TestPkgDependency).Git.Url, "https://github.com/awesome-kusion/konfig.git")
	assert.Equal(t, kclPkg.Deps.GetOrDefault("konfig", TestPkgDependency).Git.Tag, "v0.0.1")
	assert.Equal(t, kclPkg.Deps.GetOrDefault("konfig", TestPkgDependency).Sum, "XFvHdBAoY/+qpJWmj8cjwOwZO8a3nX/7SE35cTxQOFU=")

	assert.Equal(t, kclPkg.Deps.GetOrDefault("oci_konfig", TestPkgDependency).Name, "oci_konfig")
	assert.Equal(t, kclPkg.Deps.GetOrDefault("oci_konfig", TestPkgDependency).FullName, "oci_konfig_0.0.1")
	assert.Equal(t, kclPkg.Deps.GetOrDefault("oci_konfig", TestPkgDependency).Oci.Reg, "ghcr.io")
	assert.Equal(t, kclPkg.Deps.GetOrDefault("oci_konfig", TestPkgDependency).Oci.Repo, "kcl-lang/oci_konfig")
	assert.Equal(t, kclPkg.Deps.GetOrDefault("oci_konfig", TestPkgDependency).Oci.Tag, "0.0.1")
	assert.Equal(t, kclPkg.Deps.GetOrDefault("oci_konfig", TestPkgDependency).Sum, "sLr3e6W4RPrXYyswdOSiKqkHes1QHX2tk6SwxAPDqqo=")

	assert.Equal(t, kclPkg.GetPkgTag(), "0.0.3")
	assert.Equal(t, kclPkg.GetPkgName(), "kcl1")
	assert.Equal(t, kclPkg.GetPkgFullName(), "kcl1_0.0.3")
	assert.Equal(t, kclPkg.GetPkgTarName(), "kcl1_0.0.3.tar")

	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "kcl1-v0.0.3")), true)
	err = os.RemoveAll(filepath.Join(testDir, "kcl1-v0.0.3"))
	assert.Equal(t, err, nil)
}

// Test load package whose dependencies in kcl.mod and kcl.mod.lock is different
func TestLoadPkgFromLock(t *testing.T) {
	pkgPath := getTestDir("load_from_lock")
	kpkg, err := LoadKclPkgWithOpts(
		WithPath(pkgPath),
		WithSettings(settings.GetSettings()),
	)

	assert.Equal(t, err, nil)
	assert.Equal(t, kpkg.Dependencies.Deps.Len(), 1)
	assert.Equal(t, kpkg.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).Name, "helloworld")
	assert.Equal(t, kpkg.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).FullName, "helloworld_0.1.2")
	assert.Equal(t, kpkg.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, kpkg.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).Source.Oci.Repo, "kcl-lang/helloworld")
	assert.Equal(t, kpkg.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).Source.Oci.Tag, "0.1.2")
}

func TestLoadKclPkgWithoutSettings(t *testing.T) {
	modPath := getTestDir("load_without_settings")
	kMod, err := LoadKclPkgWithOpts(
		WithPath(modPath),
	)
	assert.Equal(t, err, nil)
	assert.Equal(t, kMod.ModFile.Dependencies.Deps.Len(), 1)
	assert.Equal(t, kMod.ModFile.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).Name, "helloworld")
	assert.Equal(t, kMod.ModFile.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).FullName, "helloworld_0.1.4")
	assert.Equal(t, kMod.ModFile.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, kMod.ModFile.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).Source.Oci.Repo, "kcl-lang/helloworld")
	assert.Equal(t, kMod.ModFile.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).Source.Oci.Tag, "0.1.4")
}

func TestLockDepsVersionNoChange(t *testing.T) {
	testDir := initTestDir("test_lock_deps_no_change")
	defer os.RemoveAll(testDir)

	// Initialize a new KclPkg
	kclPkg := NewKclPkg(&opt.InitOptions{
		Name:     "test_pkg",
		InitPath: testDir,
	})

	// Add a dependency to Dependencies
	dep := Dependency{
		Name:     "test_dep",
		FullName: "test_dep_0.1.0",
		Version:  "0.1.0",
		Sum:      "dummy_sum",
	}
	kclPkg.Dependencies.Deps.Set(dep.Name, dep)

	// Generate initial lock content
	initialLockContent, err := kclPkg.Dependencies.MarshalLockTOML()
	assert.NoError(t, err)

	// Write initial kcl.mod.lock
	lockPath := filepath.Join(testDir, "kcl.mod.lock")
	err = os.WriteFile(lockPath, []byte(initialLockContent), 0644)
	assert.NoError(t, err)

	// Get initial modification time
	initialStat, err := os.Stat(lockPath)
	assert.NoError(t, err)
	initialModTime := initialStat.ModTime()

	// Call LockDepsVersion
	err = kclPkg.LockDepsVersion()
	assert.NoError(t, err)

	// Get updated modification time and content
	updatedStat, err := os.Stat(lockPath)
	assert.NoError(t, err)
	updatedModTime := updatedStat.ModTime()
	updatedLockContent, err := os.ReadFile(lockPath)
	assert.NoError(t, err)

	// Check mod time and content remain unchanged
	assert.Equal(t, initialModTime, updatedModTime, "kcl.mod.lock should not be modified")
	assert.Equal(t, string(initialLockContent), string(updatedLockContent), "kcl.mod.lock content should remain the same")
}
