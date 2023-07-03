package modfile

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/utils"
)

const testTomlDir = "test_data_toml"

func TestMarshalTOML(t *testing.T) {
	modfile := ModFile{
		Pkg: Package{
			Name:    "MyKcl",
			Edition: "v0.0.1",
			Version: "v0.0.1",
		},
		Dependencies: Dependencies{
			make(map[string]Dependency),
		},
	}

	dep := Dependency{
		Name:     "MyKcl1",
		FullName: "MyKcl1_v0.0.2",
		Source: Source{
			Git: &Git{
				Url: "https://github.com/test/MyKcl1.git",
				Tag: "v0.0.2",
			},
		},
	}

	ociDep := Dependency{
		Name:     "MyOciKcl1",
		FullName: "MyOciKcl1_0.0.1",
		Version:  "0.0.1",
		Source: Source{
			Oci: &Oci{
				Tag: "0.0.1",
			},
		},
	}

	modfile.Dependencies.Deps["MyOciKcl1_0.0.1"] = ociDep
	modfile.Dependencies.Deps["MyKcl1_v0.0.2"] = dep

	got_data := modfile.MarshalTOML()

	expected_data, _ := os.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected.toml"))
	expected_toml := string(expected_data)

	reversed_expected_data, _ := os.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected_reversed.toml"))
	reversed_expected_toml := string(reversed_expected_data)
	fmt.Printf("expected_toml: '%q'\n", expected_toml)
	fmt.Printf("reversed_expected_toml: '%q'\n", reversed_expected_toml)
	fmt.Printf("modfile: '%q'\n", got_data)
	fmt.Printf("expected_toml == got_data: '%t'\n", expected_toml == got_data)
	fmt.Printf("reversed_expected_toml == got_data: '%t'\n", reversed_expected_toml == got_data)
	assert.Equal(t, (utils.RmNewline(expected_toml) == utils.RmNewline(got_data)) ||
		(utils.RmNewline(reversed_expected_toml) == utils.RmNewline(got_data)), true)
}

func TestUnMarshalTOML(t *testing.T) {
	modfile := ModFile{}
	expected_data, _ := os.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected.toml"))

	_ = toml.Unmarshal(expected_data, &modfile)
	fmt.Printf("modfile: %v\n", modfile)

	assert.Equal(t, modfile.Pkg.Name, "MyKcl")
	assert.Equal(t, modfile.Pkg.Edition, "v0.0.1")
	assert.Equal(t, modfile.Pkg.Version, "v0.0.1")
	assert.Equal(t, len(modfile.Dependencies.Deps), 2)
	assert.NotEqual(t, modfile.Dependencies.Deps["MyKcl1"], nil)
	assert.Equal(t, modfile.Dependencies.Deps["MyKcl1"].Name, "MyKcl1")
	assert.Equal(t, modfile.Dependencies.Deps["MyKcl1"].FullName, "MyKcl1_v0.0.2")
	assert.NotEqual(t, modfile.Dependencies.Deps["MyKcl1"].Source.Git, nil)
	assert.Equal(t, modfile.Dependencies.Deps["MyKcl1"].Source.Git.Url, "https://github.com/test/MyKcl1.git")
	assert.Equal(t, modfile.Dependencies.Deps["MyKcl1"].Source.Git.Tag, "v0.0.2")

	assert.NotEqual(t, modfile.Dependencies.Deps["MyOciKcl1"], nil)
	assert.Equal(t, modfile.Dependencies.Deps["MyOciKcl1"].Name, "MyOciKcl1")
	assert.Equal(t, modfile.Dependencies.Deps["MyOciKcl1"].FullName, "MyOciKcl1_0.0.1")
	assert.NotEqual(t, modfile.Dependencies.Deps["MyOciKcl1"].Source.Oci, nil)
	assert.Equal(t, modfile.Dependencies.Deps["MyOciKcl1"].Source.Oci.Tag, "0.0.1")
}

func TestMarshalLockToml(t *testing.T) {
	dep := Dependency{
		Name:     "MyKcl1",
		FullName: "MyKcl1_v0.0.2",
		Version:  "v0.0.2",
		Sum:      "hjkasdahjksdasdhjk",
		Source: Source{
			Git: &Git{
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
		Source: Source{
			Oci: &Oci{
				Reg:  "test_reg",
				Repo: "test_repo",
				Tag:  "0.0.1",
			},
		},
	}

	deps := Dependencies{
		make(map[string]Dependency),
	}

	deps.Deps[dep.Name] = dep
	deps.Deps[ociDep.Name] = ociDep
	tomlStr, _ := deps.MarshalLockTOML()
	expected_data, _ := os.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected_lock.toml"))
	expected_toml := string(expected_data)
	assert.Equal(t, utils.RmNewline(expected_toml), utils.RmNewline(tomlStr))
}

func TestUnmarshalLockToml(t *testing.T) {
	deps := Dependencies{
		make(map[string]Dependency),
	}

	expected_data, _ := os.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected_lock.toml"))
	expected_toml := string(expected_data)
	_ = deps.UnmarshalLockTOML(expected_toml)

	assert.Equal(t, len(deps.Deps), 2)
	assert.NotEqual(t, deps.Deps["MyKcl1"], nil)
	assert.Equal(t, deps.Deps["MyKcl1"].Name, "MyKcl1")
	assert.Equal(t, deps.Deps["MyKcl1"].FullName, "MyKcl1_v0.0.2")
	assert.Equal(t, deps.Deps["MyKcl1"].Version, "v0.0.2")
	assert.Equal(t, deps.Deps["MyKcl1"].Sum, "hjkasdahjksdasdhjk")
	assert.NotEqual(t, deps.Deps["MyKcl1"].Source.Git, nil)
	assert.Equal(t, deps.Deps["MyKcl1"].Source.Git.Url, "https://github.com/test/MyKcl1.git")
	assert.Equal(t, deps.Deps["MyKcl1"].Source.Git.Tag, "v0.0.2")

	assert.NotEqual(t, deps.Deps["MyOciKcl1"], nil)
	assert.Equal(t, deps.Deps["MyOciKcl1"].Name, "MyOciKcl1")
	assert.Equal(t, deps.Deps["MyOciKcl1"].FullName, "MyOciKcl1_0.0.1")
	assert.Equal(t, deps.Deps["MyOciKcl1"].Version, "0.0.1")
	assert.Equal(t, deps.Deps["MyOciKcl1"].Sum, "hjkasdahjksdasdhjk")
	assert.NotEqual(t, deps.Deps["MyOciKcl1"].Source.Oci, nil)
	assert.Equal(t, deps.Deps["MyOciKcl1"].Source.Oci.Reg, "test_reg")
	assert.Equal(t, deps.Deps["MyOciKcl1"].Source.Oci.Repo, "test_repo")
	assert.Equal(t, deps.Deps["MyOciKcl1"].Source.Oci.Tag, "0.0.1")
}

func TestUnMarshalTOMLWithProfile(t *testing.T) {
	modfile, err := LoadModFile(getTestDir("test_profile"))
	assert.Equal(t, err, nil)
	assert.Equal(t, modfile.Pkg.Name, "kpm")
	assert.Equal(t, modfile.Pkg.Version, "0.0.1")
	assert.Equal(t, modfile.Pkg.Edition, "0.0.1")
	assert.Equal(t, modfile.Profiles.Entries, []string{"main.k", "xxx/xxx/dir", "test.yaml"})
}
