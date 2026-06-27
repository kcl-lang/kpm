package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	orderedmap "github.com/elliotchance/orderedmap/v2"
	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/utils"
)

const testTomlDir = "test_data_toml"

func TestMarshalTOML(t *testing.T) {
	modfile := ModFile{
		Pkg: Package{
			Name:    "MyKcl",
			Edition: "v0.0.1",
			Version: "v0.0.1",
			Include: []string{"src/", "README.md", "LICENSE"},
			Exclude: []string{"target/", ".git/", "*.log"},
		},
		Dependencies: Dependencies{
			orderedmap.NewOrderedMap[string, Dependency](),
		},
	}

	dep := Dependency{
		Name:     "MyKcl1",
		FullName: "MyKcl1_v0.0.2",
		Source: downloader.Source{
			ModSpec: &downloader.ModSpec{
				Name:    "MyKcl1",
				Version: "0.0.2",
			},
			Git: &downloader.Git{
				Url: "https://github.com/test/MyKcl1.git",
				Tag: "v0.0.2",
			},
		},
	}

	ociDep := Dependency{
		Name:     "MyOciKcl1",
		FullName: "MyOciKcl1_0.0.1",
		Version:  "0.0.1",
		Source: downloader.Source{
			ModSpec: &downloader.ModSpec{
				Name:    "MyOciKcl1",
				Version: "0.0.1",
			},
		},
	}

	modfile.Dependencies.Deps.Set("MyOciKcl1_0.0.1", ociDep)
	modfile.Dependencies.Deps.Set("MyKcl1_v0.0.2", dep)

	got_data := modfile.MarshalTOML()

	expected_data, _ := os.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected.toml"))
	expected_toml := utils.RmNewline(string(expected_data))

	fmt.Printf("expected_toml: '%q'\n", expected_toml)

	fmt.Printf("modfile: '%q'\n", got_data)
	assert.Equal(t, utils.RmNewline(expected_toml), utils.RmNewline(got_data))
}

func TestUnMarshalTOML(t *testing.T) {
	modfile := ModFile{}
	expected_data, _ := os.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected.toml"))

	_ = toml.Unmarshal(expected_data, &modfile)
	fmt.Printf("modfile: %v\n", modfile)

	assert.Equal(t, modfile.Pkg.Name, "MyKcl")
	assert.Equal(t, modfile.Pkg.Edition, "v0.0.1")
	assert.Equal(t, modfile.Pkg.Version, "v0.0.1")
	assert.Equal(t, modfile.Pkg.Include, []string{"src/", "README.md", "LICENSE"})
	assert.Equal(t, modfile.Pkg.Exclude, []string{"target/", ".git/", "*.log"})
	assert.Equal(t, modfile.Dependencies.Deps.Len(), 2)
	assert.NotEqual(t, modfile.Dependencies.Deps.GetOrDefault("MyKcl1", TestPkgDependency), nil)
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("MyKcl1", TestPkgDependency).Name, "MyKcl1")
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("MyKcl1", TestPkgDependency).FullName, "MyKcl1_0.0.2")
	assert.NotEqual(t, modfile.Dependencies.Deps.GetOrDefault("MyKcl1", TestPkgDependency).Source.Git, nil)
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("MyKcl1", TestPkgDependency).Source.Git.Url, "https://github.com/test/MyKcl1.git")
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("MyKcl1", TestPkgDependency).Source.Git.Tag, "v0.0.2")

	assert.NotEqual(t, modfile.Dependencies.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency), nil)
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).Name, "MyOciKcl1")
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).FullName, "MyOciKcl1_0.0.1")
	assert.NotEqual(t, modfile.Dependencies.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).Source.ModSpec, nil)
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).Source.ModSpec.Version, "0.0.1")
}

