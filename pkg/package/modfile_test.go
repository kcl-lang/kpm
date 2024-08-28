package pkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/runner"
	"kcl-lang.io/kpm/pkg/utils"
)

func TestModFileWithDesc(t *testing.T) {
	testPath := getTestDir("test_mod_with_desc")
	isExist, err := ModFileExists(testPath)
	assert.Equal(t, isExist, true)
	assert.Equal(t, err, nil)
	modFile, err := LoadModFile(testPath)
	assert.Equal(t, modFile.Pkg.Name, "test_mod_with_desc")
	assert.Equal(t, modFile.Pkg.Version, "0.0.1")
	assert.Equal(t, modFile.Pkg.Edition, "0.0.1")
	assert.Equal(t, modFile.Pkg.Description, "This is a test module with a description")
	assert.Equal(t, modFile.Dependencies.Deps.Len(), 0)
	assert.Equal(t, err, nil)
}

func TestDepEquals(t *testing.T) {
	d := Dependency{
		Name:    "test",
		Version: "0.0.1",
	}

	d2 := Dependency{
		Name:    "test",
		Version: "0.0.2",
	}

	assert.Equal(t, d.Equals(d2), false)

	d2.Version = "0.0.1"
	assert.Equal(t, d.Equals(d2), true)

	d2.Name = "test2"
	assert.Equal(t, d.Equals(d2), false)
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
	if err != nil || NewModFile.Pkg.Name != "test_kcl_pkg" || NewModFile.Pkg.Version != "0.0.1" || NewModFile.Pkg.Edition != runner.GetKclVersion() {
		t.Errorf("test 'LoadModFile' failed.")
	}
}

func TestParseOpt(t *testing.T) {
	_, err := ParseOpt(&opt.RegistryOptions{
		Git: &opt.GitOptions{
			Url:    "test.git",
			Branch: "test_branch",
			Commit: "test_commit",
			Tag:    "test_tag",
		},
	})
	assert.Equal(t, err.Error(), "only one of branch, tag or commit is allowed")

	dep, err := ParseOpt(&opt.RegistryOptions{
		Git: &opt.GitOptions{
			Url:    "test.git",
			Branch: "test_branch",
			Commit: "",
			Tag:    "",
		},
	})
	assert.Equal(t, err, nil)
	assert.Equal(t, dep.Name, "test")
	assert.Equal(t, dep.FullName, "test_test_branch")
	assert.Equal(t, dep.Url, "test.git")
	assert.Equal(t, dep.Branch, "test_branch")
	assert.Equal(t, dep.Commit, "")
	assert.Equal(t, dep.Git.Tag, "")

	dep, err = ParseOpt(&opt.RegistryOptions{
		Git: &opt.GitOptions{
			Url:    "test.git",
			Branch: "",
			Commit: "test_commit",
			Tag:    "",
		},
	})
	assert.Equal(t, err, nil)
	assert.Equal(t, dep.Name, "test")
	assert.Equal(t, dep.FullName, "test_test_commit")
	assert.Equal(t, dep.Url, "test.git")
	assert.Equal(t, dep.Branch, "")
	assert.Equal(t, dep.Commit, "test_commit")
	assert.Equal(t, dep.Git.Tag, "")

	dep, err = ParseOpt(&opt.RegistryOptions{
		Git: &opt.GitOptions{
			Url:    "test.git",
			Branch: "",
			Commit: "",
			Tag:    "test_tag",
		},
	})
	assert.Equal(t, err, nil)
	assert.Equal(t, dep.Name, "test")
	assert.Equal(t, dep.FullName, "test_test_tag")
	assert.Equal(t, dep.Url, "test.git")
	assert.Equal(t, dep.Branch, "")
	assert.Equal(t, dep.Commit, "")
	assert.Equal(t, dep.Git.Tag, "test_tag")

	dep, _ = ParseOpt(&opt.RegistryOptions{
		Git: &opt.GitOptions{
			Url:     "test.git",
			Branch:  "",
			Commit:  "",
			Tag:     "test_tag",
			Package: "k8s",
		},
	})

	assert.Equal(t, dep.Name, "k8s")
	assert.Equal(t, dep.FullName, "test_test_tag")
	assert.Equal(t, dep.Git.Package, "k8s")
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

	assert.Equal(t, modFile.Dependencies.Deps.Len(), 3)
	assert.Equal(t, modFile.Dependencies.Deps.GetOrDefault("name", TestPkgDependency).Name, "name")
	assert.Equal(t, modFile.Dependencies.Deps.GetOrDefault("name", TestPkgDependency).Source.Git.Url, "test_url")
	assert.Equal(t, modFile.Dependencies.Deps.GetOrDefault("name", TestPkgDependency).Source.Git.Tag, "test_tag")
	assert.Equal(t, modFile.Dependencies.Deps.GetOrDefault("name", TestPkgDependency).FullName, "name_test_tag")

	assert.Equal(t, modFile.Dependencies.Deps.GetOrDefault("oci_name", TestPkgDependency).Name, "oci_name")
	assert.Equal(t, modFile.Dependencies.Deps.GetOrDefault("oci_name", TestPkgDependency).Version, "oci_tag")
	assert.Equal(t, modFile.Dependencies.Deps.GetOrDefault("oci_name", TestPkgDependency).Source.Registry.Tag, "oci_tag")
	assert.Equal(t, err, nil)

	assert.Equal(t, modFile.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).Name, "helloworld")
	assert.Equal(t, modFile.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).Version, "0.1.2")
	assert.Equal(t, modFile.Dependencies.Deps.GetOrDefault("helloworld", TestPkgDependency).Source.Oci.Tag, "0.1.2")
	assert.Equal(t, err, nil)
}

