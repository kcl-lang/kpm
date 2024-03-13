package client

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dominikbraun/graph"
	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/git"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
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

// TestDownloadGit test download from oci registry.
func TestDownloadOci(t *testing.T) {
	testPath := filepath.Join(getTestDir("download"), "k8s_1.27")
	err := os.MkdirAll(testPath, 0755)
	assert.Equal(t, err, nil)

	defer func() {
		err := os.RemoveAll(getTestDir("download"))
		if err != nil {
			t.Errorf("Failed to remove directory: %v", err)
		}
	}()

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
	downloadPath := getTestDir("download")
	assert.Equal(t, utils.DirExists(filepath.Join(downloadPath, "k8s_1.27.tar")), false)

	assert.Equal(t, utils.DirExists(filepath.Join(downloadPath, "k8s_1.27")), true)
	assert.Equal(t, utils.DirExists(filepath.Join(downloadPath, "k8s")), true)

	// Check whether the reference and the dependency have the same hash.
	hashDep, err := utils.HashDir(filepath.Join(downloadPath, "k8s_1.27"))
	assert.Equal(t, err, nil)

	depRefPath := filepath.Join(downloadPath, "k8s")
	info, err := os.Lstat(depRefPath)
	assert.Equal(t, err, nil)

	if info.Mode()&os.ModeSymlink != 0 {
		depRefPath, err = os.Readlink(depRefPath)
		assert.Equal(t, err, nil)
	}

	hashRef, err := utils.HashDir(depRefPath)
	assert.Equal(t, err, nil)
	assert.Equal(t, hashDep, hashRef)
}

// TestDownloadLatestOci tests the case that the version is empty.
func TestDownloadLatestOci(t *testing.T) {
	testPath := filepath.Join(getTestDir("download"), "a_random_name")
	defer func() {
		err := os.RemoveAll(getTestDir("download"))
		if err != nil {
			t.Errorf("Failed to remove directory: %v", err)
		}
	}()
	err := os.MkdirAll(testPath, 0755)
	assert.Equal(t, err, nil)
	depFromOci := pkg.Dependency{
		Name:    "helloworld",
		Version: "",
		Source: pkg.Source{
			Oci: &pkg.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Tag:  "",
			},
		},
	}
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	dep, err := kpmcli.Download(&depFromOci, testPath)
	assert.Equal(t, err, nil)
	assert.Equal(t, dep.Name, "helloworld")
	assert.Equal(t, dep.FullName, "helloworld_0.1.1")
	assert.Equal(t, dep.Version, "0.1.1")
	assert.Equal(t, dep.Sum, "7OO4YK2QuRWPq9C7KTzcWcti5yUnueCjptT3OXiPVeQ=")
	assert.NotEqual(t, dep.Source.Oci, nil)
	assert.Equal(t, dep.Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Oci.Repo, "kcl-lang/helloworld")
	assert.Equal(t, dep.Source.Oci.Tag, "0.1.1")
	assert.Equal(t, dep.LocalFullPath, testPath+"0.1.1")
	assert.Equal(t, err, nil)

	// Check whether the tar downloaded by `kpm add` has been deleted.
	assert.Equal(t, utils.DirExists(filepath.Join(testPath, "helloworld_0.1.1.tar")), false)

	assert.Equal(t, utils.DirExists(filepath.Join(getTestDir("download"), "helloworld")), true)

	// Check whether the reference and the dependency have the same hash.
	hashDep, err := utils.HashDir(dep.LocalFullPath)
	assert.Equal(t, err, nil)

	depRefPath := filepath.Join(getTestDir("download"), "helloworld")
	info, err := os.Lstat(depRefPath)
	assert.Equal(t, err, nil)

	if info.Mode()&os.ModeSymlink != 0 {
		depRefPath, err = os.Readlink(depRefPath)
		assert.Equal(t, err, nil)
	}

	hashRef, err := utils.HashDir(depRefPath)
	assert.Equal(t, err, nil)
	assert.Equal(t, hashDep, hashRef)
}

