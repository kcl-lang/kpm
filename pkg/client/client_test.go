package client

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/dominikbraun/graph"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/hashicorp/go-version"
	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/module"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/features"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/runner"
	"kcl-lang.io/kpm/pkg/test"
	"kcl-lang.io/kpm/pkg/utils"
)

func TestWithGlobalLock(t *testing.T) {
	test.RunTestWithGlobalLock(t, "TestAddWithDiffVersionNoSumCheck", testAddWithDiffVersionNoSumCheck)
	test.RunTestWithGlobalLock(t, "TestAddWithDiffVersionWithSumCheck", testAddWithDiffVersionWithSumCheck)
	test.RunTestWithGlobalLock(t, "TestDownloadOci", testDownloadOci)
	test.RunTestWithGlobalLock(t, "TestAddWithOciDownloader", testAddWithOciDownloader)
	test.RunTestWithGlobalLock(t, "TestAddDefaultRegistryDep", testAddDefaultRegistryDep)
	test.RunTestWithGlobalLock(t, "TestAddWithNoSumCheck", testAddWithNoSumCheck)
	test.RunTestWithGlobalLock(t, "TestAddWithGitCommit", testAddWithGitCommit)
	test.RunTestWithGlobalLock(t, "TestDependenciesOrder", testDependenciesOrder)
	test.RunTestWithGlobalLock(t, "TestPkgWithInVendorMode", testPkgWithInVendorMode)
	test.RunTestWithGlobalLock(t, "TestResolveMetadataInJsonStrWithPackage", testResolveMetadataInJsonStrWithPackage)
	test.RunTestWithGlobalLock(t, "TestResolveMetadataInJsonStr", testResolveMetadataInJsonStr)
	test.RunTestWithGlobalLock(t, "testPackageCurrentPkgPath", testPackageCurrentPkgPath)
	test.RunTestWithGlobalLock(t, "TestResolveDepsWithOnlyKclMod", testResolveDepsWithOnlyKclMod)
	test.RunTestWithGlobalLock(t, "TestResolveDepsVendorMode", testResolveDepsVendorMode)
	test.RunTestWithGlobalLock(t, "TestCompileWithEntryFile", testCompileWithEntryFile)
	test.RunTestWithGlobalLock(t, "TestDownloadLatestOci", testDownloadLatestOci)
	test.RunTestWithGlobalLock(t, "TestDownloadGitWithPackage", testDownloadGitWithPackage)
	test.RunTestWithGlobalLock(t, "TestModandLockFilesWithGitPackageDownload", testModandLockFilesWithGitPackageDownload)
	test.RunTestWithGlobalLock(t, "TestDependencyGraph", testDependencyGraph)
	test.RunTestWithGlobalLock(t, "TestPull", testPull)
	test.RunTestWithGlobalLock(t, "TestPullWithInsecureSkipTLSverify", testPullWithInsecureSkipTLSverify)
	test.RunTestWithGlobalLock(t, "TestPullWithModSpec", testPullWithModSpec)
	test.RunTestWithGlobalLock(t, "testPullWithOnlySpec", testPullWithOnlySpec)
	test.RunTestWithGlobalLock(t, "TestGraph", testGraph)
	test.RunTestWithGlobalLock(t, "testCyclicDependency", testCyclicDependency)
	test.RunTestWithGlobalLock(t, "testNewKpmClient", testNewKpmClient)
	test.RunTestWithGlobalLock(t, "testLoadPkgFormOci", testLoadPkgFormOci)
	test.RunTestWithGlobalLock(t, "testAddWithLocalPath", testAddWithLocalPath)
	test.RunTestWithGlobalLock(t, "testRunLocalWithoutArgs", testRunLocalWithoutArgs)
	test.RunTestWithGlobalLock(t, "TestRunLocalWithArgs", testRunLocalWithArgs)
	test.RunTestWithGlobalLock(t, "testInsecureSkipTLSverifyOCIRegistry", testInsecureSkipTLSverifyOCIRegistry)
	test.RunTestWithGlobalLock(t, "testRunWithInsecureSkipTLSverify", testRunWithInsecureSkipTLSverify)
	test.RunTestWithGlobalLock(t, "TestAddDepsWithInsecureSkipTLSverify", testAddDepsWithInsecureSkipTLSverify)
	test.RunTestWithGlobalLock(t, "testPushWithInsecureSkipTLSverify", testPushWithInsecureSkipTLSverify)
	test.RunTestWithGlobalLock(t, "testMetadataOffline", testMetadataOffline)
}

// TestDownloadOci test download from oci registry.
func testDownloadOci(t *testing.T) {
	testPath := filepath.Join(getTestDir("download"), "helloworld_0.1.2")
	err := os.MkdirAll(testPath, 0755)
	assert.Equal(t, err, nil)

	defer func() {
		err := os.RemoveAll(getTestDir("download"))
		if err != nil {
			t.Errorf("Failed to remove directory: %v", err)
		}
	}()

	depFromOci := pkg.Dependency{
		Name:    "helloworld",
		Version: "0.1.2",
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Tag:  "0.1.2",
			},
		},
	}
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	dep, err := kpmcli.Download(&depFromOci, "", testPath)
	assert.Equal(t, err, nil)
	assert.Equal(t, dep.Name, "helloworld")
	assert.Equal(t, dep.FullName, "helloworld_0.1.2")
	assert.Equal(t, dep.Version, "0.1.2")
	assert.NotEqual(t, dep.Source.Oci, nil)
	assert.Equal(t, dep.Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Oci.Repo, "kcl-lang/helloworld")
	assert.Equal(t, dep.Source.Oci.Tag, "0.1.2")
	assert.Equal(t, dep.LocalFullPath, testPath)

	// Check whether the tar downloaded by `kpm add` has been deleted.
	downloadPath := getTestDir("download")
	assert.Equal(t, utils.DirExists(filepath.Join(downloadPath, "helloworld_0.1.2.tar")), false)

	assert.Equal(t, utils.DirExists(filepath.Join(downloadPath, "helloworld_0.1.2")), true)
	assert.Equal(t, utils.DirExists(filepath.Join(downloadPath, "helloworld")), false)
}

// TestDownloadLatestOci tests the case that the version is empty.
func testDownloadLatestOci(t *testing.T) {
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
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Tag:  "",
			},
		},
	}
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	dep, err := kpmcli.Download(&depFromOci, "", testPath)
	assert.Equal(t, err, nil)
	assert.Equal(t, dep.Name, "helloworld")
	assert.Equal(t, dep.FullName, "helloworld_0.1.4")
	assert.Equal(t, dep.Version, "0.1.4")
	assert.Equal(t, dep.Sum, "9J9HOMhdypaDYf0J7PqtpGTdlkbxkN0HFEYhosHhf4U=")
	assert.NotEqual(t, dep.Source.Oci, nil)
	assert.Equal(t, dep.Source.Oci.Reg, "ghcr.io")
	assert.Equal(t, dep.Source.Oci.Repo, "kcl-lang/helloworld")
	assert.Equal(t, dep.Source.Oci.Tag, "0.1.4")
	assert.Equal(t, dep.LocalFullPath, filepath.Join(getTestDir("download"), "helloworld_0.1.4"))
	assert.Equal(t, err, nil)

	// Check whether the tar downloaded by `kpm add` has been deleted.
	assert.Equal(t, utils.DirExists(filepath.Join(testPath, "helloworld_0.1.4.tar")), false)

	assert.Equal(t, utils.DirExists(filepath.Join(getTestDir("download"), "helloworld")), false)
}

func testDownloadGitWithPackage(t *testing.T) {
	testPath := filepath.Join(getTestDir("download"), "a_random_name")

	defer func() {
		err := os.RemoveAll(getTestDir("download"))
		if err != nil {
			t.Errorf("Failed to remove directory: %v", err)
		}
	}()

	err := os.MkdirAll(testPath, 0755)
	assert.Equal(t, err, nil)

	depFromGit := pkg.Dependency{
		Name:    "k8s",
		Version: "",
		Source: downloader.Source{
			Git: &downloader.Git{
				Url:     "https://github.com/kcl-lang/modules.git",
				Commit:  "bdd4d00a88bc3534ae50affa8328df2927fd2171",
				Package: "add-ndots",
			},
		},
	}

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	dep, err := kpmcli.Download(&depFromGit, "", testPath)

	assert.Equal(t, err, nil)
	assert.Equal(t, dep.Source.Git.Package, "add-ndots")
}

