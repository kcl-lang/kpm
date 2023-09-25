package client

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
	pkg "kcl-lang.io/kpm/pkg/package"
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

// TestDownloadGit test download from oci registry.
func TestDownloadOci(t *testing.T) {
	testPath := filepath.Join(getTestDir("download"), "k8s_1.27")
	err := os.MkdirAll(testPath, 0755)
	assert.Equal(t, err, nil)

	depFromOci := pkg.Dependency{
		Name:    "k8s",
		Version: "1.27",
		Source: pkg.Source{
			Oci: &pkg.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/k8s",
				Tag:  "1.27",
			},
		},
	}
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	dep, err := kpmcli.Download(&depFromOci, testPath)
	assert.Equal(t, err, nil)
	assert.Equal(t, dep.Name, "k8s")
	assert.Equal(t, dep.FullName, "k8s_1.27")
	assert.Equal(t, dep.Version, "1.27")
	assert.Equal(t, dep.Sum, "xnYM1FWHAy3m+KcQMQb2rjZouTxumqYt6FGZpu2T4yM=")
	assert.NotEqual(t, dep.Source.Oci, nil)
	assert.Equal(t, dep.Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Oci.Repo, "kcl-lang/k8s")
	assert.Equal(t, dep.Source.Oci.Tag, "1.27")
	assert.Equal(t, dep.LocalFullPath, testPath)

	// Check whether the tar downloaded by `kpm add` has been deleted.
	assert.Equal(t, utils.DirExists(filepath.Join(testPath, "k8s_1.27.tar")), false)

	err = os.RemoveAll(getTestDir("download"))
	assert.Equal(t, err, nil)
}

// TestDownloadLatestOci tests the case that the version is empty.
func TestDownloadLatestOci(t *testing.T) {
	testPath := filepath.Join(getTestDir("download"), "a_random_name")
	err := os.MkdirAll(testPath, 0755)
	assert.Equal(t, err, nil)
	depFromOci := pkg.Dependency{
		Name:    "k8s",
		Version: "",
		Source: pkg.Source{
			Oci: &pkg.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/k8s",
				Tag:  "",
			},
		},
	}
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	dep, err := kpmcli.Download(&depFromOci, testPath)
	assert.Equal(t, err, nil)
	assert.Equal(t, dep.Name, "k8s")
	assert.Equal(t, dep.FullName, "k8s_1.27")
	assert.Equal(t, dep.Version, "1.27")
	assert.Equal(t, dep.Sum, "xnYM1FWHAy3m+KcQMQb2rjZouTxumqYt6FGZpu2T4yM=")
	assert.NotEqual(t, dep.Source.Oci, nil)
	assert.Equal(t, dep.Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Oci.Repo, "kcl-lang/k8s")
	assert.Equal(t, dep.Source.Oci.Tag, "1.27")
	assert.Equal(t, dep.LocalFullPath, testPath+"1.27")
	assert.Equal(t, err, nil)

	// Check whether the tar downloaded by `kpm add` has been deleted.
	assert.Equal(t, utils.DirExists(filepath.Join(testPath, "k8s_1.27.tar")), false)

	err = os.RemoveAll(getTestDir("download"))
	assert.Equal(t, err, nil)
}

func TestInitEmptyPkg(t *testing.T) {
	testDir := initTestDir("test_init_empty_mod")
	kclPkg := pkg.NewKclPkg(&opt.InitOptions{Name: "test_name", InitPath: testDir})
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	err = kpmcli.InitEmptyPkg(&kclPkg)
	assert.Equal(t, err, nil)

	testKclPkg, err := pkg.LoadKclPkg(testDir)
	if err != nil {
		t.Errorf("Failed to 'LoadKclPkg'.")
	}

	assert.Equal(t, testKclPkg.ModFile.Pkg.Name, "test_name")
	assert.Equal(t, testKclPkg.ModFile.Pkg.Version, "0.0.1")
	assert.Equal(t, testKclPkg.ModFile.Pkg.Edition, "0.0.1")
}

