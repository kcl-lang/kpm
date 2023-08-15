package pkg

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/runner"
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

func TestUpdateKclModAndLock(t *testing.T) {
	testDir := initTestDir("test_data_add_deps")
	// Init an empty package
	kclPkg := NewKclPkg(&opt.InitOptions{
		Name:     "test_add_deps",
		InitPath: testDir,
	})

	_ = kclPkg.InitEmptyPkg()

	dep := Dependency{
		Name:     "name",
		FullName: "test_version",
		Version:  "test_version",
		Sum:      "test_sum",
		Source: Source{
			Git: &Git{
				Url: "test_url",
				Tag: "test_tag",
			},
		},
	}

	oci_dep := Dependency{
		Name:     "oci_name",
		FullName: "test_version",
		Version:  "test_version",
		Sum:      "test_sum",
		Source: Source{
			Oci: &Oci{
				Reg:  "test_reg",
				Repo: "test_repo",
				Tag:  "test_tag",
			},
		},
	}

	kclPkg.Dependencies.Deps["oci_test"] = oci_dep
	kclPkg.modFile.Dependencies.Deps["oci_test"] = oci_dep

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

	if gotKclMod, err := os.ReadFile(filepath.Join(testDir, "kcl.mod")); os.IsNotExist(err) {
		t.Errorf("failed to find kcl.mod.")
	} else {
		assert.Equal(t, len(kclPkg.Dependencies.Deps), 2)
		assert.Equal(t, len(kclPkg.modFile.Deps), 2)
		expectKclMod, _ := os.ReadFile(filepath.Join(expectDir, "kcl.mod"))
		expectKclModReverse, _ := os.ReadFile(filepath.Join(expectDir, "kcl.reverse.mod"))

		gotKclModStr := utils.RmNewline(string(gotKclMod))
		fmt.Printf("gotKclModStr: '%v'\n", gotKclModStr)
		expectKclModStr := utils.RmNewline(string(expectKclMod))
		fmt.Printf("expectKclModStr: '%v'\n", expectKclModStr)
		expectKclModReverseStr := utils.RmNewline(string(expectKclModReverse))
		fmt.Printf("expectKclModReverseStr: '%v'\n", expectKclModReverseStr)

		assert.Equal(t,
			(gotKclModStr == expectKclModStr || gotKclModStr == expectKclModReverseStr),
			true,
		)
	}

	if gotKclModLock, err := os.ReadFile(filepath.Join(testDir, "kcl.mod.lock")); os.IsNotExist(err) {
		t.Errorf("failed to find kcl.mod.lock.")
	} else {
		assert.Equal(t, len(kclPkg.Dependencies.Deps), 2)
		assert.Equal(t, len(kclPkg.modFile.Deps), 2)
		expectKclModLock, _ := os.ReadFile(filepath.Join(expectDir, "kcl.mod.lock"))
		expectKclModLockReverse, _ := os.ReadFile(filepath.Join(expectDir, "kcl.mod.reverse.lock"))

		gotKclModLockStr := utils.RmNewline(string(gotKclModLock))
		fmt.Printf("gotKclModLockStr: '%v'\n", gotKclModLockStr)
		expectKclModLockStr := utils.RmNewline(string(expectKclModLock))
		fmt.Printf("expectKclModLockStr: '%v'\n", expectKclModLockStr)
		expectKclModLockReverseStr := utils.RmNewline(string(expectKclModLockReverse))
		fmt.Printf("expectKclModLockReverseStr: '%v'\n", expectKclModLockReverseStr)

		assert.Equal(t,
			(gotKclModLockStr == expectKclModLockStr) || (gotKclModLockStr == expectKclModLockReverseStr),
			true,
		)
	}
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
		modFile: ModFile{
			Pkg: Package{
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

	depKcl1 := Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Sum:      kcl1Sum,
	}

	depKcl2 := Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Sum:      kcl2Sum,
	}

	kclPkg := KclPkg{
		modFile: ModFile{
			HomePath: filepath.Join(testDir, "my_kcl"),
			// Whether the current package uses the vendor mode
			// In the vendor mode, kpm will look for the package in the vendor subdirectory
			// in the current package directory.
			VendorMode: false,
			Dependencies: Dependencies{
				Deps: map[string]Dependency{
					"kcl1": depKcl1,
					"kcl2": depKcl2,
				},
			},
		},
		HomePath: filepath.Join(testDir, "my_kcl"),
		// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
		// not the dependencies in kcl.mod.
		Dependencies: Dependencies{
			Deps: map[string]Dependency{
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

	depKcl1 := Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Sum:      kcl1Sum,
	}

	depKcl2 := Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Sum:      kcl2Sum,
	}

	kclPkg := KclPkg{
		modFile: ModFile{
			HomePath: home_path,
			// Whether the current package uses the vendor mode
			// In the vendor mode, kpm will look for the package in the vendor subdirectory
			// in the current package directory.
			VendorMode: true,
			Dependencies: Dependencies{
				Deps: map[string]Dependency{
					"kcl1": depKcl1,
					"kcl2": depKcl2,
				},
			},
		},
		HomePath: home_path,
		// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
		// not the dependencies in kcl.mod.
		Dependencies: Dependencies{
			Deps: map[string]Dependency{
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

func checkDepsMapInSearchPath(t *testing.T, dep Dependency, searchPath string, maps map[string]string) {
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
	depKcl1 := Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Sum:      kcl1Sum,
	}
	kcl2Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl2"))
	depKcl2 := Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Sum:      kcl2Sum,
	}

	kclPkg := KclPkg{
		modFile: ModFile{
			HomePath: home_path,
			// Whether the current package uses the vendor mode
			// In the vendor mode, kpm will look for the package in the vendor subdirectory
			// in the current package directory.
			VendorMode: true,
			Dependencies: Dependencies{
				Deps: map[string]Dependency{
					"kcl1": depKcl1,
					"kcl2": depKcl2,
				},
			},
		},
		HomePath: home_path,
		// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
		// not the dependencies in kcl.mod.
		Dependencies: Dependencies{
			Deps: map[string]Dependency{
				"kcl1": depKcl1,
				"kcl2": depKcl2,
			},
		},
	}

	assert.Equal(t, utils.DirExists(vendor_path), false)

	compiler := runner.DefaultCompiler()
	compiler.AddKFile(entry_file)
	result, err := kclPkg.Compile(kpm_home, compiler)
	assert.Equal(t, utils.DirExists(filepath.Join(vendor_path, "kcl1")), true)
	assert.Equal(t, utils.DirExists(filepath.Join(vendor_path, "kcl2")), true)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.GetRawYamlResult(), "c1: 1\nc2: 2")
	os.RemoveAll(vendor_path)

	kclPkg.SetVendorMode(false)
	assert.Equal(t, utils.DirExists(vendor_path), false)

	result, err = kclPkg.Compile(kpm_home, compiler)
	assert.Equal(t, utils.DirExists(vendor_path), false)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.GetRawYamlResult(), "c1: 1\nc2: 2")
	os.RemoveAll(vendor_path)
}

func TestValidateKpmHome(t *testing.T) {
	kclPkg := NewKclPkg(&opt.InitOptions{
		Name:     "test_name",
		InitPath: "test_home_path",
	})
	oldValue := os.Getenv(env.PKG_PATH)
	os.Setenv(env.PKG_PATH, "test_home_path")
	err := kclPkg.ValidateKpmHome(os.Getenv(env.PKG_PATH))
	assert.Equal(t, err.Error(), "kpm: environment variable KCL_PKG_PATH cannot be set to the same path as the current KCL package.\n")
	assert.Equal(t, err.Type(), reporter.InvalidKpmHomeInCurrentPkg)
	os.Setenv(env.PKG_PATH, oldValue)
}

func TestPackageCurrentPkgPath(t *testing.T) {
	testDir := getTestDir("tar_kcl_pkg")

	kclPkg, err := LoadKclPkg(testDir)
	assert.Equal(t, err, nil)
	assert.Equal(t, kclPkg.GetPkgTag(), "0.0.1")
	assert.Equal(t, kclPkg.GetPkgName(), "test_tar")
	assert.Equal(t, kclPkg.GetPkgFullName(), "test_tar-0.0.1")
	assert.Equal(t, kclPkg.GetPkgTarName(), "test_tar-0.0.1.tar")

	assert.Equal(t, utils.DirExists(filepath.Join(testDir, kclPkg.GetPkgTarName())), false)

	path, err := kclPkg.PackageCurrentPkgPath(true)
	assert.Equal(t, err, nil)
	assert.Equal(t, path, filepath.Join(testDir, kclPkg.GetPkgTarName()))
	assert.Equal(t, utils.DirExists(filepath.Join(testDir, kclPkg.GetPkgTarName())), true)
	err = os.RemoveAll(filepath.Join(testDir, kclPkg.GetPkgTarName()))
	assert.Equal(t, err, nil)
}

func TestLoadKclPkgFromTar(t *testing.T) {
	testDir := getTestDir("load_kcl_tar")
	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "kcl1-v0.0.3")), false)

	kclPkg, err := LoadKclPkgFromTar(filepath.Join(testDir, "kcl1-v0.0.3.tar"))
	assert.Equal(t, err, nil)
	assert.Equal(t, kclPkg.HomePath, filepath.Join(testDir, "kcl1-v0.0.3"))
	assert.Equal(t, kclPkg.modFile.Pkg.Name, "kcl1")
	assert.Equal(t, kclPkg.modFile.Pkg.Edition, "0.0.1")
	assert.Equal(t, kclPkg.modFile.Pkg.Version, "0.0.3")

	assert.Equal(t, len(kclPkg.modFile.Deps), 2)
	assert.Equal(t, kclPkg.modFile.Deps["konfig"].Name, "konfig")
	assert.Equal(t, kclPkg.modFile.Deps["konfig"].FullName, "konfig_v0.0.1")
	assert.Equal(t, kclPkg.modFile.Deps["konfig"].Git.Url, "https://github.com/awesome-kusion/konfig.git")
	assert.Equal(t, kclPkg.modFile.Deps["konfig"].Git.Tag, "v0.0.1")

	assert.Equal(t, kclPkg.modFile.Deps["oci_konfig"].Name, "oci_konfig")
	assert.Equal(t, kclPkg.modFile.Deps["oci_konfig"].FullName, "oci_konfig_0.0.1")
	assert.Equal(t, kclPkg.modFile.Deps["oci_konfig"].Oci.Tag, "0.0.1")

	assert.Equal(t, len(kclPkg.Deps), 2)
	assert.Equal(t, kclPkg.Deps["konfig"].Name, "konfig")
	assert.Equal(t, kclPkg.Deps["konfig"].FullName, "konfig_v0.0.1")
	assert.Equal(t, kclPkg.Deps["konfig"].Git.Url, "https://github.com/awesome-kusion/konfig.git")
	assert.Equal(t, kclPkg.Deps["konfig"].Git.Tag, "v0.0.1")
	assert.Equal(t, kclPkg.Deps["konfig"].Sum, "XFvHdBAoY/+qpJWmj8cjwOwZO8a3nX/7SE35cTxQOFU=")

	assert.Equal(t, kclPkg.Deps["oci_konfig"].Name, "oci_konfig")
	assert.Equal(t, kclPkg.Deps["oci_konfig"].FullName, "oci_konfig_0.0.1")
	assert.Equal(t, kclPkg.Deps["oci_konfig"].Oci.Reg, "ghcr.io")
	assert.Equal(t, kclPkg.Deps["oci_konfig"].Oci.Repo, "awesome-kusion/oci_konfig")
	assert.Equal(t, kclPkg.Deps["oci_konfig"].Oci.Tag, "0.0.1")
	assert.Equal(t, kclPkg.Deps["oci_konfig"].Sum, "sLr3e6W4RPrXYyswdOSiKqkHes1QHX2tk6SwxAPDqqo=")

	assert.Equal(t, kclPkg.GetPkgTag(), "0.0.3")
	assert.Equal(t, kclPkg.GetPkgName(), "kcl1")
	assert.Equal(t, kclPkg.GetPkgFullName(), "kcl1-0.0.3")
	assert.Equal(t, kclPkg.GetPkgTarName(), "kcl1-0.0.3.tar")

	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "kcl1-v0.0.3")), true)
	err = os.RemoveAll(filepath.Join(testDir, "kcl1-v0.0.3"))
	assert.Equal(t, err, nil)
}