func testModandLockFilesWithGitPackageDownload(t *testing.T) {
	testPkgPath := getTestDir("test_mod_file_package")

	if runtime.GOOS == "windows" {
		testPkgPath = filepath.Join(testPkgPath, "test_pkg_win")
	} else {
		testPkgPath = filepath.Join(testPkgPath, "test_pkg")
	}

	testPkgPathMod := filepath.Join(testPkgPath, "kcl.mod")
	testPkgPathModBk := filepath.Join(testPkgPath, "kcl.mod.bk")
	testPkgPathModExpect := filepath.Join(testPkgPath, "expect.mod")
	testPkgPathModLock := filepath.Join(testPkgPath, "kcl.mod.lock")
	testPkgPathModLockBk := filepath.Join(testPkgPath, "kcl.mod.lock.bk")
	testPkgPathModLockExpect := filepath.Join(testPkgPath, "expect.mod.lock")

	if !utils.DirExists(testPkgPathMod) {
		err := copy.Copy(testPkgPathModBk, testPkgPathMod)
		assert.Equal(t, err, nil)
	}

	if !utils.DirExists(testPkgPathModLock) {
		err := copy.Copy(testPkgPathModLockBk, testPkgPathModLock)
		assert.Equal(t, err, nil)
	}

	defer func() {
		err := os.RemoveAll(testPkgPathMod)
		if err != nil {
			t.Errorf("Failed to remove directory: %v", err)
		}
		err = os.RemoveAll(testPkgPathModLock)
		if err != nil {
			t.Errorf("Failed to remove directory: %v", err)
		}
	}()

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	kclPkg, err := kpmcli.LoadPkgFromPath(testPkgPath)
	assert.Equal(t, err, nil)

	opts := opt.AddOptions{
		LocalPath: testPkgPath,
		RegistryOpts: opt.RegistryOptions{
			Git: &opt.GitOptions{
				Url:     "https://github.com/kcl-lang/flask-demo-kcl-manifests.git",
				Commit:  "8308200",
				Package: "cc",
			},
		},
	}

	_, err = kpmcli.AddDepWithOpts(kclPkg, &opts)
	assert.Equal(t, err, nil)

	modContent, err := os.ReadFile(testPkgPathMod)
	assert.Equal(t, err, nil)

	modExpectContent, err := os.ReadFile(testPkgPathModExpect)
	assert.Equal(t, err, nil)

	modContentStr := string(modContent)
	modExpectContentStr := string(modExpectContent)

	for _, str := range []*string{&modContentStr, &modExpectContentStr} {
		*str = strings.ReplaceAll(*str, " ", "")
		*str = strings.ReplaceAll(*str, "\r\n", "")
		*str = strings.ReplaceAll(*str, "\n", "")

		sumRegex := regexp.MustCompile(`sum\s*=\s*"[^"]+"`)
		*str = sumRegex.ReplaceAllString(*str, "")

		*str = strings.TrimRight(*str, ", \t\r\n")
	}

	assert.Equal(t, modExpectContentStr, modContentStr)

	modLockContent, err := os.ReadFile(testPkgPathModLock)
	assert.Equal(t, err, nil)

	modLockExpectContent, err := os.ReadFile(testPkgPathModLockExpect)
	assert.Equal(t, err, nil)

	modLockContentStr := string(modLockContent)
	modLockExpectContentStr := string(modLockExpectContent)

	for _, str := range []*string{&modLockContentStr, &modLockExpectContentStr} {
		*str = strings.ReplaceAll(*str, " ", "")
		*str = strings.ReplaceAll(*str, "\r\n", "")
		*str = strings.ReplaceAll(*str, "\n", "")

		sumRegex := regexp.MustCompile(`sum\s*=\s*"[^"]+"`)
		*str = sumRegex.ReplaceAllString(*str, "")

		*str = strings.TrimRight(*str, ", \t\r\n")
	}

	fmt.Println(modLockContentStr)

	assert.Equal(t, modLockExpectContentStr, modLockContentStr)
}

func testDependencyGraph(t *testing.T) {
	testWithoutPackageDir := filepath.Join(getTestDir("test_dependency_graph"), "without_package")
	assert.Equal(t, utils.DirExists(filepath.Join(testWithoutPackageDir, "kcl.mod.lock")), false)
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(testWithoutPackageDir)
	assert.Equal(t, err, nil)

	_, depGraph, err := kpmcli.InitGraphAndDownloadDeps(kclPkg)
	assert.Equal(t, err, nil)
	adjMap, err := depGraph.AdjacencyMap()
	assert.Equal(t, err, nil)

	m := func(Path, Version string) module.Version {
		return module.Version{Path: Path, Version: Version}
	}

	edgeProp := graph.EdgeProperties{
		Attributes: map[string]string{},
		Weight:     0,
		Data:       nil,
	}
	assert.Equal(t, adjMap,
		map[module.Version]map[module.Version]graph.Edge[module.Version]{
			m("dependency_graph", "0.0.1"): {
				m("teleport", "0.1.0"): {Source: m("dependency_graph", "0.0.1"), Target: m("teleport", "0.1.0"), Properties: edgeProp},
				m("rabbitmq", "0.0.1"): {Source: m("dependency_graph", "0.0.1"), Target: m("rabbitmq", "0.0.1"), Properties: edgeProp},
				m("agent", "0.1.0"):    {Source: m("dependency_graph", "0.0.1"), Target: m("agent", "0.1.0"), Properties: edgeProp},
			},
			m("teleport", "0.1.0"): {
				m("k8s", "1.28"): {Source: m("teleport", "0.1.0"), Target: m("k8s", "1.28"), Properties: edgeProp},
			},
			m("rabbitmq", "0.0.1"): {
				m("k8s", "1.28"): {Source: m("rabbitmq", "0.0.1"), Target: m("k8s", "1.28"), Properties: edgeProp},
			},
			m("agent", "0.1.0"): {
				m("k8s", "1.28"): {Source: m("agent", "0.1.0"), Target: m("k8s", "1.28"), Properties: edgeProp},
			},
			m("k8s", "1.28"): {},
		},
	)

	testWithPackageDir := filepath.Join(getTestDir("test_dependency_graph"), "with_package")
	assert.Equal(t, utils.DirExists(filepath.Join(testWithPackageDir, "kcl.mod.lock")), false)

	kpmcli, err = NewKpmClient()
	assert.Equal(t, err, nil)

	kclPkg, err = kpmcli.LoadPkgFromPath(testWithPackageDir)
	assert.Equal(t, err, nil)

	_, depGraph, err = kpmcli.InitGraphAndDownloadDeps(kclPkg)
	assert.Equal(t, err, nil)

	_, err = depGraph.AdjacencyMap()
	assert.Equal(t, err, nil)
}

