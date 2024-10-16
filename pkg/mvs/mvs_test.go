package mvs

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/module"
	"kcl-lang.io/kpm/pkg/3rdparty/mvs"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/test"
	"kcl-lang.io/kpm/pkg/utils"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func testMax(t *testing.T) {
	reqs := ReqsGraph{}
	assert.Equal(t, reqs.Max("", "1.0.0", "2.0.0"), "2.0.0")
	assert.Equal(t, reqs.Max("", "1.2", "2.0"), "2.0")
	assert.Equal(t, reqs.Max("", "2.5.0", "2.6"), "2.6")
	assert.Equal(t, reqs.Max("", "2.0.0", "v3.0"), "v3.0")
}

func testRequired(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_with_internal_deps"), "aaa")
	assert.Equal(t, utils.DirExists(filepath.Join(pkg_path, "kcl.mod")), true)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkg_path)
	assert.Equal(t, err, nil)

	_, depGraph, err := kpmcli.InitGraphAndDownloadDeps(kclPkg)
	assert.Equal(t, err, nil)

	reqs := ReqsGraph{
		depGraph,
		kpmcli,
		kclPkg,
	}

	req, err := reqs.Required(module.Version{Path: "aaa", Version: "0.0.1"})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(req), 2)

	expectedReqs := []module.Version{
		{Path: "bbb", Version: "0.0.1"},
		{Path: "ccc", Version: "0.0.1"},
	}
	sort.Slice(req, func(i, j int) bool {
		return req[i].Path < req[j].Path
	})
	assert.Equal(t, req, expectedReqs)
}

func testUpgrade(t *testing.T) {
	pkg_path := getTestDir("test_with_external_deps")
	assert.Equal(t, utils.DirExists(filepath.Join(pkg_path, "kcl.mod")), true)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkg_path)
	assert.Equal(t, err, nil)

	_, depGraph, err := kpmcli.InitGraphAndDownloadDeps(kclPkg)
	assert.Equal(t, err, nil)

	reqs := ReqsGraph{
		depGraph,
		kpmcli,
		kclPkg,
	}

	target := module.Version{Path: kclPkg.GetPkgName(), Version: kclPkg.GetPkgVersion()}
	upgradeList := []module.Version{
		{Path: "argo-cd-order", Version: "0.2.0"},
		{Path: "helloworld", Version: "0.1.1"},
	}
	upgrade, err := mvs.Upgrade(target, reqs, upgradeList...)
	assert.Equal(t, err, nil)

	expectedReqs := []module.Version{
		{Path: "test_with_external_deps", Version: "0.0.1"},
		{Path: "argo-cd-order", Version: "0.2.0"},
		{Path: "helloworld", Version: "0.1.1"},
		{Path: "json_merge_patch", Version: "0.1.0"},
		{Path: "k8s", Version: "1.29"},
		{Path: "podinfo", Version: "0.1.1"},
	}
	assert.Equal(t, upgrade, expectedReqs)
}

func testUpgradeToLatest(t *testing.T) {
	pkg_path := getTestDir("test_with_external_deps")
	assert.Equal(t, utils.DirExists(filepath.Join(pkg_path, "kcl.mod")), true)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkg_path)
	assert.Equal(t, err, nil)

	_, depGraph, err := kpmcli.InitGraphAndDownloadDeps(kclPkg)
	assert.Equal(t, err, nil)

	reqs := ReqsGraph{
		depGraph,
		kpmcli,
		kclPkg,
	}

	upgrade, err := reqs.Upgrade(module.Version{Path: "k8s", Version: "1.27"})
	assert.Equal(t, err, nil)
	assert.Equal(t, upgrade, module.Version{Path: "k8s", Version: "1.31.2"})
}

func testUpgradeAllToLatest(t *testing.T) {
	pkg_path := getTestDir("test_with_external_deps")
	assert.Equal(t, utils.DirExists(filepath.Join(pkg_path, "kcl.mod")), true)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkg_path)
	assert.Equal(t, err, nil)

	_, depGraph, err := kpmcli.InitGraphAndDownloadDeps(kclPkg)
	assert.Equal(t, err, nil)

	reqs := ReqsGraph{
		depGraph,
		kpmcli,
		kclPkg,
	}

	target := module.Version{Path: kclPkg.GetPkgName(), Version: kclPkg.GetPkgVersion()}

	upgrade, err := mvs.UpgradeAll(target, reqs)
	assert.Equal(t, err, nil)

	expectedReqs := []module.Version{
		{Path: "test_with_external_deps", Version: "0.0.1"},
		{Path: "argo-cd-order", Version: "0.2.0"},
		{Path: "helloworld", Version: "0.1.3"},
		{Path: "json_merge_patch", Version: "0.1.1"},
		{Path: "k8s", Version: "1.31.2"},
		{Path: "podinfo", Version: "0.2.1"},
	}
	assert.Equal(t, upgrade, expectedReqs)
}