func prepareKpmHomeInPath(path string) {
	dirPath := filepath.Join(filepath.Join(path, ".kpm"), "config")
	_ = os.MkdirAll(dirPath, 0755)

	filePath := filepath.Join(dirPath, "kpm.json")

	_ = os.WriteFile(filePath, []byte("{\"DefaultOciRegistry\":\"ghcr.io\",\"DefaultOciRepo\":\"awesome-kusion\"}"), 0644)
}

func TestResolveMetadataInJsonStr(t *testing.T) {
	originalValue := os.Getenv(env.PKG_PATH)
	defer os.Setenv(env.PKG_PATH, originalValue)

	testDir := getTestDir("resolve_metadata")

	testHomePath := filepath.Join(filepath.Dir(testDir), "test_home_path")
	prepareKpmHomeInPath(testHomePath)
	defer os.RemoveAll(testHomePath)

	os.Setenv(env.PKG_PATH, testHomePath)

	pkg, err := LoadKclPkg(testDir)
	assert.Equal(t, err, nil)

	globalPkgPath, _ := env.GetAbsPkgPath()
	res, err := pkg.ResolveDepsMetadataInJsonStr(globalPkgPath, true)
	assert.Equal(t, err, nil)

	expectedDep := Dependencies{
		Deps: make(map[string]Dependency),
	}

	expectedDep.Deps["konfig"] = Dependency{
		Name:          "konfig",
		FullName:      "konfig_v0.0.1",
		LocalFullPath: filepath.Join(globalPkgPath, "konfig_v0.0.1"),
	}

	expectedDepStr, err := json.Marshal(expectedDep)
	assert.Equal(t, err, nil)

	assert.Equal(t, res, string(expectedDepStr))

	vendorDir := filepath.Join(testDir, "vendor")
	if utils.DirExists(vendorDir) {
		err = os.RemoveAll(vendorDir)
		assert.Equal(t, err, nil)
	}
	pkg.SetVendorMode(true)
	res, err = pkg.ResolveDepsMetadataInJsonStr(globalPkgPath, true)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(vendorDir), true)
	assert.Equal(t, utils.DirExists(filepath.Join(vendorDir, "konfig_v0.0.1")), true)

	expectedDep.Deps["konfig"] = Dependency{
		Name:          "konfig",
		FullName:      "konfig_v0.0.1",
		LocalFullPath: filepath.Join(vendorDir, "konfig_v0.0.1"),
	}

	expectedDepStr, err = json.Marshal(expectedDep)
	assert.Equal(t, err, nil)

	assert.Equal(t, res, string(expectedDepStr))
	if utils.DirExists(vendorDir) {
		err = os.RemoveAll(vendorDir)
		assert.Equal(t, err, nil)
	}

	pkg, err = LoadKclPkg(testDir)
	assert.Equal(t, err, nil)
	res, err = pkg.ResolveDepsMetadataInJsonStr("not_exist", false)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(vendorDir), false)
	assert.Equal(t, utils.DirExists(filepath.Join(vendorDir, "konfig_v0.0.1")), false)
	expectedStr := "{\"packages\":{\"konfig\":{\"name\":\"konfig\",\"manifest_path\":\"\"}}}"
	assert.Equal(t, res, expectedStr)
}