func testCyclicDependency(t *testing.T) {
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
		reporter.CircularDependencyExist, nil, "adding aaa@0.0.1 as a dependency results in a cycle",
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
	modFileContent := `[dependencies]
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

func testUpdateKclModAndLock(t *testing.T, kpmcli *KpmClient) {
	testDir := initTestDir("test_data_add_deps")
	// Init an empty package
	kclPkg := pkg.NewKclPkg(&opt.InitOptions{
		Name:     "test_add_deps",
		InitPath: testDir,
	})
	err := kpmcli.InitEmptyPkg(&kclPkg)
	assert.Equal(t, err, nil)

	dep := pkg.Dependency{
		Name:     "name",
		FullName: "test_version",
		Version:  "test_version",
		Sum:      "test_sum",
		Source: downloader.Source{
			Git: &downloader.Git{
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
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "test_reg",
				Repo: "test_repo",
				Tag:  "test_tag",
			},
		},
	}

	kclPkg.Dependencies.Deps.Set("oci_test", oci_dep)
	kclPkg.ModFile.Dependencies.Deps.Set("oci_test", oci_dep)

	kclPkg.Dependencies.Deps.Set("test", dep)
	kclPkg.ModFile.Dependencies.Deps.Set("test", dep)

	err = kclPkg.ModFile.StoreModFile()
	assert.Equal(t, err, nil)
	err = kclPkg.LockDepsVersion()
	assert.Equal(t, err, nil)

	expectDir := getTestDir("expected")

	if gotKclMod, err := os.ReadFile(filepath.Join(testDir, "kcl.mod")); os.IsNotExist(err) {
		t.Errorf("failed to find kcl.mod.")
	} else {
		assert.Equal(t, kclPkg.Dependencies.Deps.Len(), 2)
		assert.Equal(t, kclPkg.ModFile.Deps.Len(), 2)
		expectKclMod, _ := os.ReadFile(filepath.Join(expectDir, "kcl.mod"))
		expectKclModReverse, _ := os.ReadFile(filepath.Join(expectDir, "kcl.reverse.mod"))

		gotKclModStr := utils.RmNewline(string(gotKclMod))
		expectKclModStr := utils.RmNewline(string(expectKclMod))
		expectKclModReverseStr := utils.RmNewline(string(expectKclModReverse))

		assert.Equal(t,
			true,
			(gotKclModStr == expectKclModStr || gotKclModStr == expectKclModReverseStr),
			"'%v'\n'%v'\n'%v'\n",
			gotKclModStr,
			expectKclModStr,
			expectKclModReverseStr,
		)
	}

	if gotKclModLock, err := os.ReadFile(filepath.Join(testDir, "kcl.mod.lock")); os.IsNotExist(err) {
		t.Errorf("failed to find kcl.mod.lock.")
	} else {
		assert.Equal(t, kclPkg.Dependencies.Deps.Len(), 2)
		assert.Equal(t, kclPkg.ModFile.Deps.Len(), 2)
		expectKclModLock, _ := os.ReadFile(filepath.Join(expectDir, "kcl.mod.lock"))
		expectKclModLockReverse, _ := os.ReadFile(filepath.Join(expectDir, "kcl.mod.reverse.lock"))

		gotKclModLockStr := utils.RmNewline(string(gotKclModLock))
		expectKclModLockStr := utils.RmNewline(string(expectKclModLock))
		expectKclModLockReverseStr := utils.RmNewline(string(expectKclModLockReverse))

		assert.Equal(t,
			true,
			(gotKclModLockStr == expectKclModLockStr) || (gotKclModLockStr == expectKclModLockReverseStr),
			"'%v'\n'%v'\n'%v'\n",
			gotKclModLockStr,
			expectKclModLockStr,
			expectKclModLockReverseStr,
		)
	}
}

func testResolveDepsWithOnlyKclMod(t *testing.T) {
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

func testResolveDepsVendorMode(t *testing.T) {
	testDir := getTestDir("resolve_deps")
	kpm_home := filepath.Join(testDir, "kpm_home")
	home_path := filepath.Join(testDir, "my_kcl_resolve_deps_vendor_mode")
	os.RemoveAll(home_path)
	kcl1Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl1_0.0.1"))
	kcl2Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl2_0.0.1"))

	depKcl1 := pkg.Dependency{
		Name:     "kcl1",
		FullName: "kcl1_0.0.1",
		Version:  "0.0.1",
		Sum:      kcl1Sum,
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/kcl1",
				Tag:  "0.0.1",
			},
		},
	}

	depKcl2 := pkg.Dependency{
		Name:     "kcl2",
		FullName: "kcl2_0.0.1",
		Version:  "0.0.1",
		Sum:      kcl2Sum,
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/kcl2",
				Tag:  "0.0.1",
			},
		},
	}

	mppTest := orderedmap.NewOrderedMap[string, pkg.Dependency]()
	mppTest.Set("kcl1", depKcl1)
	mppTest.Set("kcl2", depKcl2)
	kclPkg := pkg.KclPkg{
		ModFile: pkg.ModFile{
			HomePath: home_path,
			// Whether the current package uses the vendor mode
			// In the vendor mode, kpm will look for the package in the vendor subdirectory
			// in the current package directory.
			VendorMode: true,
			Dependencies: pkg.Dependencies{
				Deps: mppTest,
			},
		},
		HomePath: home_path,
		// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
		// not the dependencies in kcl.mod.
		Dependencies: pkg.Dependencies{
			Deps: mppTest,
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

func testCompileWithEntryFile(t *testing.T) {
	testDir := getTestDir("resolve_deps")
	kpm_home := filepath.Join(testDir, "kpm_home")
	home_path := filepath.Join(testDir, "my_kcl_compile")
	vendor_path := filepath.Join(home_path, "vendor")
	entry_file := filepath.Join(home_path, "main.k")
	os.RemoveAll(vendor_path)

	kcl1Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl1_0.0.1"))
	depKcl1 := pkg.Dependency{
		Name:     "kcl1",
		FullName: "kcl1_0.0.1",
		Version:  "0.0.1",
		Sum:      kcl1Sum,
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/kcl1",
				Tag:  "0.0.1",
			},
		},
	}
	kcl2Sum, _ := utils.HashDir(filepath.Join(kpm_home, "kcl2_0.0.1"))
	depKcl2 := pkg.Dependency{
		Name:     "kcl2",
		FullName: "kcl2_0.0.1",
		Version:  "0.0.1",
		Sum:      kcl2Sum,
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/kcl2",
				Tag:  "0.0.1",
			},
		},
	}

	mppTest := orderedmap.NewOrderedMap[string, pkg.Dependency]()
	mppTest.Set("kcl1", depKcl1)
	mppTest.Set("kcl2", depKcl2)

	kclPkg := pkg.KclPkg{
		ModFile: pkg.ModFile{
			HomePath: home_path,
			// Whether the current package uses the vendor mode
			// In the vendor mode, kpm will look for the package in the vendor subdirectory
			// in the current package directory.
			VendorMode: true,
			Dependencies: pkg.Dependencies{
				Deps: mppTest,
			},
		},
		HomePath: home_path,
		// The dependencies in the current kcl package are the dependencies of kcl.mod.lock,
		// not the dependencies in kcl.mod.
		Dependencies: pkg.Dependencies{
			Deps: mppTest,
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
	assert.Equal(t, utils.DirExists(filepath.Join(vendor_path, "kcl1_0.0.1")), true)
	assert.Equal(t, utils.DirExists(filepath.Join(vendor_path, "kcl2_0.0.1")), true)
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

func testPackageCurrentPkgPath(t *testing.T) {
	testDir := getTestDir("tar_kcl_pkg")

	kclPkg, err := pkg.LoadKclPkg(testDir)
	assert.Equal(t, err, nil)
	assert.Equal(t, kclPkg.GetPkgTag(), "0.0.1")
	assert.Equal(t, kclPkg.GetPkgName(), "test_tar")
	assert.Equal(t, kclPkg.GetPkgFullName(), "test_tar_0.0.1")
	assert.Equal(t, kclPkg.GetPkgTarName(), "test_tar_0.0.1.tar")

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

func testResolveMetadataInJsonStr(t *testing.T) {
	originalValue := os.Getenv(env.PKG_PATH)
	defer os.Setenv(env.PKG_PATH, originalValue)

	testDir := filepath.Join(getTestDir("resolve_metadata"), "without_package")

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kclpkg, err := kpmcli.LoadPkgFromPath(testDir)
	assert.Equal(t, err, nil)

	globalPkgPath, _ := env.GetAbsPkgPath()
	res, err := kpmcli.ResolveDepsMetadataInJsonStr(kclpkg, true)
	fmt.Printf("err: %v\n", err)
	assert.Equal(t, err, nil)

	expectedDep := pkg.DependenciesUI{
		Deps: make(map[string]pkg.Dependency),
	}

	expectedDep.Deps["flask_demo_kcl_manifests"] = pkg.Dependency{
		Name:          "flask_demo_kcl_manifests",
		FullName:      "flask-demo-kcl-manifests_ade147b",
		Version:       "0.1.0",
		LocalFullPath: filepath.Join(globalPkgPath, "flask-demo-kcl-manifests_ade147b"),
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
	assert.Equal(t, utils.DirExists(filepath.Join(vendorDir, "flask-demo-kcl-manifests_ade147b")), true)

	expectedDep.Deps["flask_demo_kcl_manifests"] = pkg.Dependency{
		Name:          "flask_demo_kcl_manifests",
		FullName:      "flask-demo-kcl-manifests_ade147b",
		Version:       "0.1.0",
		LocalFullPath: filepath.Join(vendorDir, "flask-demo-kcl-manifests_ade147b"),
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
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(vendorDir), false)
	assert.Equal(t, utils.DirExists(filepath.Join(vendorDir, "flask-demo-kcl-manifests_ade147b")), false)
	assert.Equal(t, err, nil)
	expectedPath := filepath.Join("not_exist", "flask-demo-kcl-manifests_ade147b")
	if runtime.GOOS == "windows" {
		expectedPath = strings.ReplaceAll(expectedPath, "\\", "\\\\")
	}
	expectedStr := "{\"packages\":{\"flask_demo_kcl_manifests\":{\"name\":\"flask_demo_kcl_manifests\",\"manifest_path\":\"\"}}}"
	assert.Equal(t, res, expectedStr)
	defer func() {
		if r := os.RemoveAll(expectedPath); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
}

func testResolveMetadataInJsonStrWithPackage(t *testing.T) {
	// Unit tests for package flag
	testDir := filepath.Join(getTestDir("resolve_metadata"), "with_package")

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	kclpkg, err := kpmcli.LoadPkgFromPath(testDir)
	assert.Equal(t, err, nil)

	globalPkgPath, _ := env.GetAbsPkgPath()

	res, err := kpmcli.ResolveDepsMetadataInJsonStr(kclpkg, true)
	fmt.Printf("err: %v\n", err)
	assert.Equal(t, err, nil)

	expectedDep := pkg.DependenciesUI{
		Deps: make(map[string]pkg.Dependency),
	}

	localFullPath, err := utils.FindPackage(filepath.Join(globalPkgPath, "flask-demo-kcl-manifests_8308200"), "cc")
	assert.Equal(t, err, nil)

	expectedDep.Deps["cc"] = pkg.Dependency{
		Name:          "cc",
		FullName:      "flask-demo-kcl-manifests_8308200",
		Version:       "8308200",
		LocalFullPath: localFullPath,
	}

	expectedDepStr, err := json.Marshal(expectedDep)
	assert.Equal(t, err, nil)

	assert.Equal(t, res, string(expectedDepStr))

	vendorDir := filepath.Join(testDir, "vendor")

	kpmcli, err = NewKpmClient()
	assert.Equal(t, err, nil)

	kclpkg, err = kpmcli.LoadPkgFromPath(testDir)
	assert.Equal(t, err, nil)

	if utils.DirExists(vendorDir) {
		err = os.RemoveAll(vendorDir)
		assert.Equal(t, err, nil)
	}

	kclpkg.SetVendorMode(true)

	res, err = kpmcli.ResolveDepsMetadataInJsonStr(kclpkg, true)
	assert.Equal(t, err, nil)

	assert.Equal(t, utils.DirExists(vendorDir), true)
	assert.Equal(t, utils.DirExists(filepath.Join(vendorDir, "flask-demo-kcl-manifests_8308200")), true)

	localFullPath, err = utils.FindPackage(filepath.Join(vendorDir, "flask-demo-kcl-manifests_8308200"), "cc")
	assert.Equal(t, err, nil)

	expectedDep = pkg.DependenciesUI{
		Deps: make(map[string]pkg.Dependency),
	}

	expectedDep.Deps["cc"] = pkg.Dependency{
		Name:          "cc",
		FullName:      "flask-demo-kcl-manifests_8308200",
		Version:       "8308200",
		LocalFullPath: localFullPath,
	}

	expectedDepStr, err = json.Marshal(expectedDep)
	assert.Equal(t, err, nil)

	assert.Equal(t, res, string(expectedDepStr))

	defer func() {
		err = os.RemoveAll(vendorDir)
		assert.Equal(t, err, nil)
	}()
}

func testPkgWithInVendorMode(t *testing.T) {
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

func testNewKpmClient(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kpmhome, err := env.GetAbsPkgPath()
	assert.Equal(t, err, nil)
	assert.Equal(t, kpmcli.homePath, kpmhome)
	assert.Equal(t, kpmcli.GetSettings().KpmConfFile, filepath.Join(kpmhome, ".kpm", "config", "kpm.json"))
	assert.Equal(t, kpmcli.GetSettings().CredentialsFile, filepath.Join(kpmhome, ".kpm", "config", "config.json"))
	assert.Equal(t, kpmcli.GetSettings().Conf.DefaultOciRepo, "kcl-lang")
	assert.Equal(t, kpmcli.GetSettings().Conf.DefaultOciRegistry, "ghcr.io")
	plainHttp, force := kpmcli.GetSettings().ForceOciPlainHttp()
	assert.Equal(t, plainHttp, false)
	assert.Equal(t, force, false)
}

func TestParseOciOptionFromString(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	oci_ref_with_tag := "test_oci_repo:test_oci_tag"
	ociOption, err := kpmcli.ParseOciOptionFromString(oci_ref_with_tag, "test_tag")
	assert.Equal(t, err, nil)
	assert.Equal(t, ociOption.Ref, "")
	assert.Equal(t, ociOption.Reg, "ghcr.io")
	assert.Equal(t, ociOption.Repo, "kcl-lang/test_oci_repo")
	assert.Equal(t, ociOption.Tag, "test_oci_tag")

	oci_ref_without_tag := "test_oci_repo:test_oci_tag"
	ociOption, err = kpmcli.ParseOciOptionFromString(oci_ref_without_tag, "test_tag")
	assert.Equal(t, err, nil)
	assert.Equal(t, ociOption.Ref, "")
	assert.Equal(t, ociOption.Reg, "ghcr.io")
	assert.Equal(t, ociOption.Repo, "kcl-lang/test_oci_repo")
	assert.Equal(t, ociOption.Tag, "test_oci_tag")

	oci_url_with_tag := "oci://test_reg/test_oci_repo"
	ociOption, err = kpmcli.ParseOciOptionFromString(oci_url_with_tag, "test_tag")
	assert.Equal(t, err, nil)
	assert.Equal(t, ociOption.Ref, "")
	assert.Equal(t, ociOption.Reg, "test_reg")
	assert.Equal(t, ociOption.Repo, "test_oci_repo")
	assert.Equal(t, ociOption.Tag, "test_tag")
}

func TestGetReleasesFromSource(t *testing.T) {
	sortVersions := func(versions []string) ([]string, error) {
		var vers []*version.Version
		for _, raw := range versions {
			v, err := version.NewVersion(raw)
			if err != nil {
				return nil, err
			}
			vers = append(vers, v)
		}
		sort.Slice(vers, func(i, j int) bool {
			return vers[i].LessThan(vers[j])
		})
		var res []string
		for _, v := range vers {
			res = append(res, v.Original())
		}
		return res, nil
	}

	releases, err := GetReleasesFromSource(pkg.GIT, "https://github.com/kcl-lang/kpm")
	assert.Equal(t, err, nil)
	length := len(releases)
	assert.True(t, length >= 5)
	releasesVersions, err := sortVersions(releases)
	assert.Equal(t, err, nil)
	assert.Equal(t, releasesVersions[:5], []string{"v0.1.0", "v0.2.0", "v0.2.1", "v0.2.2", "v0.2.3"})

	releases, err = GetReleasesFromSource(pkg.OCI, "oci://ghcr.io/kcl-lang/k8s")
	assert.Equal(t, err, nil)
	length = len(releases)
	assert.True(t, length >= 5)
	releasesVersions, err = sortVersions(releases)
	assert.Equal(t, err, nil)
	assert.Equal(t, releasesVersions[:5], []string{"1.14", "1.14.1", "1.15", "1.15.1", "1.16"})
}

func testUpdateWithKclMod(t *testing.T, kpmcli *KpmClient) {
	testDir := getTestDir("test_update")
	src_testDir := filepath.Join(testDir, "test_update_kcl_mod")
	dest_testDir := filepath.Join(testDir, "test_update_kcl_mod_tmp")
	err := copy.Copy(src_testDir, dest_testDir)
	assert.Equal(t, err, nil)

	kclPkg, err := kpmcli.LoadPkgFromPath(dest_testDir)
	assert.Equal(t, err, nil)
	err = kpmcli.UpdateDeps(kclPkg)
	fmt.Printf("err: %v\n", err)
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

func testUpdateWithKclModlock(t *testing.T, kpmcli *KpmClient) {
	testDir := getTestDir("test_update")
	src_testDir := filepath.Join(testDir, "test_update_kcl_mod_lock")
	dest_testDir := filepath.Join(testDir, "test_update_kcl_mod_lock_tmp")
	err := copy.Copy(src_testDir, dest_testDir)
	assert.Equal(t, err, nil)

	kclPkg, err := pkg.LoadKclPkg(dest_testDir)
	assert.Equal(t, err, nil)
	err = kpmcli.UpdateDeps(kclPkg)
	assert.Equal(t, err, nil)
	got_lock_file := filepath.Join(dest_testDir, "kcl.mod.lock")
	got_content, err := os.ReadFile(got_lock_file) // help
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

func TestNoDownloadWithMetadataOffline(t *testing.T) {
	testFunc := func(t *testing.T, kpmcli *KpmClient) {
		testDir := getTestDir("test_no_download_with_metadata_offline")
		kclPkg, err := pkg.LoadKclPkg(testDir)
		assert.Equal(t, err, nil)
		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)
		res, err := kpmcli.ResolveDepsMetadataInJsonStr(kclPkg, false)
		assert.Equal(t, err, nil)
		assert.Equal(t, buf.String(), "")
		assert.Equal(t, res, "{\"packages\":{\"kcl4\":{\"name\":\"kcl4\",\"manifest_path\":\"\"}}}")
	}

	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "test_no_download_with_metadata_offline", TestFunc: testFunc}})
}

func testMetadataOffline(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	testDir := getTestDir("test_metadata_offline")
	kclMod := filepath.Join(testDir, "kcl.mod")
	uglyKclMod := filepath.Join(testDir, "ugly.kcl.mod")

	uglyContent, err := os.ReadFile(uglyKclMod)
	assert.Equal(t, err, nil)
	err = copy.Copy(uglyKclMod, kclMod)
	assert.Equal(t, err, nil)
	defer func() {
		err := os.Remove(kclMod)
		assert.Equal(t, err, nil)
	}()

	kclPkg, err := pkg.LoadKclPkg(testDir)
	assert.Equal(t, err, nil)

	res, err := kpmcli.ResolveDepsMetadataInJsonStr(kclPkg, false)
	assert.Equal(t, err, nil)
	assert.Equal(t, res, "{\"packages\":{}}")
	content_after_metadata, err := os.ReadFile(kclMod)
	assert.Equal(t, err, nil)
	if runtime.GOOS == "windows" {
		uglyContent = []byte(strings.ReplaceAll(string(uglyContent), "\r\n", "\n"))
		content_after_metadata = []byte(strings.ReplaceAll(string(content_after_metadata), "\r\n", "\n"))
	}
	assert.Equal(t, string(content_after_metadata), string(uglyContent))

	res, err = kpmcli.ResolveDepsMetadataInJsonStr(kclPkg, true)
	assert.Equal(t, err, nil)
	assert.Equal(t, res, "{\"packages\":{}}")
	content_after_metadata, err = os.ReadFile(kclMod)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.RmNewline(string(content_after_metadata)), utils.RmNewline(string(uglyContent)))
}

func testAddWithNoSumCheck(t *testing.T) {
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
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Ref:  "helloworld",
				Tag:  "0.1.0",
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

func testUpdateWithNoSumCheck(t *testing.T, kpmcli *KpmClient) {
	pkgPath := getTestDir("test_update_no_sum_check")
	defer func() {
		_ = os.Remove(filepath.Join(pkgPath, "kcl.mod.lock"))
	}()

	var buf bytes.Buffer
	kpmcli.SetLogWriter(&buf)

	kpmcli.SetNoSumCheck(true)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkgPath)
	assert.Equal(t, err, nil)

	err = kpmcli.UpdateDeps(kclPkg)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), false)
	buf.Reset()

	kpmcli.SetNoSumCheck(false)
	kclPkg, err = kpmcli.LoadPkgFromPath(pkgPath)
	assert.Equal(t, err, nil)
	kclPkg.NoSumCheck = false

	err = kpmcli.UpdateDeps(kclPkg)
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(filepath.Join(pkgPath, "kcl.mod.lock")), true)
	assert.Equal(t, buf.String(), "")
}

func testAddWithDiffVersionNoSumCheck(t *testing.T) {
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
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Ref:  "helloworld",
				Tag:  "0.1.2",
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

func testAddWithDiffVersionWithSumCheck(t *testing.T) {
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
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Ref:  "helloworld",
				Tag:  "0.1.2",
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

func testAddWithGitCommit(t *testing.T) {
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
				Url:    "https://github.com/kcl-lang/flask-demo-kcl-manifests.git",
				Commit: "ade147b",
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

func testLoadPkgFormOci(t *testing.T) {
	type testCase struct {
		Reg  string
		Repo string
		Tag  string
		Name string
	}

	testCases := []testCase{
		{
			Reg:  "ghcr.io",
			Repo: "kusionstack/opsrule",
			Tag:  "0.0.9",
			Name: "opsrule",
		},
		{
			Reg:  "ghcr.io",
			Repo: "kcl-lang/helloworld",
			Tag:  "0.1.2",
			Name: "helloworld",
		},
	}

	cli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	pkgPath := getTestDir("test_load_pkg_from_oci")

	for _, tc := range testCases {
		localpath := filepath.Join(pkgPath, tc.Name)

		err = os.MkdirAll(localpath, 0755)
		assert.Equal(t, err, nil)
		defer func() {
			err := os.RemoveAll(localpath)
			assert.Equal(t, err, nil)
		}()

		kclpkg, err := cli.DownloadPkgFromOci(&downloader.Oci{
			Reg:  tc.Reg,
			Repo: tc.Repo,
			Tag:  tc.Tag,
		}, localpath)
		assert.Equal(t, err, nil)
		assert.Equal(t, kclpkg.GetPkgName(), tc.Name)
	}
}

func testAddWithLocalPath(t *testing.T) {

	testpath := getTestDir("add_with_local_path")

	initpath := filepath.Join(testpath, "init")
	tmppath := filepath.Join(testpath, "tmp")
	expectpath := filepath.Join(testpath, "expect")

	defer func() {
		err := os.RemoveAll(tmppath)
		assert.Equal(t, err, nil)
	}()

	err := copy.Copy(initpath, tmppath)
	assert.Equal(t, err, nil)

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kpmcli.SetLogWriter(nil)

	tmpPkgPath := filepath.Join(tmppath, "pkg")
	opts := opt.AddOptions{
		LocalPath: tmpPkgPath,
		RegistryOpts: opt.RegistryOptions{
			Oci: &opt.OciOptions{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Ref:  "helloworld",
				Tag:  "0.1.1",
			},
		},
	}

	kclPkg, err := kpmcli.LoadPkgFromPath(tmpPkgPath)
	assert.Equal(t, err, nil)

	_, err = kpmcli.AddDepWithOpts(kclPkg, &opts)
	assert.Equal(t, err, nil)

	gotpkg, err := kpmcli.LoadPkgFromPath(tmpPkgPath)
	assert.Equal(t, err, nil)
	expectpath = filepath.Join(expectpath, "pkg")
	expectpkg, err := kpmcli.LoadPkgFromPath(expectpath)
	assert.Equal(t, err, nil)

	assert.Equal(t, gotpkg.Dependencies.Deps.Len(), expectpkg.Dependencies.Deps.Len())
	assert.Equal(t, gotpkg.Dependencies.Deps.GetOrDefault("dep_pkg", pkg.TestPkgDependency).FullName, expectpkg.Dependencies.Deps.GetOrDefault("dep_pkg", pkg.TestPkgDependency).FullName)
	assert.Equal(t, gotpkg.Dependencies.Deps.GetOrDefault("dep_pkg", pkg.TestPkgDependency).Version, expectpkg.Dependencies.Deps.GetOrDefault("dep_pkg", pkg.TestPkgDependency).Version)
	assert.Equal(t, gotpkg.Dependencies.Deps.GetOrDefault("dep_pkg", pkg.TestPkgDependency).LocalFullPath, filepath.Join(tmppath, "dep_pkg"))
	assert.Equal(t, gotpkg.Dependencies.Deps.GetOrDefault("dep_pkg", pkg.TestPkgDependency).Source.Local.Path, "../dep_pkg")
}

func TestLoadOciUrlDiffSetting(t *testing.T) {
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	testPath := getTestDir("diffsettings")

	pkg_local, err := kpmcli.LoadPkgFromPath(testPath)
	assert.Equal(t, err, nil)
	assert.Equal(t, pkg_local.ModFile.Deps.Len(), 1)
	assert.Equal(t, pkg_local.ModFile.Deps.GetOrDefault("oci_pkg", pkg.TestPkgDependency).Oci.Reg, "docker.io")
	assert.Equal(t, pkg_local.ModFile.Deps.GetOrDefault("oci_pkg", pkg.TestPkgDependency).Oci.Repo, "test/oci_pkg")
	assert.Equal(t, pkg_local.ModFile.Deps.GetOrDefault("oci_pkg", pkg.TestPkgDependency).Oci.Tag, "0.0.1")
	assert.Equal(t, err, nil)
}

func testAddWithOciDownloader(t *testing.T) {
	kpmCli, err := NewKpmClient()
	path := getTestDir("test_oci_downloader")
	assert.Equal(t, err, nil)

	kpmCli.DepDownloader = downloader.NewOciDownloader("linux/amd64")
	kpkg, err := kpmCli.LoadPkgFromPath(filepath.Join(path, "add_dep", "pkg"))
	assert.Equal(t, err, nil)
	dep := pkg.Dependency{
		Name:     "helloworld",
		FullName: "helloworld_0.0.3",
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "zong-zhe/helloworld",
				Tag:  "0.0.3",
			},
		},
	}
	kpkg.HomePath = filepath.Join(path, "add_dep", "pkg")
	err = kpmCli.AddDepToPkg(kpkg, &dep)
	assert.Equal(t, err, nil)
	kpkg.NoSumCheck = false
	err = kpkg.UpdateModAndLockFile()
	assert.Equal(t, err, nil)

	expectmod := filepath.Join(path, "add_dep", "pkg", "except")
	expectmodContent, err := os.ReadFile(expectmod)
	assert.Equal(t, err, nil)
	gotContent, err := os.ReadFile(filepath.Join(path, "add_dep", "pkg", "kcl.mod"))
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.RmNewline(string(expectmodContent)), utils.RmNewline(string(gotContent)))
}

func TestAddLocalPath(t *testing.T) {

	kpmCli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	path := getTestDir("test_add_local_path")
	err = copy.Copy(filepath.Join(path, "kcl.mod.bak"), filepath.Join(path, "kcl.mod"))
	assert.Equal(t, err, nil)

	kclPkg, err := kpmCli.LoadPkgFromPath(path)
	assert.Equal(t, err, nil)

	opts := opt.AddOptions{
		LocalPath: path,
		RegistryOpts: opt.RegistryOptions{
			Local: &opt.LocalOptions{
				Path: filepath.Join(path, "dep"),
			},
		},
	}
	_, err = kpmCli.AddDepWithOpts(kclPkg, &opts)
	assert.Equal(t, err, nil)

	modFile, err := kpmCli.LoadModFile(path)
	assert.Equal(t, err, nil)
	assert.Equal(t, modFile.Deps.Len(), 1)
	assert.Equal(t, modFile.Deps.GetOrDefault("dep", pkg.Dependency{}).Name, "dep")
	assert.Equal(t, modFile.Deps.GetOrDefault("dep", pkg.Dependency{}).LocalFullPath, filepath.Join(path, "dep"))

	kclPkg1, err := kpmCli.LoadPkgFromPath(path)
	assert.Equal(t, err, nil)
	assert.Equal(t, kclPkg1.Dependencies.Deps.GetOrDefault("dep", pkg.Dependency{}).Name, "dep")
	assert.Equal(t, kclPkg1.Dependencies.Deps.GetOrDefault("dep", pkg.Dependency{}).FullName, "dep_0.0.1")
	assert.Equal(t, kclPkg1.Dependencies.Deps.GetOrDefault("dep", pkg.Dependency{}).Version, "0.0.1")
	defer func() {
		_ = os.Remove(filepath.Join(path, "kcl.mod.lock"))
		_ = os.Remove(filepath.Join(path, "kcl.mod"))
	}()
}

func testAddDefaultRegistryDep(t *testing.T) {
	type testCase struct {
		tag           string
		pkgPath       string
		modBak        string
		mod           string
		modExpect     string
		modLockBak    string
		modLock       string
		modLockExpect string
	}

	rootTestPath := getTestDir("add_with_default_dep")
	testCases := []testCase{
		{
			tag:     "",
			pkgPath: filepath.Join(rootTestPath, "no_tag"),
		},
		{
			tag:     "0.1.2",
			pkgPath: filepath.Join(rootTestPath, "with_tag"),
		},
	}

	for _, tc := range testCases {
		tc.modBak = filepath.Join(tc.pkgPath, "kcl.mod.bak")
		tc.mod = filepath.Join(tc.pkgPath, "kcl.mod")
		tc.modExpect = filepath.Join(tc.pkgPath, "kcl.mod.expect")
		tc.modLockBak = filepath.Join(tc.pkgPath, "kcl.mod.lock.bak")
		tc.modLock = filepath.Join(tc.pkgPath, "kcl.mod.lock")
		tc.modLockExpect = filepath.Join(tc.pkgPath, "kcl.mod.lock.expect")

		err := copy.Copy(tc.modBak, tc.mod)
		assert.Equal(t, err, nil)
		err = copy.Copy(tc.modLockBak, tc.modLock)
		assert.Equal(t, err, nil)

		kpmcli, err := NewKpmClient()
		assert.Equal(t, err, nil)

		kclPkg, err := kpmcli.LoadPkgFromPath(tc.pkgPath)
		assert.Equal(t, err, nil)

		opts := opt.AddOptions{
			LocalPath: tc.pkgPath,
			RegistryOpts: opt.RegistryOptions{
				Registry: &opt.OciOptions{
					Reg:  "ghcr.io",
					Repo: "kcl-lang/helloworld",
					Ref:  "helloworld",
					Tag:  tc.tag,
				},
			},
		}

		_, err = kpmcli.AddDepWithOpts(kclPkg, &opts)
		assert.Equal(t, err, nil)

		verifyFileContent(t, tc.mod, tc.modExpect)
		lockGot, err := os.ReadFile(tc.modLock)
		assert.Equal(t, err, nil)
		lockExpect, err := os.ReadFile(tc.modLockExpect)
		assert.Equal(t, err, nil)
		assert.Contains(t, utils.RmNewline(string(lockGot)), utils.RmNewline(string(lockExpect)))

		defer func() {
			_ = os.Remove(tc.mod)
			_ = os.Remove(tc.modLock)
		}()
	}
}

func verifyFileContent(t *testing.T, filePath, expectPath string) {
	content, err := os.ReadFile(filePath)
	assert.Equal(t, err, nil)
	contentStr := strings.ReplaceAll(string(content), "\r\n", "")
	contentStr = strings.ReplaceAll(contentStr, "\n", "")

	expectContent, err := os.ReadFile(expectPath)
	assert.Equal(t, err, nil)
	expectContentStr := strings.ReplaceAll(string(expectContent), "\r\n", "")
	expectContentStr = strings.ReplaceAll(expectContentStr, "\n", "")

	assert.Equal(t, contentStr, expectContentStr)
}

func testUpdateDefaultRegistryDep(t *testing.T, kpmcli *KpmClient) {
	pkgPath := getTestDir("update_with_default_dep")

	pkgWithSumCheckPathModBak := filepath.Join(pkgPath, "kcl.mod.bak")
	pkgWithSumCheckPathMod := filepath.Join(pkgPath, "kcl.mod")
	pkgWithSumCheckPathModExpect := filepath.Join(pkgPath, "kcl.mod.expect")

	pkgWithSumCheckPathModLockBak := filepath.Join(pkgPath, "kcl.mod.lock.bak")
	pkgWithSumCheckPathModLock := filepath.Join(pkgPath, "kcl.mod.lock")
	pkgWithSumCheckPathModLockExpect := filepath.Join(pkgPath, "kcl.mod.lock.expect")

	err := copy.Copy(pkgWithSumCheckPathModBak, pkgWithSumCheckPathMod)
	assert.Equal(t, err, nil)
	err = copy.Copy(pkgWithSumCheckPathModLockBak, pkgWithSumCheckPathModLock)
	assert.Equal(t, err, nil)

	kclPkg, err := kpmcli.LoadPkgFromPath(pkgPath)
	assert.Equal(t, err, nil)

	err = kpmcli.UpdateDeps(kclPkg)
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

func testRunDefaultRegistryDep(t *testing.T, kpmcli *KpmClient) {
	pkgPath := getTestDir("run_with_default_dep")

	pkgWithSumCheckPathModBak := filepath.Join(pkgPath, "kcl.mod.bak")
	pkgWithSumCheckPathMod := filepath.Join(pkgPath, "kcl.mod")
	pkgWithSumCheckPathModExpect := filepath.Join(pkgPath, "kcl.mod.expect")

	pkgWithSumCheckPathModLockBak := filepath.Join(pkgPath, "kcl.mod.lock.bak")
	pkgWithSumCheckPathModLock := filepath.Join(pkgPath, "kcl.mod.lock")
	pkgWithSumCheckPathModLockExpect := filepath.Join(pkgPath, "kcl.mod.lock.expect")

	err := copy.Copy(pkgWithSumCheckPathModBak, pkgWithSumCheckPathMod)
	assert.Equal(t, err, nil)
	err = copy.Copy(pkgWithSumCheckPathModLockBak, pkgWithSumCheckPathModLock)
	assert.Equal(t, err, nil)

	kclPkg, err := kpmcli.LoadPkgFromPath(pkgPath)
	assert.Equal(t, err, nil)

	opts := opt.DefaultCompileOptions()
	opts.Merge(kcl.WithWorkDir(pkgPath)).Merge(kcl.WithKFilenames(filepath.Join(pkgPath, "main.k")))
	compiler := runner.NewCompilerWithOpts(opts)

	res, err := kpmcli.Compile(kclPkg, compiler)
	assert.Equal(t, err, nil)
	assert.Equal(t, res.GetRawYamlResult(), "a: Hello World!")

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

func testDependenciesOrder(t *testing.T) {
	pkgPath := getTestDir("test_dep_order")

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkgPath)
	assert.Equal(t, err, nil)

	err = kpmcli.UpdateDeps(kclPkg)
	assert.Equal(t, err, nil)

	got, err := os.ReadFile(filepath.Join(pkgPath, "kcl.mod"))
	assert.Equal(t, err, nil)

	expect, err := os.ReadFile(filepath.Join(pkgPath, "expect.mod"))
	assert.Equal(t, err, nil)

	assert.Equal(t, utils.RmNewline(string(got)), utils.RmNewline(string(expect)))
}

func testRunLocalWithoutArgs(t *testing.T) {
	pkgPath := getTestDir("test_run_options")

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	logbuf := new(bytes.Buffer)
	kpmcli.SetLogWriter(logbuf)

	tests := []struct {
		workdirSuffix string
		withVendor    bool
		diagnostic    string
		expected      string
	}{
		{"run_0", false, "", "The_first_kcl_program: Hello World!"},
		{"run_1", false, "", "The_sub_kcl_program: Hello Sub World!"},
		{"run_2", false, "", "The_sub_kcl_program: Hello Sub World!"},
		{"run_3", false, "", "The_yaml_sub_kcl_program: Hello Yaml Sub World!"},
		{"run_4", true, "", "a: A package in vendor path"},
		{"run_5", true, "", "kcl_6: KCL 6\na: sub6\nkcl_7: KCL 7\nb: sub7"},
		{filepath.Join("run_6", "main"), true, "", "The_sub_kcl_program: Hello Sub World!\nThe_first_kcl_program: Hello World!"},
		{"run_7", true, "", "hello: Hello World!\nThe_first_kcl_program: Hello World!"},
		{filepath.Join("run_8", "sub"), true, "", "sub: Hello Sub !"},
	}

	for _, test := range tests {
		workdir := filepath.Join(pkgPath, "no_args", test.workdirSuffix)
		res, err := kpmcli.Run(
			WithWorkDir(workdir),
			WithVendor(test.withVendor),
		)

		assert.Equal(t, err, nil)
		assert.Equal(t, logbuf.String(), test.diagnostic)
		assert.Equal(t, res.GetRawYamlResult(), test.expected)
		logbuf.Reset()
	}
}

func testRunLocalWithArgs(t *testing.T) {
	pkgPath := getTestDir("test_run_options")

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	logbuf := new(bytes.Buffer)
	kpmcli.SetLogWriter(logbuf)

	tests := []struct {
		inputs        []string
		settingsFiles []string
		workdir       string
		withVendor    bool
		diagnostic    string
		expected      string
	}{
		{
			[]string{filepath.Join(pkgPath, "with_args", "run_0", "main.k")}, []string{}, filepath.Join(pkgPath, "with_args", "run_0"),
			false, "", "The_first_kcl_program: Hello World!"},
		{
			[]string{filepath.Join(pkgPath, "with_args", "run_1", "main.k")}, []string{}, filepath.Join(pkgPath, "with_args", "run_1"),
			false, "", "The_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_2", "base.k"),
			filepath.Join(pkgPath, "with_args", "run_2", "main.k"),
		}, []string{}, filepath.Join(pkgPath, "with_args", "run_2"), false, "", "base: Base\nThe_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_3", "main.k"),
		}, []string{}, filepath.Join(pkgPath, "with_args", "run_3"), false, "", "The_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_4", "main.k"),
		}, []string{}, filepath.Join(pkgPath, "with_args", "run_4"), false, "", "The_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_5"),
		}, []string{}, filepath.Join(pkgPath, "with_args", "run_5"), false, "", "The_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_6"),
		}, []string{}, filepath.Join(pkgPath, "with_args", "run_6"), false, "", "The_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_7"),
		}, []string{}, filepath.Join(pkgPath, "with_args", "run_7"), false, "", "base: Base\nThe_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_8"),
		}, []string{}, filepath.Join(pkgPath, "with_args", "run_8"), false, "", "sub: SUB"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_9"),
		}, []string{}, filepath.Join(pkgPath, "with_args", "run_9"), false, "", "The_sub_kcl_program: Hello Sub World!"},
		{[]string{}, []string{
			filepath.Join(pkgPath, "with_args", "run_10", "sub", "kcl.yaml"),
		}, filepath.Join(pkgPath, "with_args", "run_10"), false, "", "The_sub_kcl_program_1: Hello Sub World 1!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_11", "sub", "sub.k"),
		}, []string{
			filepath.Join(pkgPath, "with_args", "run_11", "sub", "kcl.yaml"),
		}, filepath.Join(pkgPath, "with_args", "run_11"), false, "", "The_sub_kcl_program: Hello Sub World!"},
		{
			[]string{filepath.Join(pkgPath, "with_args", "run_0", "main.k")}, []string{}, filepath.Join(pkgPath, "with_args", "run_0"),
			false, "", "The_first_kcl_program: Hello World!"},
		{
			[]string{filepath.Join(pkgPath, "with_args", "run_1", "main.k")}, []string{}, filepath.Join(pkgPath, "with_args", "run_1"),
			false, "", "The_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_2", "base.k"),
			filepath.Join(pkgPath, "with_args", "run_2", "main.k"),
		}, []string{}, "", false, "", "base: Base\nThe_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_3", "main.k"),
		}, []string{}, "", false, "", "The_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_4", "main.k"),
		}, []string{}, "", false, "", "The_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_5"),
		}, []string{}, "", false, "", "The_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_6"),
		}, []string{}, "", false, "", "The_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_7"),
		}, []string{}, "", false, "", "base: Base\nThe_first_kcl_program: Hello World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_8"),
		}, []string{}, "", false, "", "sub: SUB"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_9"),
		}, []string{}, "", false, "", "The_sub_kcl_program: Hello Sub World!"},
		{[]string{}, []string{
			filepath.Join(pkgPath, "with_args", "run_10", "sub", "kcl.yaml"),
		}, "", false, "", "The_sub_kcl_program_1: Hello Sub World 1!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_11", "sub", "sub.k"),
		}, []string{
			filepath.Join(pkgPath, "with_args", "run_11", "sub", "kcl.yaml"),
		}, "", false, "", "The_sub_kcl_program: Hello Sub World!"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_12", "sub1", "main.k"),
			filepath.Join(pkgPath, "with_args", "run_12", "sub2", "main.k"),
		}, []string{}, "", false, "", "sub1: 1\nsub2: 2"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_13", "temp"),
		}, []string{}, "", false, "", "temp: non-k-file"},
		{[]string{
			filepath.Join(pkgPath, "with_args", "run_14", "**", "*.k"),
		}, []string{}, "", false, "", "main: main\nmain1: main1\nsub: sub\nsub1: sub1"},
	}

	for _, test := range tests {
		res, err := kpmcli.Run(
			WithRunSourceUrls(test.inputs),
			WithSettingFiles(test.settingsFiles),
			WithWorkDir(test.workdir),
		)

		assert.Equal(t, err, nil)
		assert.Equal(t, logbuf.String(), test.diagnostic)
		assert.Equal(t, res.GetRawYamlResult(), test.expected)
		logbuf.Reset()
	}
}

func testRunRemoteWithArgsInvalid(t *testing.T, kpmcli *KpmClient) {
	logbuf := new(bytes.Buffer)
	kpmcli.SetLogWriter(logbuf)

	type testCase struct {
		sourceURL      string
		expectedLog    string
		expectedErrMsg string
	}

	testCases := []testCase{
		{
			sourceURL:      "git://github.com/kcl-lang/flask-demo-kcl-manifests?commit=8308200&mod=cc:0.0.2",
			expectedLog:    "cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests' with commit '8308200'\n",
			expectedErrMsg: "package 'cc:0.0.2' not found",
		},
	}

	for _, tc := range testCases {
		_, err := kpmcli.Run(WithRunSourceUrl(tc.sourceURL))
		assert.Equal(t, err.Error(), tc.expectedErrMsg)
		assert.Equal(t, logbuf.String(), tc.expectedLog)
		logbuf.Reset()
	}
}

func testRunRemoteWithArgs(t *testing.T, kpmcli *KpmClient) {
	pkgPath := getTestDir("test_run_options")

	logbuf := new(bytes.Buffer)
	kpmcli.SetLogWriter(logbuf)

	type testCase struct {
		sourceURL      string
		expectedLog    string
		expectedYaml   string
		expectedYamlFn func(pkgPath string) (string, error) // Function to dynamically get expected YAML
	}

	testCases := []testCase{
		{
			sourceURL:    "oci://ghcr.io/kcl-lang/helloworld?tag=0.1.2",
			expectedLog:  "downloading 'kcl-lang/helloworld:0.1.2' from 'ghcr.io/kcl-lang/helloworld:0.1.2'\n",
			expectedYaml: "The_first_kcl_program: Hello World!",
		},
		{
			sourceURL:   "git://github.com/kcl-lang/flask-demo-kcl-manifests?branch=main",
			expectedLog: "cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests' with branch 'main'\n",
			expectedYamlFn: func(pkgPath string) (string, error) {
				expected, err := os.ReadFile(filepath.Join(pkgPath, "remote", "expect_1.yaml"))
				return string(expected), err
			},
		},
		{
			sourceURL:   "git://github.com/kcl-lang/flask-demo-kcl-manifests?commit=ade147b",
			expectedLog: "cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests' with commit 'ade147b'\n",
			expectedYamlFn: func(pkgPath string) (string, error) {
				expected, err := os.ReadFile(filepath.Join(pkgPath, "remote", "expect_2.yaml"))
				return string(expected), err
			},
		},
		{
			sourceURL:   "git://github.com/kcl-lang/flask-demo-kcl-manifests?commit=8308200&mod=cc:0.0.1",
			expectedLog: "cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests' with commit '8308200'\n",
			expectedYamlFn: func(pkgPath string) (string, error) {
				expected, err := os.ReadFile(filepath.Join(pkgPath, "remote", "expect_3.yaml"))
				return string(expected), err
			},
		},
	}

	for i, tc := range testCases {
		res, err := kpmcli.Run(WithRunSourceUrl(tc.sourceURL))
		assert.Equal(t, err, nil, "%v-st", i)
		assert.Equal(t, logbuf.String(), tc.expectedLog)

		var expectedYaml string
		if tc.expectedYamlFn != nil {
			var err error
			expectedYaml, err = tc.expectedYamlFn(pkgPath)
			assert.Equal(t, err, nil)
		} else {
			expectedYaml = tc.expectedYaml
		}

		assert.Equal(t, utils.RmNewline(res.GetRawYamlResult()), utils.RmNewline(expectedYaml))
		logbuf.Reset()
	}
}

func testRunInVendor(t *testing.T, kpmcli *KpmClient) {
	pkgPath := getTestDir("test_run_in_vendor")
	workdir := filepath.Join(pkgPath, "pkg")

	buf := new(bytes.Buffer)
	kpmcli.logWriter = buf

	// Run the kcl package with vendor mode.
	res, err := kpmcli.Run(
		WithWorkDir(workdir),
		WithVendor(true),
	)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, buf.String(), "")
	assert.Equal(t, res.GetRawYamlResult(), "The_first_kcl_program: Hello World!")
}

func TestRunWithLogger(t *testing.T) {
	pkgPath := getTestDir("test_run_with_logger")
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	logbuf := new(bytes.Buffer)

	_, err = kpmcli.Run(
		WithWorkDir(pkgPath),
		WithLogger(logbuf),
	)

	assert.Equal(t, err, nil)
	assert.Equal(t, logbuf.String(), "Hello, World!\n")
}

func TestVirtualPackageVisiter(t *testing.T) {
	pkgPath := getTestDir("test_virtual_pkg_visitor")
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	pkgSource, err := downloader.NewSourceFromStr(pkgPath)
	assert.Equal(t, err, nil)

	v := newVisitor(*pkgSource, kpmcli)
	err = v.Visit(pkgSource, func(p *pkg.KclPkg) error {
		assert.Contains(t, p.GetPkgName(), "vPkg_")
		_, err = os.Stat(filepath.Join(pkgPath, "kcl.mod"))
		assert.Equal(t, os.IsNotExist(err), true)
		_, err = os.Stat(filepath.Join(pkgPath, "kcl.mod.lock"))
		assert.Equal(t, os.IsNotExist(err), true)
		return nil
	})
	assert.Equal(t, err, nil)
	_, err = os.Stat(filepath.Join(pkgPath, "kcl.mod"))
	assert.Equal(t, os.IsNotExist(err), true)
	_, err = os.Stat(filepath.Join(pkgPath, "kcl.mod.lock"))
	assert.Equal(t, os.IsNotExist(err), true)
}

func testRunWithInsecureSkipTLSverify(t *testing.T) {

	var buf bytes.Buffer

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		buf.WriteString("Called Success\n")
		fmt.Fprintln(w, "Hello, client")
	})

	mux.HandleFunc("/subpath", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from subpath")
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	fmt.Printf("ts.URL: %v\n", ts.URL)
	turl, err := url.Parse(ts.URL)
	assert.Equal(t, err, nil)

	turl.Scheme = "oci"
	turl.Path = filepath.Join(turl.Path, "subpath")
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	_, _ = kpmcli.Run(
		WithRunSourceUrl(turl.String()),
	)

	assert.Equal(t, buf.String(), "")

	kpmcli.SetInsecureSkipTLSverify(true)
	_, _ = kpmcli.Run(
		WithRunSourceUrl(turl.String()),
	)

	assert.Equal(t, buf.String(), "Called Success\n")
}

func testAddDepsWithInsecureSkipTLSverify(t *testing.T) {

	var buf bytes.Buffer

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		buf.WriteString("Called Success\n")
		fmt.Fprintln(w, "Hello, client")
	})

	mux.HandleFunc("/subpath", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from subpath")
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	fmt.Printf("ts.URL: %v\n", ts.URL)
	turl, err := url.Parse(ts.URL)
	assert.Equal(t, err, nil)

	turl.Scheme = "oci"
	turl.Path = filepath.Join(turl.Path, "subpath")

	ociOpts := opt.NewOciOptionsFromUrl(turl)

	addOpts := opt.AddOptions{
		RegistryOpts: opt.RegistryOptions{
			Oci: ociOpts,
		},
	}

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	kpkg := pkg.NewKclPkg(&opt.InitOptions{
		Name: "test",
	})

	_, _ = kpmcli.AddDepWithOpts(
		&kpkg, &addOpts,
	)

	assert.Equal(t, buf.String(), "")

	kpmcli.SetInsecureSkipTLSverify(true)
	_, _ = kpmcli.AddDepWithOpts(
		&kpkg, &addOpts,
	)

	assert.Equal(t, buf.String(), "Called Success\n")
}

func testPushWithInsecureSkipTLSverify(t *testing.T) {
	var buf bytes.Buffer

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		buf.WriteString("Called Success\n")
		fmt.Fprintln(w, "Hello, client")
	})

	mux.HandleFunc("/subpath", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from subpath")
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	fmt.Printf("ts.URL: %v\n", ts.URL)
	turl, err := url.Parse(ts.URL)
	assert.Equal(t, err, nil)

	turl.Scheme = "oci"
	turl.Path = filepath.Join(turl.Path, "subpath")

	ociOpts := opt.NewOciOptionsFromUrl(turl)
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	_ = kpmcli.pushToOci("test", ociOpts)

	assert.Equal(t, buf.String(), "")

	kpmcli.SetInsecureSkipTLSverify(true)
	_ = kpmcli.pushToOci("test", ociOpts)

	assert.Equal(t, buf.String(), "Called Success\n")
}

func TestValidateDependency(t *testing.T) {
	features.Enable(features.SupportModCheck)
	defer features.Disable(features.SupportModCheck)

	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)

	dep1 := pkg.Dependency{
		Name:          "helloworld",
		FullName:      "helloworld_0.1.2",
		Version:       "0.1.2",
		Sum:           "PN0OMEV9M8VGFn1CtA/T3bcgZmMJmOo+RkBrLKIWYeQ=",
		LocalFullPath: "path/to/kcl/package",
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Tag:  "0.1.2",
			},
		},
	}
	err = kpmcli.ValidateDependency(&dep1)
	fmt.Printf("err: %v\n", err)
	assert.Equal(t, err, nil)

	dep2 := pkg.Dependency{
		Name:          "helloworld",
		FullName:      "helloworld_0.1.2",
		Version:       "0.1.2",
		Sum:           "fail-to-validate-dependency",
		LocalFullPath: "path/to/kcl/package",
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
				Tag:  "0.1.2",
			},
		},
	}

	err = kpmcli.ValidateDependency(&dep2)
	assert.Error(t, err)
}