func TestUpdateKclModAndLock(t *testing.T) {
	testDir := initTestDir("test_data_add_deps")
	// Init an empty package
	kclPkg := pkg.NewKclPkg(&opt.InitOptions{
		Name:     "test_add_deps",
		InitPath: testDir,
	})

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	err = kpmcli.InitEmptyPkg(&kclPkg)
	assert.Equal(t, err, nil)

	dep := pkg.Dependency{
		Name:     "name",
		FullName: "test_version",
		Version:  "test_version",
		Sum:      "test_sum",
		Source: pkg.Source{
			Git: &pkg.Git{
				Url: "test_url",
				Tag: "test_tag",
			},
		},
	}

	oci_dep := pkg.Dependency{
		Name:     "oci_name",
		FullName: "test_version",
		Version:  "test_version",
		Sum:      "test_sum",
		Source: pkg.Source{
			Oci: &pkg.Oci{
				Reg:  "test_reg",
				Repo: "test_repo",
				Tag:  "test_tag",
			},
		},
	}

	kclPkg.Dependencies.Deps["oci_test"] = oci_dep
	kclPkg.ModFile.Dependencies.Deps["oci_test"] = oci_dep

	kclPkg.Dependencies.Deps["test"] = dep
	kclPkg.ModFile.Dependencies.Deps["test"] = dep

	err = kclPkg.ModFile.StoreModFile()

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
		assert.Equal(t, len(kclPkg.ModFile.Deps), 2)
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
		assert.Equal(t, len(kclPkg.ModFile.Deps), 2)
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

func TestVendorDeps(t *testing.T) {
	testDir := getTestDir("resolve_deps")
	kpm_home := filepath.Join(testDir, "kpm_home")
	os.RemoveAll(filepath.Join(testDir, "my_kcl"))
	kcl1Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl1"))
	kcl2Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl2"))

	depKcl1 := pkg.Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Sum:      kcl1Sum,
	}

	depKcl2 := pkg.Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Sum:      kcl2Sum,
	}

	kclPkg := pkg.KclPkg{
		ModFile: pkg.ModFile{
			HomePath: filepath.Join(testDir, "my_kcl"),
			// Whether the current package uses the vendor mode
			// In the vendor mode, kpm will look for the package in the vendor subdirectory
			// in the current package directory.
			VendorMode: false,
			Dependencies: pkg.Dependencies{
				Deps: map[string]pkg.Dependency{
					"kcl1": depKcl1,
					"kcl2": depKcl2,
				},
			},
		},
		HomePath: filepath.Join(testDir, "my_kcl"),
		// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
		// not the dependencies in kcl.mod.
		Dependencies: pkg.Dependencies{
			Deps: map[string]pkg.Dependency{
				"kcl1": depKcl1,
				"kcl2": depKcl2,
			},
		},
	}

	mykclVendorPath := filepath.Join(filepath.Join(testDir, "my_kcl"), "vendor")
	assert.Equal(t, utils.DirExists(mykclVendorPath), false)
	kpmcli, err := NewKpmClient()
	kpmcli.homePath = kpm_home
	assert.Equal(t, err, nil)
	err = kpmcli.VendorDeps(&kclPkg)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(mykclVendorPath), true)
	assert.Equal(t, utils.DirExists(filepath.Join(mykclVendorPath, "kcl1")), true)
	assert.Equal(t, utils.DirExists(filepath.Join(mykclVendorPath, "kcl2")), true)

	maps, err := kpmcli.ResolveDepsIntoMap(&kclPkg)
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

	depKcl1 := pkg.Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Sum:      kcl1Sum,
	}

	depKcl2 := pkg.Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Sum:      kcl2Sum,
	}

	kclPkg := pkg.KclPkg{
		ModFile: pkg.ModFile{
			HomePath: home_path,
			// Whether the current package uses the vendor mode
			// In the vendor mode, kpm will look for the package in the vendor subdirectory
			// in the current package directory.
			VendorMode: true,
			Dependencies: pkg.Dependencies{
				Deps: map[string]pkg.Dependency{
					"kcl1": depKcl1,
					"kcl2": depKcl2,
				},
			},
		},
		HomePath: home_path,
		// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
		// not the dependencies in kcl.mod.
		Dependencies: pkg.Dependencies{
			Deps: map[string]pkg.Dependency{
				"kcl1": depKcl1,
				"kcl2": depKcl2,
			},
		},
	}
	mySearchPath := filepath.Join(home_path, "vendor")
	assert.Equal(t, utils.DirExists(mySearchPath), false)

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kpmcli.homePath = kpm_home

	maps, err := kpmcli.ResolveDepsIntoMap(&kclPkg)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(maps), 2)
	checkDepsMapInSearchPath(t, depKcl1, mySearchPath, maps)

	kclPkg.SetVendorMode(false)
	maps, err = kpmcli.ResolveDepsIntoMap(&kclPkg)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(maps), 2)
	checkDepsMapInSearchPath(t, depKcl1, kpm_home, maps)

	os.RemoveAll(home_path)
}

