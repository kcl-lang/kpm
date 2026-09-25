// Copyright 2026 The KCL Authors. All rights reserved.

package pkg

import (
	"strings"
	"testing"

	orderedmap "github.com/elliotchance/orderedmap/v2"
	"github.com/stretchr/testify/assert"
)

// TestUnmarshalLockTOMLWithDevDeps verifies the lockfile loader is able to
// recover a separate [dev_dependencies] table next to the regular table and
// route entries into the second Dependencies struct.
func TestUnmarshalLockTOMLWithDevDeps(t *testing.T) {
	lockData := `
[dependencies.helper]
name = "helper"
full_name = "helper_0.1.0"
version = "0.1.0"
sum = ""
url = "https://example.com"

[dev_dependencies.dev_helper]
name = "dev_helper"
full_name = "dev_helper_0.2.0"
version = "0.2.0"
sum = ""
url = "https://example.com"
`
	regular := &Dependencies{Deps: orderedmap.NewOrderedMap[string, Dependency]()}
	test := &Dependencies{Deps: orderedmap.NewOrderedMap[string, Dependency]()}
	assert.NoError(t, UnmarshalLockTOML(lockData, regular, test))

	assert.Equal(t, 1, regular.Deps.Len())
	assert.Equal(t, 1, test.Deps.Len())

	helper, ok := regular.Deps.Get("helper")
	assert.True(t, ok)
	assert.Equal(t, "helper", helper.Name)

	testHelper, ok := test.Deps.Get("dev_helper")
	assert.True(t, ok)
	assert.Equal(t, "0.2.0", testHelper.Version)
}

// TestUnmarshalLockTOMLNilTest ensures the loader is happy when the caller
// passes nil for the dev deps — backward-compatible single-section parsing.
func TestUnmarshalLockTOMLNilTest(t *testing.T) {
	lockData := `
[dependencies.helper]
name = "helper"
full_name = "helper_0.1.0"
version = "0.1.0"
sum = ""
url = ""
[dev_dependencies.dev_helper]
name = "dev_helper"
full_name = "dev_helper_0.2.0"
version = "0.2.0"
sum = ""
url = ""
`
	regular := &Dependencies{Deps: orderedmap.NewOrderedMap[string, Dependency]()}
	assert.NoError(t, UnmarshalLockTOML(lockData, regular, nil))
	assert.Equal(t, 1, regular.Deps.Len())
}

// TestMarshalLockFileProducesBothSections ensures the full kcl.mod.lock file
// contains both tables when dev deps are present.
func TestMarshalLockFileProducesBothSections(t *testing.T) {
	kpkg := &KclPkg{
		HomePath: "/tmp/x",
		Dependencies: Dependencies{
			Deps: orderedmap.NewOrderedMap[string, Dependency](),
		},
		DevDependencies: Dependencies{
			Deps: orderedmap.NewOrderedMap[string, Dependency](),
		},
	}
	kpkg.Dependencies.Deps.Set("helper", Dependency{
		Name: "helper", Version: "0.1.0", FullName: "helper_0.1.0",
	})
	kpkg.DevDependencies.Deps.Set("dev_helper", Dependency{
		Name: "dev_helper", Version: "0.2.0", FullName: "dev_helper_0.2.0",
	})

	out, err := kpkg.MarshalLockFile()
	assert.NoError(t, err)
	assert.True(t, strings.Contains(out, "[dependencies]"))
	assert.True(t, strings.Contains(out, "[dev_dependencies]"))
	assert.True(t, strings.Contains(out, "dev_helper"))
}
