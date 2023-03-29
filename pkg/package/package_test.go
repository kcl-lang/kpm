package pkg

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	modfile "kusionstack.io/kpm/pkg/mod"
	"kusionstack.io/kpm/pkg/opt"
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

// Load the kcl package from directory containing kcl.mod and kcl.mod.lock file.
func TestLoadKclPkg(t *testing.T) {
	testDir := initTestDir("test_data_modfile")
	kclPkg, err := LoadKclPkg(testDir)
	if err == nil && kclPkg != nil {
		t.Errorf("Failed to 'LoadKclPkg'.")
	}

	_ = modfile.NewModFile(&opt.InitOptions{Name: "test_name", InitPath: testDir}).Store()

	kclPkg, err = LoadKclPkg(testDir)
	if err != nil {
		t.Errorf("Failed to 'LoadKclPkg'.")
	}
	assert.Equal(t, kclPkg.modFile.Pkg.Name, "test_name")
	assert.Equal(t, kclPkg.modFile.Pkg.Version, "0.0.1")
	assert.Equal(t, kclPkg.modFile.Pkg.Edition, "0.0.1")
}

// InitEmptyModule inits an empty kcl module and create a default kcl.modfile.
func TestInitEmptyPkg(t *testing.T) {
	testDir := initTestDir("test_data_modfile")
	kclPkg := NewKclPkg(&opt.InitOptions{Name: "test_name", InitPath: testDir})
	err := kclPkg.InitEmptyPkg()
	if err != nil {
		t.Errorf("Failed to 'InitEmptyPkg'.")
	}

	testKclPkg, err := LoadKclPkg(testDir)
	if err != nil {
		t.Errorf("Failed to 'LoadKclPkg'.")
	}

	assert.Equal(t, testKclPkg.modFile.Pkg.Name, "test_name")
	assert.Equal(t, testKclPkg.modFile.Pkg.Version, "0.0.1")
	assert.Equal(t, testKclPkg.modFile.Pkg.Edition, "0.0.1")
}

func TestLockDepsVersion(t *testing.T) {
	testDir := initTestDir("test_data_add_deps")
	// Init an empty package
	kclPkg := NewKclPkg(&opt.InitOptions{
		Name:     "test_add_deps",
		InitPath: testDir,
	})

	_ = kclPkg.InitEmptyPkg()

	kclPkg.Dependencies.Deps["test"] = modfile.Dependency{
		Name:    "test",
		Version: "test_version",
		Sum:     "test_sum",
	}

	err := kclPkg.LockDepsVersion()

	if err != nil {
		t.Errorf("failed to LockDepsVersion.")
	}

	expectDir := getTestDir("expected")

	if gotKclMod, err := ioutil.ReadFile(filepath.Join(testDir, "kcl.mod")); os.IsNotExist(err) {
		t.Errorf("failed to find kcl.mod.")
	} else {
		assert.Equal(t, len(kclPkg.Dependencies.Deps), 1)
		assert.Equal(t, len(kclPkg.modFile.Deps), 0)
		expectKclMod, _ := ioutil.ReadFile(filepath.Join(expectDir, "kcl.mod"))
		assert.Equal(t, string(gotKclMod), string(expectKclMod))
	}

	if gotKclModLock, err := ioutil.ReadFile(filepath.Join(testDir, "kcl.mod.lock")); os.IsNotExist(err) {
		t.Errorf("failed to find kcl.mod.lock.")
	} else {
		assert.Equal(t, len(kclPkg.Dependencies.Deps), 1)
		assert.Equal(t, len(kclPkg.modFile.Deps), 0)
		expectKclModLock, _ := ioutil.ReadFile(filepath.Join(expectDir, "kcl.mod.lock"))
		assert.Equal(t, string(gotKclModLock), string(expectKclModLock))
	}
}
