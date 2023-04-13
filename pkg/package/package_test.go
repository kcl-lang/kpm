package pkg

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kusionstack.io/kpm/pkg/errors"
	modfile "kusionstack.io/kpm/pkg/mod"
	"kusionstack.io/kpm/pkg/opt"
	"kusionstack.io/kpm/pkg/runner"
	"kusionstack.io/kpm/pkg/utils"
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

func TestGetPkgName(t *testing.T) {
	kclPkg := KclPkg{
		modFile: modfile.ModFile{
			Pkg: modfile.Package{
				Name: "test",
			},
		},
	}
	assert.Equal(t, kclPkg.GetPkgName(), "test")
}

func TestVendorDeps(t *testing.T) {
	testDir := getTestDir("resolve_deps")
	kpm_home := filepath.Join(testDir, "kpm_home")
	os.RemoveAll(filepath.Join(testDir, "my_kcl"))
	kcl1Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl1"))
	kcl2Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl2"))

	depKcl1 := modfile.Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Sum:      kcl1Sum,
	}

	depKcl2 := modfile.Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Sum:      kcl2Sum,
	}

	kclPkg := KclPkg{
		modFile: modfile.ModFile{
			HomePath: filepath.Join(testDir, "my_kcl"),
			// Whether the current package uses the vendor mode
			// In the vendor mode, kpm will look for the package in the vendor subdirectory
			// in the current package directory.
			VendorMode: false,
			Dependencies: modfile.Dependencies{
				Deps: map[string]modfile.Dependency{
					"kcl1": depKcl1,
					"kcl2": depKcl2,
				},
			},
		},
		HomePath: filepath.Join(testDir, "my_kcl"),
		// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
		// not the dependencies in kcl.mod.
		Dependencies: modfile.Dependencies{
			Deps: map[string]modfile.Dependency{
				"kcl1": depKcl1,
				"kcl2": depKcl2,
			},
		},
	}

	mykclVendorPath := filepath.Join(filepath.Join(testDir, "my_kcl"), "vendor")
	assert.Equal(t, utils.DirExists(mykclVendorPath), false)
	err := kclPkg.VendorDeps(kpm_home)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(mykclVendorPath), true)
	assert.Equal(t, utils.DirExists(filepath.Join(mykclVendorPath, "kcl1")), true)
	assert.Equal(t, utils.DirExists(filepath.Join(mykclVendorPath, "kcl2")), true)

	maps, err := kclPkg.ResolveDeps(kpm_home)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(maps), 2)

	os.RemoveAll(filepath.Join(testDir, "my_kcl"))
}

func TestResolveDepsVendorMode(t *testing.T) {
	testDir := getTestDir("resolve_deps")
	kpm_home := filepath.Join(testDir, "kpm_home")
	home_path := filepath.Join(testDir, "my_kcl_resolve_deps_vendor_mode")
	os.RemoveAll(home_path)
	kcl1Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl1"))
	kcl2Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl2"))

	depKcl1 := modfile.Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Sum:      kcl1Sum,
	}

	depKcl2 := modfile.Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Sum:      kcl2Sum,
	}

	kclPkg := KclPkg{
		modFile: modfile.ModFile{
			HomePath: home_path,
			// Whether the current package uses the vendor mode
			// In the vendor mode, kpm will look for the package in the vendor subdirectory
			// in the current package directory.
			VendorMode: true,
			Dependencies: modfile.Dependencies{
				Deps: map[string]modfile.Dependency{
					"kcl1": depKcl1,
					"kcl2": depKcl2,
				},
			},
		},
		HomePath: home_path,
		// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
		// not the dependencies in kcl.mod.
		Dependencies: modfile.Dependencies{
			Deps: map[string]modfile.Dependency{
				"kcl1": depKcl1,
				"kcl2": depKcl2,
			},
		},
	}
	mySearchPath := filepath.Join(home_path, "vendor")
	assert.Equal(t, utils.DirExists(mySearchPath), false)

	maps, err := kclPkg.ResolveDeps(kpm_home)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(maps), 2)
	checkDepsMapInSearchPath(t, depKcl1, mySearchPath, maps)

	kclPkg.SetVendorMode(false)
	maps, err = kclPkg.ResolveDeps(kpm_home)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(maps), 2)
	checkDepsMapInSearchPath(t, depKcl1, kpm_home, maps)

	os.RemoveAll(home_path)
}