func TestCompileWithEntryFile(t *testing.T) {
	testDir := getTestDir("resolve_deps")
	kpm_home := filepath.Join(testDir, "kpm_home")
	home_path := filepath.Join(testDir, "my_kcl_compile")
	vendor_path := filepath.Join(home_path, "vendor")
	entry_file := filepath.Join(home_path, "main.k")
	os.RemoveAll(vendor_path)

	kcl1Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl1"))
	depKcl1 := pkg.Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Sum:      kcl1Sum,
	}
	kcl2Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl2"))
	depKcl2 := pkg.Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Sum:      kcl2Sum,
	}

	kclPkg := pkg.KclPkg{
		ModFile: pkg.ModFile{
			HomePath: home_path,
			// Whether the current package uses the vendor mode
			// In the vendor mode, kpm will look for the package in the vendor subdirectory
			// in the current package directory.
			VendorMode: true,
			Dependencies: pkg.Dependencies{
				Deps: map[string]pkg.Dependency{
					"kcl1": depKcl1,
					"kcl2": depKcl2,
				},
			},
		},
		HomePath: home_path,
		// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
		// not the dependencies in kcl.mod.
		Dependencies: pkg.Dependencies{
			Deps: map[string]pkg.Dependency{
				"kcl1": depKcl1,
				"kcl2": depKcl2,
			},
		},
	}

	assert.Equal(t, utils.DirExists(vendor_path), false)

	compiler := runner.DefaultCompiler()
	compiler.AddKFile(entry_file)
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kpmcli.homePath = kpm_home
	result, err := kpmcli.Compile(&kclPkg, compiler)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(vendor_path, "kcl1")), true)
	assert.Equal(t, utils.DirExists(filepath.Join(vendor_path, "kcl2")), true)
	assert.Equal(t, result.GetRawYamlResult(), "c1: 1\nc2: 2")
	os.RemoveAll(vendor_path)

	kclPkg.SetVendorMode(false)
	assert.Equal(t, utils.DirExists(vendor_path), false)

	result, err = kpmcli.Compile(&kclPkg, compiler)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(vendor_path), false)
	assert.Equal(t, result.GetRawYamlResult(), "c1: 1\nc2: 2")
	os.RemoveAll(vendor_path)
}

func checkDepsMapInSearchPath(t *testing.T, dep pkg.Dependency, searchPath string, maps map[string]string) {
	assert.Equal(t, maps[dep.Name], filepath.Join(searchPath, dep.FullName))
	assert.Equal(t, utils.DirExists(filepath.Join(searchPath, dep.FullName)), true)
}

