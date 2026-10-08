// Copyright 2026 The KCL Authors. All rights reserved.

package pkg

import (
	"os"
	"path/filepath"
	"testing"

	orderedmap "github.com/elliotchance/orderedmap/v2"
	"github.com/stretchr/testify/assert"
)

// TestModFileDevDependenciesRoundTrip ensures [dev_dependencies] is
// preserved when a kcl.mod file is parsed and re-serialized.
func TestModFileDevDependenciesRoundTrip(t *testing.T) {
	dir := t.TempDir()

	modPath := filepath.Join(dir, MOD_FILE)
	mod := `
[package]
name = "my-pkg"
version = "0.0.1"
edition = "0.0.1"

[dependencies]
helper = { oci = "oci://ghcr.io/kcl-lang/helper", tag = "0.1.0" }

[dev_dependencies]
dev_helper = { oci = "oci://ghcr.io/kcl-lang/dev_helper", tag = "0.2.0" }

[profile]
entries = ["main.k"]
`
	assert.NoError(t, os.WriteFile(modPath, []byte(mod), 0644))

	mf := new(ModFile)
	assert.NoError(t, mf.LoadModFile(modPath))

	// Regular dep loaded.
	d, ok := mf.Dependencies.Deps.Get("helper")
	assert.True(t, ok)
	assert.Equal(t, "helper", d.Name)

	// Dev dep loaded.
	td, ok := mf.DevDependencies.Deps.Get("dev_helper")
	assert.True(t, ok)
	assert.Equal(t, "dev_helper", td.Name)

	// Re-marshal should include the [dev_dependencies] section.
	out := mf.MarshalTOML()
	assert.Contains(t, out, "[dev_dependencies]")
	assert.Contains(t, out, "dev_helper")
}

// TestModFileAllDeps verifies the merge logic: regular wins on conflict,
// and nil sub-maps do not panic.
func TestModFileAllDeps(t *testing.T) {
	mf := &ModFile{
		HomePath: "/tmp/x",
	}
	mf.Dependencies.Deps = orderedmap.NewOrderedMap[string, Dependency]()
	mf.DevDependencies.Deps = orderedmap.NewOrderedMap[string, Dependency]()

	mf.Dependencies.Deps.Set("regular", Dependency{Name: "regular", Version: "1.0.0"})
	mf.Dependencies.Deps.Set("shared", Dependency{Name: "shared", Version: "1.0.0"})
	mf.DevDependencies.Deps.Set("dev_only", Dependency{Name: "dev_only", Version: "0.1.0"})
	mf.DevDependencies.Deps.Set("shared", Dependency{Name: "shared", Version: "0.5.0"})

	merged := mf.AllDeps()
	assert.Equal(t, 3, merged.Len())

	// Regular wins for `shared`.
	shared, ok := merged.Get("shared")
	assert.True(t, ok)
	assert.Equal(t, "1.0.0", shared.Version)

	testOnly, ok := merged.Get("dev_only")
	assert.True(t, ok)
	assert.Equal(t, "0.1.0", testOnly.Version)
}

// TestModFileAllDepsNilSafe verifies the merge handles nil sub-maps without
// panicking — the case for ModFile built via composite literals in tests.
func TestModFileAllDepsNilSafe(t *testing.T) {
	mf := &ModFile{HomePath: "/tmp/x"}
	merged := mf.AllDeps()
	assert.NotNil(t, merged)
	assert.Equal(t, 0, merged.Len())
}
