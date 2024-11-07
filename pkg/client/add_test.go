package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/utils"
)

func testAddWithModSpec(t *testing.T) {
	tests := []struct {
		name       string
		pkgSubPath string
		sourceUrl  string
	}{
		{
			name:       "TestAddOciWithModSpec",
			pkgSubPath: "oci",
			sourceUrl:  "oci://ghcr.io/kcl-lang/helloworld?tag=0.1.4&mod=subhelloworld:0.0.1",
		},
		{
			name:       "TestAddGitWithModSpec",
			pkgSubPath: "git",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git?commit=8308200&mod=cc:0.0.1",
		},
		{
			name:       "TestAddLocalWithModSpec",
			pkgSubPath: filepath.Join("local", "pkg"),
			sourceUrl:  "../dep?mod=sub:0.0.1",
		},
		{
			name:       "TestAddOciWithEmptyVersion",
			pkgSubPath: "empty_version",
			sourceUrl:  "oci://ghcr.io/kcl-lang/helloworld?tag=0.1.4&mod=subhelloworld",
		},
		{
			name:       "TestAddOciWithNoSpec",
			pkgSubPath: "no_spec",
			sourceUrl:  "oci://ghcr.io/kcl-lang/helloworld?tag=0.1.4",
		},
		{
			name:       "TestAddOciWithNoTag",
			pkgSubPath: "no_oci_ref",
			sourceUrl:  "oci://ghcr.io/kcl-lang/helloworld",
		},
		{
			name:       "TestAddGitWithNoTag",
			pkgSubPath: "no_git_ref",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := getTestDir("add_with_mod_spec")
			pkgPath := filepath.Join(testDir, tt.pkgSubPath)

			modbkPath := filepath.Join(pkgPath, "kcl.mod.bk")
			modPath := filepath.Join(pkgPath, "kcl.mod")
			modExpect := filepath.Join(pkgPath, "kcl.mod.expect")
			lockbkPath := filepath.Join(pkgPath, "kcl.mod.lock.bk")
			lockPath := filepath.Join(pkgPath, "kcl.mod.lock")
			lockExpect := filepath.Join(pkgPath, "kcl.mod.lock.expect")

			err := copy.Copy(modbkPath, modPath)
			if err != nil {
				t.Fatal(err)
			}

			err = copy.Copy(lockbkPath, lockPath)
			if err != nil {
				t.Fatal(err)
			}

			defer func() {
				// remove the copied files
				err := os.RemoveAll(modPath)
				if err != nil {
					t.Fatal(err)
				}
				err = os.RemoveAll(lockPath)
				if err != nil {
					t.Fatal(err)
				}
			}()

			kpmcli, err := NewKpmClient()
			if err != nil {
				t.Fatal(err)
			}

			kpkg, err := pkg.LoadKclPkgWithOpts(
				pkg.WithPath(pkgPath),
				pkg.WithSettings(kpmcli.GetSettings()),
			)

			if err != nil {
				t.Fatal(err)
			}

			err = kpmcli.Add(
				WithAddKclPkg(kpkg),
				WithAddSourceUrl(tt.sourceUrl),
			)

			if err != nil {
				t.Fatal(err)
			}

			expectedMod, err := os.ReadFile(modExpect)
			if err != nil {
				t.Fatal(err)
			}
			gotMod, err := os.ReadFile(modPath)
			if err != nil {
				t.Fatal(err)
			}

			expectedLock, err := os.ReadFile(lockExpect)
			if err != nil {
				t.Fatal(err)
			}

			gotLock, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, utils.RmNewline(string(expectedMod)), utils.RmNewline(string(gotMod)))
			assert.Equal(t, utils.RmNewline(string(expectedLock)), utils.RmNewline(string(gotLock)))
		})
	}
}

func TestAddRenameWithModSpec(t *testing.T) {
	testDir := getTestDir("add_with_mod_spec")
	pkgPath := filepath.Join(testDir, "rename")

	modbkPath := filepath.Join(pkgPath, "kcl.mod.bk")
	modPath := filepath.Join(pkgPath, "kcl.mod")
	modExpect := filepath.Join(pkgPath, "kcl.mod.expect")
	lockbkPath := filepath.Join(pkgPath, "kcl.mod.lock.bk")
	lockPath := filepath.Join(pkgPath, "kcl.mod.lock")
	lockExpect := filepath.Join(pkgPath, "kcl.mod.lock.expect")

	err := copy.Copy(modbkPath, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = copy.Copy(lockbkPath, lockPath)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		// remove the copied files
		err := os.RemoveAll(modPath)
		if err != nil {
			t.Fatal(err)
		}
		err = os.RemoveAll(lockPath)
		if err != nil {
			t.Fatal(err)
		}
	}()

	kpmcli, err := NewKpmClient()
	if err != nil {
		t.Fatal(err)
	}

	kpkg, err := pkg.LoadKclPkgWithOpts(
		pkg.WithPath(pkgPath),
		pkg.WithSettings(kpmcli.GetSettings()),
	)

	if err != nil {
		t.Fatal(err)
	}

	err = kpmcli.Add(
		WithAddKclPkg(kpkg),
		WithAddSourceUrl("oci://ghcr.io/kcl-lang/helloworld?tag=0.1.4&mod=subhelloworld:0.0.1"),
		WithAddPkgNameAlias("newpkg"),
	)

	if err != nil {
		t.Fatal(err)
	}

	expectedMod, err := os.ReadFile(modExpect)
	if err != nil {
		t.Fatal(err)
	}
	gotMod, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedLock, err := os.ReadFile(lockExpect)
	if err != nil {
		t.Fatal(err)
	}

	gotLock, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, utils.RmNewline(string(expectedMod)), utils.RmNewline(string(gotMod)))
	assert.Equal(t, utils.RmNewline(string(expectedLock)), utils.RmNewline(string(gotLock)))
}