func TestPackageCurrentPkgPath(t *testing.T) {
	testDir := getTestDir("tar_kcl_pkg")

	kclPkg, err := pkg.LoadKclPkg(testDir)
	assert.Equal(t, err, nil)
	assert.Equal(t, kclPkg.GetPkgTag(), "0.0.1")
	assert.Equal(t, kclPkg.GetPkgName(), "test_tar")
	assert.Equal(t, kclPkg.GetPkgFullName(), "test_tar-0.0.1")
	assert.Equal(t, kclPkg.GetPkgTarName(), "test_tar-0.0.1.tar")

	assert.Equal(t, utils.DirExists(filepath.Join(testDir, kclPkg.GetPkgTarName())), false)

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	path, err := kpmcli.PackagePkg(kclPkg, true)
	assert.Equal(t, err, nil)
	assert.Equal(t, path, filepath.Join(testDir, kclPkg.GetPkgTarName()))
	assert.Equal(t, utils.DirExists(filepath.Join(testDir, kclPkg.GetPkgTarName())), true)
	defer func() {
		if r := os.RemoveAll(filepath.Join(testDir, kclPkg.GetPkgTarName())); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
}

func TestResolveMetadataInJsonStr(t *testing.T) {
	originalValue := os.Getenv(env.PKG_PATH)
	defer os.Setenv(env.PKG_PATH, originalValue)

	testDir := getTestDir("resolve_metadata")

	testHomePath := filepath.Join(filepath.Dir(testDir), "test_home_path")
	prepareKpmHomeInPath(testHomePath)
	defer os.RemoveAll(testHomePath)

	os.Setenv(env.PKG_PATH, testHomePath)

	kclpkg, err := pkg.LoadKclPkg(testDir)
	assert.Equal(t, err, nil)

	globalPkgPath, _ := env.GetAbsPkgPath()
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	res, err := kpmcli.ResolveDepsMetadataInJsonStr(kclpkg, true)
	assert.Equal(t, err, nil)

	expectedDep := pkg.Dependencies{
		Deps: make(map[string]pkg.Dependency),
	}

	expectedDep.Deps["konfig"] = pkg.Dependency{
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
	kclpkg.SetVendorMode(true)
	res, err = kpmcli.ResolveDepsMetadataInJsonStr(kclpkg, true)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(vendorDir), true)
	assert.Equal(t, utils.DirExists(filepath.Join(vendorDir, "konfig_v0.0.1")), true)

	expectedDep.Deps["konfig"] = pkg.Dependency{
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

	kclpkg, err = pkg.LoadKclPkg(testDir)
	assert.Equal(t, err, nil)
	kpmcli.homePath = "not_exist"
	res, err = kpmcli.ResolveDepsMetadataInJsonStr(kclpkg, true)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(vendorDir), false)
	assert.Equal(t, utils.DirExists(filepath.Join(vendorDir, "konfig_v0.0.1")), false)
	jsonPath, err := json.Marshal(filepath.Join("not_exist", "konfig_v0.0.1"))
	assert.Equal(t, err, nil)
	expectedStr := fmt.Sprintf("{\"packages\":{\"konfig\":{\"name\":\"konfig\",\"manifest_path\":\"%s\"}}}", string(jsonPath))
	assert.Equal(t, res, expectedStr)
	defer func() {
		if r := os.RemoveAll(filepath.Join("not_exist", "konfig_v0.0.1")); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
}

func prepareKpmHomeInPath(path string) {
	dirPath := filepath.Join(filepath.Join(path, ".kpm"), "config")
	_ = os.MkdirAll(dirPath, 0755)

	filePath := filepath.Join(dirPath, "kpm.json")

	_ = os.WriteFile(filePath, []byte("{\"DefaultOciRegistry\":\"ghcr.io\",\"DefaultOciRepo\":\"awesome-kusion\"}"), 0644)
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
	kclPkg1 := pkg.NewKclPkg(&initOpts)
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	_, err = kpmcli.AddDepWithOpts(&kclPkg1, &opt.AddOptions{
		LocalPath: "localPath",
		RegistryOpts: opt.RegistryOptions{
			Local: &opt.LocalOptions{
				Path: filepath.Join(testDir, "kcl2"),
			},
		},
	})

	assert.Equal(t, err, nil)

	// package the kcl1 into tar in vendor mode.
	tarPath, err := kpmcli.PackagePkg(&kclPkg1, true)
	assert.Equal(t, err, nil)
	hasSubDir, err := hasSubdirInTar(tarPath, "vendor")
	assert.Equal(t, err, nil)
	assert.Equal(t, hasSubDir, true)

	// clean the kcl1
	err = os.RemoveAll(kcl1Path)
	assert.Equal(t, err, nil)

	createKclPkg1()
	// package the kcl1 into tar in non-vendor mode.
	tarPath, err = kpmcli.PackagePkg(&kclPkg1, false)
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

func TestNewKpmClient(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kpmhome, err := env.GetAbsPkgPath()
	assert.Equal(t, err, nil)
	assert.Equal(t, kpmcli.homePath, kpmhome)
	assert.Equal(t, kpmcli.GetSettings().KpmConfFile, filepath.Join(kpmhome, ".kpm", "config", "kpm.json"))
	assert.Equal(t, kpmcli.GetSettings().CredentialsFile, filepath.Join(kpmhome, ".kpm", "config", "config.json"))
	assert.Equal(t, kpmcli.GetSettings().Conf.DefaultOciRepo, "kcl-lang")
	assert.Equal(t, kpmcli.GetSettings().Conf.DefaultOciRegistry, "ghcr.io")
	assert.Equal(t, kpmcli.GetSettings().Conf.DefaultOciPlainHttp, false)
}

func TestParseOciOptionFromString(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	oci_ref_with_tag := "test_oci_repo:test_oci_tag"
	ociOption, err := kpmcli.ParseOciOptionFromString(oci_ref_with_tag, "test_tag")
	assert.Equal(t, err, nil)
	assert.Equal(t, ociOption.PkgName, "")
	assert.Equal(t, ociOption.Reg, "ghcr.io")
	assert.Equal(t, ociOption.Repo, "kcl-lang/test_oci_repo")
	assert.Equal(t, ociOption.Tag, "test_oci_tag")

	oci_ref_without_tag := "test_oci_repo:test_oci_tag"
	ociOption, err = kpmcli.ParseOciOptionFromString(oci_ref_without_tag, "test_tag")
	assert.Equal(t, err, nil)
	assert.Equal(t, ociOption.PkgName, "")
	assert.Equal(t, ociOption.Reg, "ghcr.io")
	assert.Equal(t, ociOption.Repo, "kcl-lang/test_oci_repo")
	assert.Equal(t, ociOption.Tag, "test_oci_tag")

	oci_url_with_tag := "oci://test_reg/test_oci_repo"
	ociOption, err = kpmcli.ParseOciOptionFromString(oci_url_with_tag, "test_tag")
	assert.Equal(t, err, nil)
	assert.Equal(t, ociOption.PkgName, "")
	assert.Equal(t, ociOption.Reg, "test_reg")
	assert.Equal(t, ociOption.Repo, "/test_oci_repo")
	assert.Equal(t, ociOption.Tag, "test_tag")
}