func TestMarshalLockTOML(t *testing.T) {
	dep := Dependency{
		Name:     "MyKcl1",
		FullName: "MyKcl1_v0.0.2",
		Version:  "v0.0.2",
		Sum:      "hjkasdahjksdasdhjk",
		Source: downloader.Source{
			Git: &downloader.Git{
				Url: "https://github.com/test/MyKcl1.git",
				Tag: "v0.0.2",
			},
		},
	}

	ociDep := Dependency{
		Name:     "MyOciKcl1",
		FullName: "MyOciKcl1_0.0.1",
		Version:  "0.0.1",
		Sum:      "hjkasdahjksdasdhjk",
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  "test_reg",
				Repo: "test_repo",
				Tag:  "0.0.1",
			},
		},
	}

	deps := Dependencies{
		orderedmap.NewOrderedMap[string, Dependency](),
	}

	deps.Deps.Set(dep.Name, dep)
	deps.Deps.Set(ociDep.Name, ociDep)
	tomlStr, _ := deps.MarshalLockTOML()
	expected_data, _ := os.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected_lock.toml"))
	expected_toml := string(expected_data)
	assert.Equal(t, utils.RmNewline(expected_toml), utils.RmNewline(tomlStr))
}

func TestUnmarshalLockTOML(t *testing.T) {
	deps := Dependencies{
		orderedmap.NewOrderedMap[string, Dependency](),
	}

	expected_data, _ := os.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected_lock.toml"))
	expected_toml := string(expected_data)
	_ = deps.UnmarshalLockTOML(expected_toml)

	assert.Equal(t, deps.Deps.Len(), 2)
	assert.NotEqual(t, deps.Deps.GetOrDefault("MyKcl1", TestPkgDependency), nil)
	assert.Equal(t, deps.Deps.GetOrDefault("MyKcl1", TestPkgDependency).Name, "MyKcl1")
	assert.Equal(t, deps.Deps.GetOrDefault("MyKcl1", TestPkgDependency).FullName, "MyKcl1_v0.0.2")
	assert.Equal(t, deps.Deps.GetOrDefault("MyKcl1", TestPkgDependency).Version, "v0.0.2")
	assert.Equal(t, deps.Deps.GetOrDefault("MyKcl1", TestPkgDependency).Sum, "hjkasdahjksdasdhjk")
	assert.NotEqual(t, deps.Deps.GetOrDefault("MyKcl1", TestPkgDependency).Source.Git, nil)
	assert.Equal(t, deps.Deps.GetOrDefault("MyKcl1", TestPkgDependency).Source.Git.Url, "https://github.com/test/MyKcl1.git")
	assert.Equal(t, deps.Deps.GetOrDefault("MyKcl1", TestPkgDependency).Source.Git.Tag, "v0.0.2")

	assert.NotEqual(t, deps.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency), nil)
	assert.Equal(t, deps.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).Name, "MyOciKcl1")
	assert.Equal(t, deps.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).FullName, "MyOciKcl1_0.0.1")
	assert.Equal(t, deps.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).Version, "0.0.1")
	assert.Equal(t, deps.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).Sum, "hjkasdahjksdasdhjk")
	assert.NotEqual(t, deps.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).Source.Oci, nil)
	assert.Equal(t, deps.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).Source.Oci.Reg, "test_reg")
	assert.Equal(t, deps.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).Source.Oci.Repo, "test_repo")
	assert.Equal(t, deps.Deps.GetOrDefault("MyOciKcl1", TestPkgDependency).Source.Oci.Tag, "0.0.1")
}

func TestUnMarshalTOMLWithProfile(t *testing.T) {
	modfile, err := LoadModFile(getTestDir("test_profile"))
	assert.Equal(t, err, nil)
	assert.Equal(t, modfile.Pkg.Name, "kpm")
	assert.Equal(t, modfile.Pkg.Version, "0.0.1")
	assert.Equal(t, modfile.Pkg.Edition, "0.0.1")
	assert.Equal(t, *modfile.Profiles.Entries, []string{"main.k", "xxx/xxx/dir", "test.yaml"})
}