func testPrevious(t *testing.T) {
	pkg_path := getTestDir("test_with_external_deps")
	assert.Equal(t, utils.DirExists(filepath.Join(pkg_path, "kcl.mod")), true)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkg_path)
	assert.Equal(t, err, nil)

	_, depGraph, err := kpmcli.InitGraphAndDownloadDeps(kclPkg)
	assert.Equal(t, err, nil)

	reqs := ReqsGraph{
		depGraph,
		kpmcli,
		kclPkg,
	}

	downgrade, err := reqs.Previous(module.Version{Path: "k8s", Version: "1.27"})
	assert.Equal(t, err, nil)
	assert.Equal(t, downgrade, module.Version{Path: "k8s", Version: "1.14"})
}

func testUpgradePreviousOfLocalDependency(t *testing.T) {
	pkg_path := filepath.Join(getTestDir("test_with_internal_deps"), "aaa")
	assert.Equal(t, utils.DirExists(filepath.Join(pkg_path, "kcl.mod")), true)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkg_path)
	assert.Equal(t, err, nil)

	_, depGraph, err := kpmcli.InitGraphAndDownloadDeps(kclPkg)
	assert.Equal(t, err, nil)

	reqs := ReqsGraph{
		depGraph,
		kpmcli,
		kclPkg,
	}

	upgrade, err := reqs.Upgrade(module.Version{Path: "bbb", Version: "0.0.1"})
	assert.Equal(t, err, nil)
	assert.Equal(t, upgrade, module.Version{Path: "bbb", Version: "0.0.1"})

	downgrade, err := reqs.Previous(module.Version{Path: "bbb", Version: "0.0.1"})
	assert.Equal(t, err, nil)
	assert.Equal(t, downgrade, module.Version{Path: "bbb", Version: "0.0.1"})
}

func testDowngrade(t *testing.T) {
	pkg_path := getTestDir("test_with_external_deps")
	assert.Equal(t, utils.DirExists(filepath.Join(pkg_path, "kcl.mod")), true)
	kpmcli, err := client.NewKpmClient()
	assert.Equal(t, err, nil)
	kclPkg, err := kpmcli.LoadPkgFromPath(pkg_path)
	assert.Equal(t, err, nil)

	_, depGraph, err := kpmcli.InitGraphAndDownloadDeps(kclPkg)
	assert.Equal(t, err, nil)

	reqs := ReqsGraph{
		depGraph,
		kpmcli,
		kclPkg,
	}

	target := module.Version{Path: kclPkg.GetPkgName(), Version: kclPkg.GetPkgVersion()}
	downgradeList := []module.Version{
		{Path: "k8s", Version: "1.17"},
	}
	downgrade, err := mvs.Downgrade(target, reqs, downgradeList...)
	assert.Equal(t, err, nil)

	expectedReqs := []module.Version{
		{Path: "test_with_external_deps", Version: "0.0.1"},
		{Path: "argo-cd-order", Version: "0.1.2"},
		{Path: "helloworld", Version: "0.1.0"},
		{Path: "json_merge_patch", Version: "0.1.0"},
		{Path: "k8s", Version: "1.17"},
	}
	assert.Equal(t, downgrade, expectedReqs)
}

func TestMvsWithGloablLock(t *testing.T) {
	test.RunTestWithGlobalLock(t, "TestMax", testMax)
	test.RunTestWithGlobalLock(t, "TestRequired", testRequired)
	test.RunTestWithGlobalLock(t, "TestUpgrade", testUpgrade)
	test.RunTestWithGlobalLock(t, "TestUpgradeToLatest", testUpgradeToLatest)
	test.RunTestWithGlobalLock(t, "TestUpgradeAllToLatest", testUpgradeAllToLatest)
	test.RunTestWithGlobalLock(t, "TestPrevious", testPrevious)
	test.RunTestWithGlobalLock(t, "TestUpgradePreviousOfLocalDependency", testUpgradePreviousOfLocalDependency)
	test.RunTestWithGlobalLock(t, "TestDowngrade", testDowngrade)
}