func TestDependencyGraph(t *testing.T) {
	testDir := getTestDir("test_dependency_graph")
	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "kcl.mod.lock")), false)
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(testDir)
	assert.Equal(t, err, nil)

	_, depGraph, err := kpmcli.InitGraphAndDownloadDeps(kclPkg)
	assert.Equal(t, err, nil)
	adjMap, err := depGraph.AdjacencyMap()
	assert.Equal(t, err, nil)

	edgeProp := graph.EdgeProperties{
		Attributes: map[string]string{},
		Weight:     0,
		Data:       nil,
	}

	assert.Equal(t, adjMap,
		map[string]map[string]graph.Edge[string]{
			"dependency_graph@0.0.1": {
				"teleport@0.1.0": {Source: "dependency_graph@0.0.1", Target: "teleport@0.1.0", Properties: edgeProp},
				"rabbitmq@0.0.1": {Source: "dependency_graph@0.0.1", Target: "rabbitmq@0.0.1", Properties: edgeProp},
				"agent@0.1.0":    {Source: "dependency_graph@0.0.1", Target: "agent@0.1.0", Properties: edgeProp},
			},
			"teleport@0.1.0": {
				"k8s@1.28": {Source: "teleport@0.1.0", Target: "k8s@1.28", Properties: edgeProp},
			},
			"rabbitmq@0.0.1": {
				"k8s@1.28": {Source: "rabbitmq@0.0.1", Target: "k8s@1.28", Properties: edgeProp},
			},
			"agent@0.1.0": {
				"k8s@1.28": {Source: "agent@0.1.0", Target: "k8s@1.28", Properties: edgeProp},
			},
			"k8s@1.28": {},
		},
	)
}

func TestCyclicDependency(t *testing.T) {
	testDir := getTestDir("test_cyclic_dependency")
	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "aaa")), true)
	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "aaa/kcl.mod")), true)
	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "bbb")), true)
	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "bbb/kcl.mod")), true)

	pkg_path := filepath.Join(testDir, "aaa")

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkg_path)
	assert.Equal(t, err, nil)

	currentDir, err := os.Getwd()
	assert.Equal(t, err, nil)
	err = os.Chdir(pkg_path)
	assert.Equal(t, err, nil)

	_, _, err = kpmcli.InitGraphAndDownloadDeps(kclPkg)
	assert.Equal(t, err, reporter.NewErrorEvent(
		reporter.CircularDependencyExist, nil, "adding bbb as a dependency results in a cycle",
	))

	err = os.Chdir(currentDir)
	assert.Equal(t, err, nil)
}

func TestParseKclModFile(t *testing.T) {
	// Create a temporary directory for testing
	testDir := initTestDir("test_parse_kcl_mod_file")

	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "kcl.mod")), false)

	kpmcli, err := NewKpmClient()
	assert.Nil(t, err, "error creating KpmClient")

	// Construct the modFilePath using filepath.Join
	modFilePath := filepath.Join(testDir, "kcl.mod")

	// Write modFileContent to modFilePath
	modFileContent := `
        [dependencies]
        teleport = "0.1.0"
        rabbitmq = "0.0.1"
        gitdep = { git = "git://example.com/repo.git", tag = "v1.0.0" }
        localdep = { path = "/path/to/local/dependency" }
    `

	err = os.WriteFile(modFilePath, []byte(modFileContent), 0644)
	assert.Nil(t, err, "error writing mod file")

	// Create a mock KclPkg
	mockKclPkg, err := kpmcli.LoadPkgFromPath(testDir)

	assert.Nil(t, err, "error loading package from path")

	// Test the ParseKclModFile function
	dependencies, err := kpmcli.ParseKclModFile(mockKclPkg)
	assert.Nil(t, err, "error parsing kcl.mod file")

	expectedDependencies := map[string]map[string]string{
		"teleport": {"version": "0.1.0"},
		"rabbitmq": {"version": "0.0.1"},
		"gitdep":   {"git": "git://example.com/repo.git", "tag": "v1.0.0"},
		"localdep": {"path": "/path/to/local/dependency"},
	}

	assert.Equal(t, expectedDependencies, dependencies, "parsed dependencies do not match expected dependencies")
}

