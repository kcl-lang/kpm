package modfile

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
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
			&Git{
				Url: "https://github.com/test/MyKcl1.git",
				Tag: "v0.0.2",
			},
		},
	}

	modfile.Dependencies.Deps["MyKcl1_v0.0.2"] = dep

	expected_data, _ := ioutil.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected.toml"))
	expected_toml := string(expected_data)

	assert.Equal(t, expected_toml, modfile.MarshalTOML())
}

func TestUnMarshalTOML(t *testing.T) {
	modfile := ModFile{}
	expected_data, _ := ioutil.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected.toml"))

	_ = toml.Unmarshal(expected_data, &modfile)
	fmt.Printf("modfile: %v\n", modfile)

	assert.Equal(t, modfile.Pkg.Name, "MyKcl")
	assert.Equal(t, modfile.Pkg.Edition, "v0.0.1")
	assert.Equal(t, modfile.Pkg.Version, "v0.0.1")
	assert.Equal(t, len(modfile.Dependencies.Deps), 1)
	assert.NotEqual(t, modfile.Dependencies.Deps["MyKcl1"], nil)
	assert.Equal(t, modfile.Dependencies.Deps["MyKcl1"].Name, "MyKcl1")
	assert.Equal(t, modfile.Dependencies.Deps["MyKcl1"].FullName, "MyKcl1_v0.0.2")
	assert.NotEqual(t, modfile.Dependencies.Deps["MyKcl1"].Source.Git, nil)
	assert.Equal(t, modfile.Dependencies.Deps["MyKcl1"].Source.Git.Url, "https://github.com/test/MyKcl1.git")
	assert.Equal(t, modfile.Dependencies.Deps["MyKcl1"].Source.Git.Tag, "v0.0.2")
}

func TestMarshalLockToml(t *testing.T) {
	dep := Dependency{
		Name:     "MyKcl1",
		FullName: "MyKcl1_v0.0.2",
		Version:  "v0.0.2",
		Sum:      "hjkasdahjksdasdhjk",
		Source: Source{
			&Git{
				Url: "https://github.com/test/MyKcl1.git",
				Tag: "v0.0.2",
			},
		},
	}

	deps := Dependencies{
		make(map[string]Dependency),
	}

	deps.Deps[dep.Name] = dep
	tomlStr, _ := deps.MarshalLockTOML()
	expected_data, _ := ioutil.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected_lock.toml"))
	expected_toml := string(expected_data)
	assert.Equal(t, expected_toml, tomlStr)
}

func TestUnmarshalLockToml(t *testing.T) {
	deps := Dependencies{
		make(map[string]Dependency),
	}

	expected_data, _ := ioutil.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected_lock.toml"))
	expected_toml := string(expected_data)
	_ = deps.UnmarshalLockTOML(expected_toml)

	assert.Equal(t, len(deps.Deps), 1)
	assert.NotEqual(t, deps.Deps["MyKcl1"], nil)
	assert.Equal(t, deps.Deps["MyKcl1"].Name, "MyKcl1")
	assert.Equal(t, deps.Deps["MyKcl1"].FullName, "MyKcl1_v0.0.2")
	assert.Equal(t, deps.Deps["MyKcl1"].Version, "v0.0.2")
	assert.Equal(t, deps.Deps["MyKcl1"].Sum, "hjkasdahjksdasdhjk")
	assert.NotEqual(t, deps.Deps["MyKcl1"].Source.Git, nil)
	assert.Equal(t, deps.Deps["MyKcl1"].Source.Git.Url, "https://github.com/test/MyKcl1.git")
	assert.Equal(t, deps.Deps["MyKcl1"].Source.Git.Tag, "v0.0.2")
}