func TestUnMarshalOciUrl(t *testing.T) {
	testDataDir := getTestDir("test_oci_url")

	testCases := []struct {
		Name          string
		DepName       string
		DepFullName   string
		DepVersion    string
		DepSourceReg  string
		DepSourceRepo string
		DepSourceTag  string
	}{
		{"unmarshal_0", "oci_pkg_name", "oci_pkg_name_0.0.1", "0.0.1", "ghcr.io", "test/helloworld", "0.0.1"},
		{"unmarshal_1", "oci_pkg_name", "oci_pkg_name_0.0.1", "0.0.1", "localhost:5002", "test/helloworld", "0.0.1"},
	}

	for _, tc := range testCases {
		modfile, err := LoadModFile(filepath.Join(testDataDir, tc.Name))
		assert.Equal(t, err, nil)
		assert.Equal(t, modfile.Dependencies.Deps.Len(), 1)
		assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("oci_pkg_name", TestPkgDependency).Name, tc.DepName)
		assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("oci_pkg_name", TestPkgDependency).FullName, tc.DepFullName)
		assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("oci_pkg_name", TestPkgDependency).Version, tc.DepVersion)
		assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("oci_pkg_name", TestPkgDependency).Source.Oci.Reg, tc.DepSourceReg)
		assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("oci_pkg_name", TestPkgDependency).Source.Oci.Repo, tc.DepSourceRepo)
		assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("oci_pkg_name", TestPkgDependency).Source.Oci.Tag, tc.DepVersion)
	}
}

func TestMarshalOciUrl(t *testing.T) {
	testDataDir := getTestDir("test_oci_url")

	expectPkgPath := filepath.Join(testDataDir, "marshal_0", "kcl_mod_bk")
	gotPkgPath := filepath.Join(testDataDir, "marshal_0", "kcl_mod_tmp")

	expect, err := LoadModFile(expectPkgPath)
	assert.Equal(t, err, nil)

	err = os.MkdirAll(gotPkgPath, 0755)
	assert.Equal(t, err, nil)
	gotFile, _ := os.Create(filepath.Join(gotPkgPath, "kcl.mod"))

	defer func() {
		err = gotFile.Close()
		assert.Equal(t, err, nil)
		err = os.RemoveAll(gotPkgPath)
		assert.Equal(t, err, nil)
	}()

	modfile := ModFile{
		Pkg: Package{
			Name:    "marshal_0",
			Edition: "v0.12.3",
			Version: "0.0.1",
		},
		Dependencies: Dependencies{
			orderedmap.NewOrderedMap[string, Dependency](),
		},
	}

	ociDep := Dependency{
		Name:     "oci_pkg",
		FullName: "oci_pkg_0.0.1",
		Version:  "0.0.1",
		Source: downloader.Source{
			ModSpec: &downloader.ModSpec{
				Name:    "oci_pkg",
				Version: "0.0.1",
			},
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/oci_pkg",
				Tag:  "0.0.1",
			},
		},
	}

	modfile.Dependencies.Deps.Set("oci_pkg_0.0.1", ociDep)

	got_data := modfile.MarshalTOML()
	_, err = gotFile.WriteString(got_data)
	assert.Equal(t, err, nil)

	got := ModFile{}
	err = got.LoadModFile(filepath.Join(gotPkgPath, "kcl.mod"))
	assert.Equal(t, err, nil)

	assert.Equal(t, expect.Pkg.Name, got.Pkg.Name)
	assert.Equal(t, expect.Pkg.Edition, got.Pkg.Edition)
	assert.Equal(t, expect.Pkg.Version, got.Pkg.Version)
	assert.Equal(t, expect.Dependencies.Deps.Len(), got.Dependencies.Deps.Len())
	assert.Equal(t, expect.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).Name, got.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).Name)
	assert.Equal(t, expect.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).FullName, got.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).FullName)
	assert.Equal(t, expect.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).Source.Oci.Reg, got.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).Source.Oci.Reg)
	assert.Equal(t, expect.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).Source.Oci.Repo, got.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).Source.Oci.Repo)
	assert.Equal(t, expect.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).Source.Oci.Tag, got.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).Source.Oci.Tag)
	assert.Equal(t, expect.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).Source.IntoOciUrl(), got.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).Source.IntoOciUrl())
}