func TestInitEmptyPkg(t *testing.T) {
	testDir := initTestDir("test_init_empty_mod")
	kclPkg := pkg.NewKclPkg(&opt.InitOptions{Name: "test_name", InitPath: testDir})
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	err = kpmcli.InitEmptyPkg(&kclPkg)
	assert.Equal(t, err, nil)

	testKclPkg, err := pkg.LoadKclPkg(testDir)
	assert.Equal(t, err, nil)
	assert.Equal(t, testKclPkg.ModFile.Pkg.Name, "test_name")
	assert.Equal(t, testKclPkg.ModFile.Pkg.Version, "0.0.1")
	assert.Equal(t, testKclPkg.ModFile.Pkg.Edition, runner.GetKclVersion())
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
	assert.Equal(t, err, nil)
	err = kclPkg.LockDepsVersion()
	assert.Equal(t, err, nil)

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

func TestResolveDepsWithOnlyKclMod(t *testing.T) {
	testDir := getTestDir("resolve_dep_with_kclmod")
	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "kcl.mod.lock")), false)
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(testDir)
	assert.Equal(t, err, nil)
	depsMap, err := kpmcli.ResolveDepsIntoMap(kclPkg)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(depsMap), 1)
	assert.Equal(t, utils.DirExists(filepath.Join(testDir, "kcl.mod.lock")), true)
	assert.Equal(t, depsMap["k8s"], filepath.Join(kpmcli.homePath, "k8s_1.17"))
	assert.Equal(t, utils.DirExists(filepath.Join(kpmcli.homePath, "k8s_1.17")), true)
	defer func() {
		err := os.Remove(filepath.Join(testDir, "kcl.mod.lock"))
		assert.Equal(t, err, nil)
	}()
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

	kclpkg, err = kpmcli.LoadPkgFromPath(testDir)
	assert.Equal(t, err, nil)
	kpmcli.homePath = "not_exist"
	res, err = kpmcli.ResolveDepsMetadataInJsonStr(kclpkg, false)
	fmt.Printf("err: %v\n", err)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(vendorDir), false)
	assert.Equal(t, utils.DirExists(filepath.Join(vendorDir, "konfig_v0.0.1")), false)
	jsonPath, err := json.Marshal(filepath.Join("not_exist", "konfig_v0.0.1"))
	assert.Equal(t, err, nil)
	expectedStr := fmt.Sprintf("{\"packages\":{\"konfig\":{\"name\":\"konfig\",\"manifest_path\":%s}}}", string(jsonPath))
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

func TestUpdateWithKclMod(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	testDir := getTestDir("test_update")
	src_testDir := filepath.Join(testDir, "test_update_kcl_mod")
	dest_testDir := filepath.Join(testDir, "test_update_kcl_mod_tmp")
	err = copy.Copy(src_testDir, dest_testDir)
	assert.Equal(t, err, nil)

	kclPkg, err := pkg.LoadKclPkg(dest_testDir)
	assert.Equal(t, err, nil)
	err = kpmcli.UpdateDeps(kclPkg)
	assert.Equal(t, err, nil)
	got_lock_file := filepath.Join(dest_testDir, "kcl.mod.lock")
	got_content, err := os.ReadFile(got_lock_file)
	assert.Equal(t, err, nil)

	expected_path := filepath.Join(dest_testDir, "expected")
	expected_content, err := os.ReadFile(expected_path)

	assert.Equal(t, err, nil)
	expect := strings.ReplaceAll(string(expected_content), "\r\n", "\n")
	assert.Equal(t, string(got_content), expect)

	defer func() {
		err := os.RemoveAll(dest_testDir)
		assert.Equal(t, err, nil)
	}()
}

func TestUpdateWithKclModlock(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	testDir := getTestDir("test_update")
	src_testDir := filepath.Join(testDir, "test_update_kcl_mod_lock")
	dest_testDir := filepath.Join(testDir, "test_update_kcl_mod_lock_tmp")
	err = copy.Copy(src_testDir, dest_testDir)
	assert.Equal(t, err, nil)

	kclPkg, err := pkg.LoadKclPkg(dest_testDir)
	assert.Equal(t, err, nil)
	err = kpmcli.UpdateDeps(kclPkg)
	assert.Equal(t, err, nil)
	got_lock_file := filepath.Join(dest_testDir, "kcl.mod.lock")
	got_content, err := os.ReadFile(got_lock_file)
	assert.Equal(t, err, nil)

	expected_path := filepath.Join(dest_testDir, "expected")
	expected_content, err := os.ReadFile(expected_path)

	assert.Equal(t, err, nil)
	assert.Equal(t, string(got_content), string(expected_content))

	defer func() {
		err := os.RemoveAll(dest_testDir)
		assert.Equal(t, err, nil)
	}()
}