func TestPkgWithInVendorMode(t *testing.T) {
	testDir := getTestDir("test_pkg_with_vendor")
	kcl1Path := filepath.Join(testDir, "kcl1")

	createKclPkg1 := func() {
		assert.Equal(t, utils.DirExists(kcl1Path), false)
		err := os.MkdirAll(kcl1Path, 0755)
		assert.Equal(t, err, nil)
	}

	defer func() {
		if err := os.RemoveAll(kcl1Path); err != nil {
			log.Printf("failed to close file: %v", err)
		}
	}()

	createKclPkg1()

	initOpts := opt.InitOptions{
		Name:     "kcl1",
		InitPath: kcl1Path,
	}
	kclPkg1 := NewKclPkg(&initOpts)

	kclPkg1.AddDeps(&opt.AddOptions{
		LocalPath: "localPath",
		RegistryOpts: opt.RegistryOptions{
			Local: &opt.LocalOptions{
				Path: filepath.Join(testDir, "kcl2"),
			},
		},
	})

	// package the kcl1 into tar in vendor mode.
	tarPath, err := kclPkg1.PackageCurrentPkgPath(true)
	assert.Equal(t, err, nil)
	hasSubDir, err := hasSubdirInTar(tarPath, "vendor")
	assert.Equal(t, err, nil)
	assert.Equal(t, hasSubDir, true)

	// clean the kcl1
	err = os.RemoveAll(kcl1Path)
	assert.Equal(t, err, nil)

	createKclPkg1()
	// package the kcl1 into tar in non-vendor mode.
	tarPath, err = kclPkg1.PackageCurrentPkgPath(false)
	assert.Equal(t, err, nil)
	hasSubDir, err = hasSubdirInTar(tarPath, "vendor")
	assert.Equal(t, err, nil)
	assert.Equal(t, hasSubDir, false)
}

// check if the tar file contains the subdir
func hasSubdirInTar(tarPath, subdir string) (bool, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if hdr.Typeflag == tar.TypeDir && filepath.Base(hdr.Name) == subdir {
			return true, nil
		}
	}

	return false, nil
}