func TestMarshalOciUrlIntoFile(t *testing.T) {
	testDataDir := getTestDir("test_oci_url")

	testCases := []string{"marshal_2"}

	for _, tc := range testCases {
		readKclModPath := filepath.Join(testDataDir, tc)
		modfilePath := filepath.Join(readKclModPath, "kcl.mod")
		expectPath := filepath.Join(readKclModPath, "expect.mod")

		readKclModFile := ModFile{}
		err := readKclModFile.LoadModFile(modfilePath)
		assert.Equal(t, err, nil)
		writeKclModFileContents := readKclModFile.MarshalTOML()
		expectKclModFileContents, err := os.ReadFile(expectPath)
		assert.Equal(t, err, nil)

		assert.Equal(t, utils.RmNewline(string(expectKclModFileContents)), utils.RmNewline(writeKclModFileContents))
	}
}

func TestInitEmptyPkg(t *testing.T) {
	modfile := ModFile{
		Pkg: Package{
			Name:    "MyKcl",
			Edition: "v0.0.1",
			Version: "v0.0.1",
			Include: []string{"src/", "README.md", "LICENSE"},
			Exclude: []string{"target/", ".git/", "*.log"},
		},
		Dependencies: Dependencies{
			orderedmap.NewOrderedMap[string, Dependency](),
		},
	}

	dep := Dependency{
		Name:     "MyKcl1",
		FullName: "MyKcl1_v0.0.2",
		Source: downloader.Source{
			ModSpec: &downloader.ModSpec{
				Name:    "MyKcl1",
				Version: "0.0.2",
			},
			Git: &downloader.Git{
				Url: "https://github.com/test/MyKcl1.git",
				Tag: "v0.0.2",
			},
		},
	}

	ociDep := Dependency{
		Name:     "MyOciKcl1",
		FullName: "MyOciKcl1_0.0.1",
		Version:  "0.0.1",
		Source: downloader.Source{
			ModSpec: &downloader.ModSpec{
				Name:    "MyOciKcl1",
				Version: "0.0.1",
			},
		},
	}

	modfile.Dependencies.Deps.Set("MyOciKcl1_0.0.1", ociDep)
	modfile.Dependencies.Deps.Set("MyKcl1_v0.0.2", dep)

	got_data := modfile.MarshalTOML()

	expected_data, _ := os.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected.toml"))
	expected_toml := string(expected_data)

	expected_toml = strings.ReplaceAll(expected_toml, "\r\n", "\n")
	got_data = strings.ReplaceAll(got_data, "\r\n", "\n")

	expected_toml = strings.TrimSpace(expected_toml)
	got_data = strings.TrimSpace(got_data)

	// Ensure there's no extra newlines between sections
	expected_toml = strings.Join(strings.Fields(expected_toml), "\n")
	got_data = strings.Join(strings.Fields(got_data), "\n")

	fmt.Printf("expected_toml: '%q'\n", expected_toml)

	fmt.Printf("modfile: '%q'\n", got_data)
	assert.Equal(t, expected_toml, got_data)
}

func TestUnMarshalRename(t *testing.T) {
	modfile := ModFile{}
	modfile.LoadModFile(filepath.Join(getTestDir("test_rename_pkg"), "kcl.mod"))
	assert.Equal(t, modfile.Pkg.Name, "rename")
	assert.Equal(t, modfile.Pkg.Version, "0.0.1")
	assert.Equal(t, modfile.Pkg.Edition, "v0.12.3")
	assert.Equal(t, modfile.Dependencies.Deps.Len(), 1)
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("newpkg", TestPkgDependency).Name, "newpkg")
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("newpkg", TestPkgDependency).FullName, "newpkg_0.0.1")
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("newpkg", TestPkgDependency).Version, "0.0.1")
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("newpkg", TestPkgDependency).Source.ModSpec.Name, "subhelloworld")
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("newpkg", TestPkgDependency).Source.ModSpec.Version, "0.0.1")
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("newpkg", TestPkgDependency).Source.ModSpec.Alias, "newpkg")
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("newpkg", TestPkgDependency).Oci.Reg, "ghcr.io")
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("newpkg", TestPkgDependency).Oci.Repo, "kcl-lang/helloworld")
	assert.Equal(t, modfile.Dependencies.Deps.GetOrDefault("newpkg", TestPkgDependency).Oci.Tag, "0.1.4")
}