func TestMetadataOffline(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	testDir := getTestDir("test_metadata_offline")
	kclMod := filepath.Join(testDir, "kcl.mod")
	uglyKclMod := filepath.Join(testDir, "ugly.kcl.mod")
	BeautifulKclMod := filepath.Join(testDir, "beautiful.kcl.mod")

	uglyContent, err := os.ReadFile(uglyKclMod)
	assert.Equal(t, err, nil)
	err = copy.Copy(uglyKclMod, kclMod)
	assert.Equal(t, err, nil)
	defer func() {
		err := os.Remove(kclMod)
		assert.Equal(t, err, nil)
	}()

	beautifulContent, err := os.ReadFile(BeautifulKclMod)
	assert.Equal(t, err, nil)
	kclPkg, err := pkg.LoadKclPkg(testDir)
	assert.Equal(t, err, nil)

	res, err := kpmcli.ResolveDepsMetadataInJsonStr(kclPkg, false)
	assert.Equal(t, err, nil)
	assert.Equal(t, res, "{\"packages\":{}}")
	content_after_metadata, err := os.ReadFile(kclMod)
	assert.Equal(t, err, nil)
	assert.Equal(t, string(content_after_metadata), string(uglyContent))

	res, err = kpmcli.ResolveDepsMetadataInJsonStr(kclPkg, true)
	assert.Equal(t, err, nil)
	assert.Equal(t, res, "{\"packages\":{}}")
	content_after_metadata, err = os.ReadFile(kclMod)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.RmNewline(string(content_after_metadata)), utils.RmNewline(string(beautifulContent)))
}

func TestAddWithNoSumCheck(t *testing.T) {
	pkgPath := getTestDir("test_add_no_sum_check")
	err := copy.Copy(filepath.Join(pkgPath, "kcl.mod.bak"), filepath.Join(pkgPath, "kcl.mod"))
	assert.Equal(t, err, nil)

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkgPath)
	assert.Equal(t, err, nil)

	opts := opt.AddOptions{
		LocalPath: pkgPath,
		RegistryOpts: opt.RegistryOptions{
			Oci: &opt.OciOptions{
				Reg:     "ghcr.io",
				Repo:    "kcl-lang",
				PkgName: "helloworld",
				Tag:     "0.1.0",
			},
		},
		NoSumCheck: true,
	}

	_, err = kpmcli.AddDepWithOpts(kclPkg, &opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), false)

	opts.NoSumCheck = false
	_, err = kpmcli.AddDepWithOpts(kclPkg, &opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), true)
	defer func() {
		_ = os.Remove(filepath.Join(pkgPath, "kcl.mod.lock"))
		_ = os.Remove(filepath.Join(pkgPath, "kcl.mod"))
	}()
}

func TestRunWithNoSumCheck(t *testing.T) {
	pkgPath := getTestDir("test_run_no_sum_check")

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	opts := opt.DefaultCompileOptions()
	opts.SetNoSumCheck(true)
	opts.SetPkgPath(pkgPath)

	_, err = kpmcli.CompileWithOpts(opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), false)

	opts = opt.DefaultCompileOptions()
	opts.SetPkgPath(pkgPath)
	opts.SetNoSumCheck(false)
	_, err = kpmcli.CompileWithOpts(opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), true)

	defer func() {
		_ = os.Remove(filepath.Join(pkgPath, "kcl.mod.lock"))
	}()
}

