package modfile

import (
	"os"
	"path/filepath"
	"testing"

	"kusionstack.io/kpm/pkg/opt"
)

const testDataDir = "test_data_modfile"

func init_test_dir() string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)

	// clean the test data
	_ = os.RemoveAll(testDir)
	_ = os.Mkdir(testDir, 0755)

	return testDir
}

func TestModFileExists(t *testing.T) {
	testDir := init_test_dir()
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

	// generate 'kcl.mod' and 'kcl.mod.lock'.
	err = modFile.StoreLockFile()

	if err != nil {
		t.Errorf("test 'StoreLockFile' failed.")
	}

	is_exist, err = ModFileExists(testDir)
	if err != nil || !is_exist {
		t.Errorf("test 'StoreLockFile' failed.")
	}

	is_exist, err = ModLockFileExists(testDir)
	if err != nil || !is_exist {
		t.Errorf("test 'StoreLockFile' failed.")
	}

	NewModFile, err := LoadModFile(testDir)
	if err != nil || NewModFile.Pkg.Name != "test_kcl_pkg" || NewModFile.Pkg.Version != "0.0.1" || NewModFile.Pkg.Edition != "0.0.1" {
		t.Errorf("test 'LoadModFile' failed.")
	}
}
