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

func TestLoadKclPkg(t *testing.T) {
	testDir := initTestDir("test_init_empty_mod")
	kclPkg, err := LoadKclPkg(testDir)
	if err == nil && kclPkg != nil {
		t.Errorf("Failed to 'LoadKclPkg'.")
	}

	mfile := modfile.NewModFile(&opt.InitOptions{Name: "test_name", InitPath: testDir})
	_ = mfile.StoreModFile()

	kclPkg, err = LoadKclPkg(testDir)
	if err != nil {
		t.Errorf("Failed to 'LoadKclPkg'.")
	}
	assert.Equal(t, kclPkg.modFile.Pkg.Name, "test_name")
	assert.Equal(t, kclPkg.modFile.Pkg.Version, "0.0.1")
	assert.Equal(t, kclPkg.modFile.Pkg.Edition, "0.0.1")
	assert.Equal(t, len(kclPkg.modFile.Dependencies.Deps), 0)
	assert.Equal(t, len(kclPkg.Dependencies.Deps), 0)
}

func TestInitEmptyPkg(t *testing.T) {
	testDir := initTestDir("test_init_empty_mod")
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

func TestUpdataKclModAndLock(t *testing.T) {
	testDir := initTestDir("test_data_add_deps")
	// Init an empty package
	kclPkg := NewKclPkg(&opt.InitOptions{
		Name:     "test_add_deps",
		InitPath: testDir,
	})

	_ = kclPkg.InitEmptyPkg()

	dep := modfile.Dependency{
		Name:     "name",
		FullName: "test_version",
		Version:  "test_version",
		Sum:      "test_sum",
		Source: modfile.Source{
			Git: &modfile.Git{
				Url: "test_url",
				Tag: "test_tag",
			},
		},
	}

	kclPkg.Dependencies.Deps["test"] = dep
	kclPkg.modFile.Dependencies.Deps["test"] = dep

	err := kclPkg.modFile.StoreModFile()

	if err != nil {
		t.Errorf("failed to LockDepsVersion.")
	}

	err = kclPkg.LockDepsVersion()

	if err != nil {
		t.Errorf("failed to LockDepsVersion.")
	}

	expectDir := getTestDir("expected")

	if gotKclMod, err := ioutil.ReadFile(filepath.Join(testDir, "kcl.mod")); os.IsNotExist(err) {
		t.Errorf("failed to find kcl.mod.")
	} else {
		assert.Equal(t, len(kclPkg.Dependencies.Deps), 1)
		assert.Equal(t, len(kclPkg.modFile.Deps), 1)
		expectKclMod, _ := ioutil.ReadFile(filepath.Join(expectDir, "kcl.mod"))
		assert.Equal(t, string(gotKclMod), string(expectKclMod))
	}

	if gotKclModLock, err := ioutil.ReadFile(filepath.Join(testDir, "kcl.mod.lock")); os.IsNotExist(err) {
		t.Errorf("failed to find kcl.mod.lock.")
	} else {
		assert.Equal(t, len(kclPkg.Dependencies.Deps), 1)
		assert.Equal(t, len(kclPkg.modFile.Deps), 1)
		expectKclModLock, _ := ioutil.ReadFile(filepath.Join(expectDir, "kcl.mod.lock"))
		assert.Equal(t, string(gotKclModLock), string(expectKclModLock))
	}
}

func TestCheck(t *testing.T) {
	testDir := getTestDir("test_check")
	dep := modfile.Dependency{
		FullName: "test_full_name",
		Sum:      "",
	}
	assert.Equal(t, check(dep, testDir), false)
	dep.Sum = "sdfsldk"
	assert.Equal(t, check(dep, testDir), false)
	dep.Sum = "okQqHgQaR1il7vOPuZPPVostthK5nUJkZAZVgXMqU3Q="
	assert.Equal(t, check(dep, testDir), true)
}