func TestRemoteRun(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	opts := opt.DefaultCompileOptions()
	gitOpts := git.NewCloneOptions("https://github.com/KusionStack/catalog", "", "0.1.2", "", "", nil)

	opts.SetEntries([]string{"models/samples/hellocollaset/prod/main.k"})
	result, err := kpmcli.CompileGitPkg(gitOpts, opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.GetRawJsonResult(), "[{\"hellocollaset\": {\"workload\": {\"containers\": {\"nginx\": {\"image\": \"nginx:v2\"}}}}}]")

	opts.SetEntries([]string{"models/samples/pgadmin/base/base.k"})
	result, err = kpmcli.CompileGitPkg(gitOpts, opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.GetRawJsonResult(), "[{\"pgadmin\": {\"workload\": {\"containers\": {\"pgadmin\": {\"image\": \"dpage/pgadmin4:latest\", \"env\": {\"PGADMIN_DEFAULT_EMAIL\": \"admin@admin.com\", \"PGADMIN_DEFAULT_PASSWORD\": \"secret://pgadmin-secret/pgadmin-default-password\", \"PGADMIN_PORT\": \"80\"}, \"resources\": {\"cpu\": \"500m\", \"memory\": \"512Mi\"}}}, \"secrets\": {\"pgadmin-secret\": {\"type\": \"opaque\", \"data\": {\"pgadmin-default-password\": \"*******\"}}}, \"replicas\": 1, \"ports\": [{\"port\": 80, \"protocol\": \"TCP\", \"public\": false}]}, \"database\": {\"pgadmin\": {\"type\": \"cloud\", \"version\": \"14.0\"}}}}]")

	opts.SetEntries([]string{"models/samples/wordpress/prod/main.k"})
	result, err = kpmcli.CompileGitPkg(gitOpts, opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.GetRawJsonResult(), "[{\"wordpress\": {\"workload\": {\"containers\": {\"wordpress\": {\"image\": \"wordpress:6.3\", \"env\": {\"WORDPRESS_DB_HOST\": \"$(KUSION_DB_HOST_WORDPRESS)\", \"WORDPRESS_DB_USER\": \"$(KUSION_DB_USERNAME_WORDPRESS)\", \"WORDPRESS_DB_PASSWORD\": \"$(KUSION_DB_PASSWORD_WORDPRESS)\", \"WORDPRESS_DB_NAME\": \"mysql\"}, \"resources\": {\"cpu\": \"500m\", \"memory\": \"512Mi\"}}}, \"replicas\": 1, \"ports\": [{\"port\": 80, \"protocol\": \"TCP\", \"public\": false}]}, \"database\": {\"wordpress\": {\"type\": \"cloud\", \"version\": \"8.0\"}}}}]")
}

func TestUpdateWithNoSumCheck(t *testing.T) {
	pkgPath := getTestDir("test_update_no_sum_check")
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	var buf bytes.Buffer
	kpmcli.SetLogWriter(&buf)

	kpmcli.SetNoSumCheck(true)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkgPath)
	assert.Equal(t, err, nil)

	err = kpmcli.UpdateDeps(kclPkg)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), false)
	assert.Equal(t, buf.String(), "")

	kpmcli.SetNoSumCheck(false)
	kclPkg, err = kpmcli.LoadPkgFromPath(pkgPath)
	assert.Equal(t, err, nil)

	err = kpmcli.UpdateDeps(kclPkg)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), true)
	assert.Equal(t, buf.String(), "adding 'helloworld' with version '0.1.1'\ndownloading 'kcl-lang/helloworld:0.1.1' from 'ghcr.io/kcl-lang/helloworld:0.1.1'\n")

	defer func() {
		_ = os.Remove(filepath.Join(pkgPath, "kcl.mod.lock"))
	}()
}