// TestUnMarshalHostlessOciUrl verifies that a `repo = "..."` entry in kcl.mod
// is parsed into an Oci with RegFromEnv=true, so that even after
// fillDepsInfoWithSettings fills Reg with the default registry, the dep is still
// marshaled back as host-less (`repo = "..."`).
func TestUnMarshalHostlessOciUrl(t *testing.T) {
	testDataDir := getTestDir("test_oci_url")

	modfile, err := LoadModFile(filepath.Join(testDataDir, "unmarshal_hostless"))
	assert.Equal(t, err, nil)
	assert.Equal(t, modfile.Dependencies.Deps.Len(), 1)

	dep := modfile.Dependencies.Deps.GetOrDefault("oci_pkg_name", TestPkgDependency)
	assert.Equal(t, dep.Name, "oci_pkg_name")
	assert.NotNil(t, dep.Source.Oci, "expected Oci source")
	// RegFromEnv=true is preserved even after fillDepsInfoWithSettings fills Reg
	// from the default registry.  The Reg value itself may be non-empty at this
	// point; what matters is that RegFromEnv marks the dep as originally host-less.
	assert.Equal(t, dep.Source.Oci.RegFromEnv, true)
	assert.Equal(t, dep.Source.Oci.Repo, "myorg/kcl-templates/helloworld")
	assert.Equal(t, dep.Source.Oci.Tag, "0.0.1")

	// Even with Reg filled in by settings, marshaling must produce the host-less form.
	marshaled := dep.Source.MarshalTOML()
	assert.True(t, strings.Contains(marshaled, `repo = "myorg/kcl-templates/helloworld"`),
		"marshal output should use repo key, got: %s", marshaled)
	assert.False(t, strings.Contains(marshaled, "oci = "),
		"marshal output must not contain full oci URL for host-less dep, got: %s", marshaled)
}

// TestMarshalHostlessOciUrl verifies that a host-less OCI dependency is written
// back to kcl.mod using `repo = "..."` syntax and round-trips correctly.
func TestMarshalHostlessOciUrl(t *testing.T) {
	testDataDir := getTestDir("test_oci_url")

	expectPkgPath := filepath.Join(testDataDir, "marshal_hostless", "kcl_mod_bk")
	gotPkgPath := filepath.Join(testDataDir, "marshal_hostless", "kcl_mod_tmp")

	expect, err := LoadModFile(expectPkgPath)
	assert.Equal(t, err, nil)

	err = os.MkdirAll(gotPkgPath, 0755)
	assert.Equal(t, err, nil)
	gotFile, _ := os.Create(filepath.Join(gotPkgPath, "kcl.mod"))

	defer func() {
		err = gotFile.Close()
		assert.Equal(t, err, nil)
		err = os.RemoveAll(gotPkgPath)
		assert.Equal(t, err, nil)
	}()

	modfile := ModFile{
		Pkg: Package{
			Name:    "marshal_hostless",
			Edition: "v0.12.3",
			Version: "0.0.1",
		},
		Dependencies: Dependencies{
			orderedmap.NewOrderedMap[string, Dependency](),
		},
	}

	hostlessDep := Dependency{
		Name:     "oci_pkg",
		FullName: "oci_pkg_0.0.1",
		Version:  "0.0.1",
		Source: downloader.Source{
			ModSpec: &downloader.ModSpec{
				Name:    "oci_pkg",
				Version: "0.0.1",
			},
			Oci: &downloader.Oci{
				Repo:       "myorg/kcl-templates/oci_pkg",
				Tag:        "0.0.1",
				RegFromEnv: true,
			},
		},
	}

	modfile.Dependencies.Deps.Set("oci_pkg_0.0.1", hostlessDep)

	got_data := modfile.MarshalTOML()
	_, err = gotFile.WriteString(got_data)
	assert.Equal(t, err, nil)

	got := ModFile{}
	err = got.LoadModFile(filepath.Join(gotPkgPath, "kcl.mod"))
	assert.Equal(t, err, nil)

	assert.Equal(t, expect.Pkg.Name, got.Pkg.Name)
	assert.Equal(t, expect.Pkg.Edition, got.Pkg.Edition)
	assert.Equal(t, expect.Pkg.Version, got.Pkg.Version)
	assert.Equal(t, expect.Dependencies.Deps.Len(), got.Dependencies.Deps.Len())

	gotDep := got.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency)
	assert.NotNil(t, gotDep.Source.Oci, "expected Oci source after round-trip")
	// After LoadModFile → fillDepsInfoWithSettings, Reg may be non-empty.
	// What must be preserved is RegFromEnv=true (dep was originally host-less).
	assert.Equal(t, gotDep.Source.Oci.RegFromEnv, true)
	assert.Equal(t, gotDep.Source.Oci.Repo, "myorg/kcl-templates/oci_pkg")
	assert.Equal(t, gotDep.Source.Oci.Tag, "0.0.1")
	// Re-marshal must be host-less.
	assert.Equal(t, expect.Dependencies.Deps.GetOrDefault("oci_pkg", TestPkgDependency).Source.Oci.RegFromEnv,
		gotDep.Source.Oci.RegFromEnv)
}