func checkDepsMapInSearchPath(t *testing.T, dep modfile.Dependency, searchPath string, maps map[string]string) {
	assert.Equal(t, maps[dep.Name], filepath.Join(searchPath, dep.FullName))
	assert.Equal(t, utils.DirExists(filepath.Join(searchPath, dep.FullName)), true)
}

func TestCompileWithEntryFile(t *testing.T) {
	testDir := getTestDir("resolve_deps")
	kpm_home := filepath.Join(testDir, "kpm_home")
	home_path := filepath.Join(testDir, "my_kcl_compile")
	vendor_path := filepath.Join(home_path, "vendor")
	entry_file := filepath.Join(home_path, "main.k")
	os.RemoveAll(vendor_path)

	kcl1Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl1"))
	depKcl1 := modfile.Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Sum:      kcl1Sum,
	}
	kcl2Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl2"))
	depKcl2 := modfile.Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Sum:      kcl2Sum,
	}

	kclPkg := KclPkg{
		modFile: modfile.ModFile{
			HomePath: home_path,
			// Whether the current package uses the vendor mode
			// In the vendor mode, kpm will look for the package in the vendor subdirectory
			// in the current package directory.
			VendorMode: true,
			Dependencies: modfile.Dependencies{
				Deps: map[string]modfile.Dependency{
					"kcl1": depKcl1,
					"kcl2": depKcl2,
				},
			},
		},
		HomePath: home_path,
		// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
		// not the dependencies in kcl.mod.
		Dependencies: modfile.Dependencies{
			Deps: map[string]modfile.Dependency{
				"kcl1": depKcl1,
				"kcl2": depKcl2,
			},
		},
	}

	assert.Equal(t, utils.DirExists(vendor_path), false)

	compileOpts := opt.NewKclvmOpts()
	compileOpts.EntryFiles = append(compileOpts.EntryFiles, entry_file)
	cmd, err := runner.NewCompileCmd(compileOpts)
	assert.Equal(t, err, nil)
	result, err := kclPkg.CompileWithEntryFile(kpm_home, cmd)
	assert.Equal(t, utils.DirExists(filepath.Join(vendor_path, "kcl1")), true)
	assert.Equal(t, utils.DirExists(filepath.Join(vendor_path, "kcl2")), true)
	assert.Equal(t, err, nil)
	assert.Equal(t, result, "c1: 1\nc2: 2\n")
	os.RemoveAll(vendor_path)

	kclPkg.SetVendorMode(false)
	assert.Equal(t, utils.DirExists(vendor_path), false)
	cmd, err = runner.NewCompileCmd(compileOpts)
	assert.Equal(t, err, nil)
	result, err = kclPkg.CompileWithEntryFile(kpm_home, cmd)
	assert.Equal(t, utils.DirExists(vendor_path), false)
	assert.Equal(t, err, nil)
	assert.Equal(t, result, "c1: 1\nc2: 2\n")
	os.RemoveAll(vendor_path)
}

func TestValidateKpmHome(t *testing.T) {
	kclPkg := NewKclPkg(&opt.InitOptions{
		Name:     "test_name",
		InitPath: "test_home_path",
	})

	os.Setenv("KPM_HOME", "test_home_path")
	err := kclPkg.ValidateKpmHome(os.Getenv("KPM_HOME"))
	assert.Equal(t, err, errors.InvalidKpmHomeInCurrentPkg)
}