func TestAddWithDiffVersionNoSumCheck(t *testing.T) {
	pkgPath := getTestDir("test_add_diff_version")

	pkgWithSumCheckPath := filepath.Join(pkgPath, "no_sum_check")
	pkgWithSumCheckPathModBak := filepath.Join(pkgWithSumCheckPath, "kcl.mod.bak")
	pkgWithSumCheckPathMod := filepath.Join(pkgWithSumCheckPath, "kcl.mod")
	pkgWithSumCheckPathModExpect := filepath.Join(pkgWithSumCheckPath, "kcl.mod.expect")
	pkgWithSumCheckPathModLock := filepath.Join(pkgWithSumCheckPath, "kcl.mod.lock")

	err := copy.Copy(pkgWithSumCheckPathModBak, pkgWithSumCheckPathMod)
	assert.Equal(t, err, nil)

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkgWithSumCheckPath)
	assert.Equal(t, err, nil)

	opts := opt.AddOptions{
		LocalPath: pkgPath,
		RegistryOpts: opt.RegistryOptions{
			Oci: &opt.OciOptions{
				Reg:     "ghcr.io",
				Repo:    "kcl-lang",
				PkgName: "helloworld",
				Tag:     "0.1.1",
			},
		},
		NoSumCheck: true,
	}

	_, err = kpmcli.AddDepWithOpts(kclPkg, &opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(pkgWithSumCheckPathModLock), false)

	modContent, err := os.ReadFile(pkgWithSumCheckPathMod)
	modContentStr := strings.ReplaceAll(string(modContent), "\r\n", "")
	modContentStr = strings.ReplaceAll(string(modContentStr), "\n", "")
	assert.Equal(t, err, nil)
	modExpectContent, err := os.ReadFile(pkgWithSumCheckPathModExpect)
	modExpectContentStr := strings.ReplaceAll(string(modExpectContent), "\r\n", "")
	modExpectContentStr = strings.ReplaceAll(modExpectContentStr, "\n", "")
	assert.Equal(t, err, nil)
	assert.Equal(t, modContentStr, modExpectContentStr)

	opts.NoSumCheck = false
	_, err = kpmcli.AddDepWithOpts(kclPkg, &opts)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(pkgWithSumCheckPathModLock), true)
	modContent, err = os.ReadFile(pkgWithSumCheckPathMod)
	modContentStr = strings.ReplaceAll(string(modContent), "\r\n", "")
	modContentStr = strings.ReplaceAll(modContentStr, "\n", "")
	assert.Equal(t, err, nil)
	assert.Equal(t, modContentStr, modExpectContentStr)

	defer func() {
		_ = os.Remove(pkgWithSumCheckPathMod)
		_ = os.Remove(pkgWithSumCheckPathModLock)
	}()
}

func TestAddWithDiffVersionWithSumCheck(t *testing.T) {
	pkgPath := getTestDir("test_add_diff_version")

	pkgWithSumCheckPath := filepath.Join(pkgPath, "with_sum_check")
	pkgWithSumCheckPathModBak := filepath.Join(pkgWithSumCheckPath, "kcl.mod.bak")
	pkgWithSumCheckPathMod := filepath.Join(pkgWithSumCheckPath, "kcl.mod")
	pkgWithSumCheckPathModExpect := filepath.Join(pkgWithSumCheckPath, "kcl.mod.expect")
	pkgWithSumCheckPathModLock := filepath.Join(pkgWithSumCheckPath, "kcl.mod.lock")
	pkgWithSumCheckPathModLockBak := filepath.Join(pkgWithSumCheckPath, "kcl.mod.lock.bak")
	pkgWithSumCheckPathModLockExpect := filepath.Join(pkgWithSumCheckPath, "kcl.mod.lock.expect")

	err := copy.Copy(pkgWithSumCheckPathModBak, pkgWithSumCheckPathMod)
	assert.Equal(t, err, nil)
	err = copy.Copy(pkgWithSumCheckPathModLockBak, pkgWithSumCheckPathModLock)
	assert.Equal(t, err, nil)

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkgWithSumCheckPath)
	assert.Equal(t, err, nil)

	opts := opt.AddOptions{
		LocalPath: pkgPath,
		RegistryOpts: opt.RegistryOptions{
			Oci: &opt.OciOptions{
				Reg:     "ghcr.io",
				Repo:    "kcl-lang",
				PkgName: "helloworld",
				Tag:     "0.1.1",
			},
		},
	}

	_, err = kpmcli.AddDepWithOpts(kclPkg, &opts)
	assert.Equal(t, err, nil)

	modContent, err := os.ReadFile(pkgWithSumCheckPathMod)
	modContentStr := strings.ReplaceAll(string(modContent), "\r\n", "")
	modContentStr = strings.ReplaceAll(modContentStr, "\n", "")
	assert.Equal(t, err, nil)

	modExpectContent, err := os.ReadFile(pkgWithSumCheckPathModExpect)
	modExpectContentStr := strings.ReplaceAll(string(modExpectContent), "\r\n", "")
	modExpectContentStr = strings.ReplaceAll(modExpectContentStr, "\n", "")

	assert.Equal(t, err, nil)
	assert.Equal(t, modContentStr, modExpectContentStr)

	modLockContent, err := os.ReadFile(pkgWithSumCheckPathModLock)
	modLockContentStr := strings.ReplaceAll(string(modLockContent), "\r\n", "")
	modLockContentStr = strings.ReplaceAll(modLockContentStr, "\n", "")
	assert.Equal(t, err, nil)
	modLockExpectContent, err := os.ReadFile(pkgWithSumCheckPathModLockExpect)
	modLockExpectContentStr := strings.ReplaceAll(string(modLockExpectContent), "\r\n", "")
	modLockExpectContentStr = strings.ReplaceAll(modLockExpectContentStr, "\n", "")
	assert.Equal(t, err, nil)
	assert.Equal(t, modLockContentStr, modLockExpectContentStr)

	defer func() {
		_ = os.Remove(pkgWithSumCheckPathMod)
		_ = os.Remove(pkgWithSumCheckPathModLock)
	}()
}