// TestMarshalLockTOMLHostless verifies that a host-less OCI dependency is written
// to kcl.mod.lock without the `reg` key (omitted via omitempty), keeping a single
// lock file valid across multiple registry accounts.
func TestMarshalLockTOMLHostless(t *testing.T) {
	hostlessDep := Dependency{
		Name:     "MyHostlessOciKcl",
		FullName: "MyHostlessOciKcl_0.0.1",
		Version:  "0.0.1",
		Sum:      "abc123",
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Repo:       "myorg/kcl-templates/utils",
				Tag:        "0.0.1",
				RegFromEnv: true,
			},
		},
	}

	// Simulate Reg temporarily filled by resolution (should not be persisted)
	hostlessDep.Source.Oci.Reg = "672819064798.dkr.ecr.eu-west-1.amazonaws.com"

	deps := Dependencies{
		orderedmap.NewOrderedMap[string, Dependency](),
	}
	deps.Deps.Set(hostlessDep.Name, hostlessDep)

	tomlStr, err := deps.MarshalLockTOML()
	assert.Equal(t, err, nil)
	assert.False(t, strings.Contains(tomlStr, "reg ="), "lock must not contain 'reg =' for a host-less dependency")
	assert.True(t, strings.Contains(tomlStr, `repo = "myorg/kcl-templates/utils"`), "lock must contain repo")
	assert.True(t, strings.Contains(tomlStr, `oci_tag = "0.0.1"`), "lock must contain oci_tag")
	assert.True(t, strings.Contains(tomlStr, `sum = "abc123"`), "lock must contain sum")
}

// TestUnmarshalLockTOMLHostlessRestoresRegFromEnv verifies that loading a lock
// entry without `reg` sets RegFromEnv=true so subsequent writes remain host-less.
func TestUnmarshalLockTOMLHostlessRestoresRegFromEnv(t *testing.T) {
	lockData := `[dependencies]
  [dependencies.MyHostlessOciKcl]
    name = "MyHostlessOciKcl"
    full_name = "MyHostlessOciKcl_0.0.1"
    version = "0.0.1"
    sum = "abc123"
    repo = "myorg/kcl-templates/utils"
    oci_tag = "0.0.1"
`
	deps := Dependencies{
		orderedmap.NewOrderedMap[string, Dependency](),
	}
	err := deps.UnmarshalLockTOML(lockData)
	assert.Equal(t, err, nil)

	dep := deps.Deps.GetOrDefault("MyHostlessOciKcl", TestPkgDependency)
	assert.NotNil(t, dep.Source.Oci)
	assert.Equal(t, dep.Source.Oci.Reg, "")
	assert.Equal(t, dep.Source.Oci.RegFromEnv, true)
	assert.Equal(t, dep.Source.Oci.Repo, "myorg/kcl-templates/utils")
	assert.Equal(t, dep.Source.Oci.Tag, "0.0.1")
}
