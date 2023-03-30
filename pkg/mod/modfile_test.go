package modfile

import (
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
	err = modFile.Store()

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

	assert.Equal(t, dep.Name, "test", "unexpected dep name")
	assert.Equal(t, dep.Url, "test.git", "unexpected dep url")
	assert.Equal(t, dep.Branch, "test_branch", "unexpected dep branch")
	assert.Equal(t, dep.Commit, "test_commit", "unexpected dep commit")
	assert.Equal(t, dep.Tag, "test_tag", "unexpected dep tag")
}

func TestParseRepoNameFromGitUrl(t *testing.T) {
	assert.Equal(t, ParseRepoNameFromGitUrl("test"), "test", "ParseRepoNameFromGitUrl failed.")
	assert.Equal(t, ParseRepoNameFromGitUrl("test.git"), "test", "ParseRepoNameFromGitUrl failed.")
	assert.Equal(t, ParseRepoNameFromGitUrl("test.git.git"), "test.git", "ParseRepoNameFromGitUrl failed.")
	assert.Equal(t, ParseRepoNameFromGitUrl("https://test.git"), "test", "ParseRepoNameFromGitUrl failed.")
	assert.Equal(t, ParseRepoNameFromGitUrl("https://test.git.git"), "test.git", "ParseRepoNameFromGitUrl failed.")
	assert.Equal(t, ParseRepoNameFromGitUrl("httfsdafps://test.git.git"), "test.git", "ParseRepoNameFromGitUrl failed.")
}
