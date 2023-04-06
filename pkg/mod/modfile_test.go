package modfile

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestModFileExists(t *testing.T) {
	testDir := initTestDir("test_data_modfile")
	// there is no 'kcl.mod' and 'kcl.mod.lock'.
	is_exist, err := ModFileExists(testDir)
	if err != nil || is_exist {
		t.Errorf("test 'ModFileExists' failed.")
	}

	is_exist, err = ModLockFileExists(testDir)
	if err != nil || is_exist {
		t.Errorf("test 'ModLockFileExists' failed.")
	}

	modFile := NewModFile(
		&opt.InitOptions{
			Name:     "test_kcl_pkg",
			InitPath: testDir,
		},
	)
	// generate 'kcl.mod' but still no 'kcl.mod.lock'.
	err = modFile.StoreModFile()

	if err != nil {
		t.Errorf("test 'Store' failed.")
	}

	is_exist, err = ModFileExists(testDir)
	if err != nil || !is_exist {
		t.Errorf("test 'Store' failed.")
	}

	is_exist, err = ModLockFileExists(testDir)
	if err != nil || is_exist {
		t.Errorf("test 'Store' failed.")
	}

	NewModFile, err := LoadModFile(testDir)
	if err != nil || NewModFile.Pkg.Name != "test_kcl_pkg" || NewModFile.Pkg.Version != "0.0.1" || NewModFile.Pkg.Edition != "0.0.1" {
		t.Errorf("test 'LoadModFile' failed.")
	}
}

func TestParseOpt(t *testing.T) {

	dep := ParseOpt(&opt.RegistryOptions{
		Git: &opt.GitOptions{
			Url:    "test.git",
			Branch: "test_branch",
			Commit: "test_commit",
			Tag:    "test_tag",
		},
	})

	assert.Equal(t, dep.Name, "test")
	assert.Equal(t, dep.FullName, "test_test_tag")
	assert.Equal(t, dep.Url, "test.git")
	assert.Equal(t, dep.Branch, "test_branch")
	assert.Equal(t, dep.Commit, "test_commit")
	assert.Equal(t, dep.Tag, "test_tag")
}

func TestLoadModFileNotExist(t *testing.T) {
	testPath := getTestDir("mod_not_exist")
	isExist, err := ModFileExists(testPath)
	assert.Equal(t, isExist, false)
	assert.Equal(t, err, nil)
}

func TestLoadLockFileNotExist(t *testing.T) {
	testPath := getTestDir("mod_not_exist")
	isExist, err := ModLockFileExists(testPath)
	assert.Equal(t, isExist, false)
	assert.Equal(t, err, nil)
}

func TestLoadModFile(t *testing.T) {
	testPath := getTestDir("load_mod_file")
	modFile, err := LoadModFile(testPath)

	assert.Equal(t, modFile.Pkg.Name, "test_add_deps")
	assert.Equal(t, modFile.Pkg.Version, "0.0.1")
	assert.Equal(t, modFile.Pkg.Edition, "0.0.1")

	assert.Equal(t, len(modFile.Dependencies.Deps), 1)
	assert.Equal(t, modFile.Dependencies.Deps["name"].Name, "name")
	assert.Equal(t, modFile.Dependencies.Deps["name"].Source.Git.Url, "test_url")
	assert.Equal(t, modFile.Dependencies.Deps["name"].Source.Git.Tag, "test_tag")
	assert.Equal(t, modFile.Dependencies.Deps["name"].FullName, "test_url_test_tag")
	assert.Equal(t, err, nil)
}

func TestLoadLockDep(t *testing.T) {
	testPath := getTestDir("load_lock_file")
	deps, err := LoadLockDeps(testPath)

	assert.Equal(t, len(deps.Deps), 1)
	assert.Equal(t, deps.Deps["name"].Name, "name")
	assert.Equal(t, deps.Deps["name"].Version, "test_version")
	assert.Equal(t, deps.Deps["name"].Sum, "test_sum")
	assert.Equal(t, deps.Deps["name"].Source.Git.Url, "test_url")
	assert.Equal(t, deps.Deps["name"].Source.Git.Tag, "test_tag")
	assert.Equal(t, deps.Deps["name"].FullName, "test_version")
	assert.Equal(t, err, nil)
}

func TestStoreModFile(t *testing.T) {
	testPath := getTestDir("store_mod_file")
	mfile := ModFile{
		HomePath: testPath,
		Pkg: Package{
			Name:    "test_name",
			Edition: "0.0.1",
			Version: "0.0.1",
		},
	}

	_ = mfile.StoreModFile()

	expect, _ := ioutil.ReadFile(filepath.Join(testPath, "expected.toml"))
	got, _ := ioutil.ReadFile(filepath.Join(testPath, "kcl.mod"))
	assert.Equal(t, got, expect)
}

func TestGetFilePath(t *testing.T) {
	testPath := getTestDir("store_mod_file")
	mfile := ModFile{
		HomePath: testPath,
	}
	assert.Equal(t, mfile.GetModFilePath(), filepath.Join(testPath, MOD_FILE))
	assert.Equal(t, mfile.GetModLockFilePath(), filepath.Join(testPath, MOD_LOCK_FILE))
}