func TestLoadLockDeps(t *testing.T) {
	testPath := getTestDir("load_lock_file")
	deps, err := LoadLockDeps(testPath)

	assert.Equal(t, deps.Deps.Len(), 2)
	assert.Equal(t, deps.Deps.GetOrDefault("name", TestPkgDependency).Name, "name")
	assert.Equal(t, deps.Deps.GetOrDefault("name", TestPkgDependency).Version, "test_version")
	assert.Equal(t, deps.Deps.GetOrDefault("name", TestPkgDependency).Sum, "test_sum")
	assert.Equal(t, deps.Deps.GetOrDefault("name", TestPkgDependency).Source.Git.Url, "test_url")
	assert.Equal(t, deps.Deps.GetOrDefault("name", TestPkgDependency).Source.Git.Tag, "test_tag")
	assert.Equal(t, deps.Deps.GetOrDefault("name", TestPkgDependency).FullName, "test_version")

	assert.Equal(t, deps.Deps.GetOrDefault("oci_name", TestPkgDependency).Name, "oci_name")
	assert.Equal(t, deps.Deps.GetOrDefault("oci_name", TestPkgDependency).Version, "test_version")
	assert.Equal(t, deps.Deps.GetOrDefault("oci_name", TestPkgDependency).Sum, "test_sum")
	assert.Equal(t, deps.Deps.GetOrDefault("oci_name", TestPkgDependency).Source.Oci.Reg, "test_reg")
	assert.Equal(t, deps.Deps.GetOrDefault("oci_name", TestPkgDependency).Source.Oci.Repo, "test_repo")
	assert.Equal(t, deps.Deps.GetOrDefault("oci_name", TestPkgDependency).Source.Oci.Tag, "test_oci_tag")
	assert.Equal(t, deps.Deps.GetOrDefault("oci_name", TestPkgDependency).FullName, "test_version")
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

	err := mfile.StoreModFile()
	assert.Equal(t, err, nil)

	expect, _ := os.ReadFile(filepath.Join(testPath, "expected.toml"))
	got, _ := os.ReadFile(filepath.Join(testPath, "kcl.mod"))
	assert.Equal(t, utils.RmNewline(string(got)), utils.RmNewline(string(expect)))
}

func TestGetFilePath(t *testing.T) {
	testPath := getTestDir("store_mod_file")
	mfile := ModFile{
		HomePath: testPath,
	}
	assert.Equal(t, mfile.GetModFilePath(), filepath.Join(testPath, MOD_FILE))
	assert.Equal(t, mfile.GetModLockFilePath(), filepath.Join(testPath, MOD_LOCK_FILE))
}

func TestGenSource(t *testing.T) {
	src, err := GenSource("git", "https://github.com/kcl-lang/kcl", "0.8.7")
	assert.Equal(t, err, nil)
	assert.Equal(t, src.Git.Url, "https://github.com/kcl-lang/kcl")
	assert.Equal(t, src.Git.Tag, "0.8.7")

	src, err = GenSource("oci", "oci://ghcr.io/kcl-lang/k8s", "1.24")
	assert.Equal(t, err, nil)
	assert.Equal(t, src.Oci.Reg, "ghcr.io")
	assert.Equal(t, src.Oci.Repo, "kcl-lang/k8s")
	assert.Equal(t, src.Oci.Tag, "1.24")
}
