package pkg

import (
	"os"
	"path/filepath"
	"testing"

	modfile "kusionstack.io/kpm/pkg/mod"
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

// Load the kcl package from directory containing kcl.mod and kcl.mod.lock file.
func TestLoadKclPkg(t *testing.T) {
	testDir := init_test_dir()
	kclPkg, err := LoadKclPkg(testDir)
	if err == nil && kclPkg != nil {
		t.Errorf("Failed to 'LoadKclPkg'.")
	}

	_ = modfile.NewModFile(&opt.InitOptions{Name: "test_name", InitPath: testDir}).Store()

	kclPkg, err = LoadKclPkg(testDir)
	if err != nil || kclPkg.modFile.Pkg.Name != "test_name" || kclPkg.modFile.Pkg.Version != "0.0.1" || kclPkg.modFile.Pkg.Edition != "0.0.1" {
		t.Errorf("Failed to 'LoadKclPkg'.")
	}
}

// InitEmptyModule inits an empty kcl module and create a default kcl.modfile.
func TestInitEmptyPkg(t *testing.T) {
	testDir := init_test_dir()
	kclPkg := NewKclPkg(&opt.InitOptions{Name: "test_name", InitPath: testDir})
	err := kclPkg.InitEmptyPkg()
	if err != nil {
		t.Errorf("Failed to 'InitEmptyPkg'.")
	}

	testKclPkg, err := LoadKclPkg(testDir)
	if err != nil || testKclPkg.modFile.Pkg.Name != "test_name" || testKclPkg.modFile.Pkg.Version != "0.0.1" || testKclPkg.modFile.Pkg.Edition != "0.0.1" {
		t.Errorf("Failed to 'LoadKclPkg'.")
	}
}