func TestAddWithGitCommit(t *testing.T) {
	pkgPath := getTestDir("add_with_git_commit")

	testPkgPath := ""
	if runtime.GOOS == "windows" {
		testPkgPath = filepath.Join(pkgPath, "test_pkg_win")
	} else {
		testPkgPath = filepath.Join(pkgPath, "test_pkg")
	}

	testPkgPathModBak := filepath.Join(testPkgPath, "kcl.mod.bak")
	testPkgPathMod := filepath.Join(testPkgPath, "kcl.mod")
	testPkgPathModExpect := filepath.Join(testPkgPath, "kcl.mod.expect")
	testPkgPathModLock := filepath.Join(testPkgPath, "kcl.mod.lock")
	testPkgPathModLockBak := filepath.Join(testPkgPath, "kcl.mod.lock.bak")
	testPkgPathModLockExpect := filepath.Join(testPkgPath, "kcl.mod.lock.expect")

	err := copy.Copy(testPkgPathModBak, testPkgPathMod)
	assert.Equal(t, err, nil)
	err = copy.Copy(testPkgPathModLockBak, testPkgPathModLock)
	assert.Equal(t, err, nil)

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(testPkgPath)
	assert.Equal(t, err, nil)

	opts := opt.AddOptions{
		LocalPath: testPkgPath,
		RegistryOpts: opt.RegistryOptions{
			Git: &opt.GitOptions{
				Url:    "https://github.com/KusionStack/catalog.git",
				Commit: "a29e3db",
			},
		},
	}
	kpmcli.SetLogWriter(nil)
	_, err = kpmcli.AddDepWithOpts(kclPkg, &opts)

	assert.Equal(t, err, nil)

	modContent, err := os.ReadFile(testPkgPathMod)
	modContentStr := strings.ReplaceAll(string(modContent), "\r\n", "")
	modContentStr = strings.ReplaceAll(modContentStr, "\n", "")
	assert.Equal(t, err, nil)

	modExpectContent, err := os.ReadFile(testPkgPathModExpect)
	modExpectContentStr := strings.ReplaceAll(string(modExpectContent), "\r\n", "")
	modExpectContentStr = strings.ReplaceAll(modExpectContentStr, "\n", "")

	assert.Equal(t, err, nil)
	assert.Equal(t, modContentStr, modExpectContentStr)

	modLockContent, err := os.ReadFile(testPkgPathModLock)
	modLockContentStr := strings.ReplaceAll(string(modLockContent), "\r\n", "")
	modLockContentStr = strings.ReplaceAll(modLockContentStr, "\n", "")
	assert.Equal(t, err, nil)
	modLockExpectContent, err := os.ReadFile(testPkgPathModLockExpect)
	modLockExpectContentStr := strings.ReplaceAll(string(modLockExpectContent), "\r\n", "")
	modLockExpectContentStr = strings.ReplaceAll(modLockExpectContentStr, "\n", "")
	assert.Equal(t, err, nil)
	assert.Equal(t, modLockContentStr, modLockExpectContentStr)

	defer func() {
		_ = os.Remove(testPkgPathMod)
		_ = os.Remove(testPkgPathModLock)
	}()
}
