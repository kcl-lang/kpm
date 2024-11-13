package client

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/utils"
)

func testAddWithModSpec(t *testing.T) {
	tests := []struct {
		name       string
		pkgSubPath string
		sourceUrl  string
		msg        string
	}{
		{
			name:       "TestAddOciWithModSpec",
			pkgSubPath: "oci",
			sourceUrl:  "oci://ghcr.io/kcl-lang/helloworld?tag=0.1.4&mod=subhelloworld:0.0.1",
			msg:        "adding dependency 'subhelloworld'add dependency 'subhelloworld:0.0.1' successfully",
		},
		{
			name:       "TestAddGitWithModSpec",
			pkgSubPath: "git",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git?commit=8308200&mod=cc:0.0.1",
			msg:        "adding dependency 'cc'add dependency 'cc:0.0.1' successfully",
		},
		{
			name:       "TestAddGitWithModSpec",
			pkgSubPath: "git_mod_0",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git?commit=8308200&mod=cc",
			msg:        "adding dependency 'cc'add dependency 'cc:0.0.1' successfully",
		},
		{
			name:       "TestAddGitWithoutModFileWithModSpec",
			pkgSubPath: "git_mod_1",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git?commit=5ab0fff&mod=cc",
			msg:        "cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit '5ab0fff'adding dependency 'cc'add dependency 'cc:0.0.1' successfully",
		},
		{
			name:       "TestAddLocalWithModSpec",
			pkgSubPath: filepath.Join("local", "pkg"),
			sourceUrl:  "../dep?mod=sub:0.0.1",
			msg:        "adding dependency 'sub'add dependency 'sub:0.0.1' successfully",
		},
		{
			name:       "TestAddOciWithEmptyVersion",
			pkgSubPath: "empty_version",
			sourceUrl:  "oci://ghcr.io/kcl-lang/helloworld?tag=0.1.4&mod=subhelloworld",
			msg:        "adding dependency 'subhelloworld'add dependency 'subhelloworld:0.0.1' successfully",
		},
		{
			name:       "TestAddOciWithNoSpec",
			pkgSubPath: "no_spec",
			sourceUrl:  "oci://ghcr.io/kcl-lang/helloworld?tag=0.1.4",
			msg:        "adding dependency 'helloworld'add dependency 'helloworld:0.1.4' successfully",
		},
		{
			name:       "TestAddOciWithNoTag",
			pkgSubPath: "no_oci_ref",
			sourceUrl:  "oci://ghcr.io/kcl-lang/helloworld",
			msg:        "the lastest version '0.1.4' will be downloadedadding dependency 'helloworld'add dependency 'helloworld:0.1.4' successfully",
		},
		{
			name:       "TestAddGitWithNoTag",
			pkgSubPath: "no_git_ref",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git",
			msg:        "the lastest version 'ade147b' will be downloadedadding dependency 'flask_manifests'add dependency 'flask_manifests:0.0.1' successfully",
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

			var buf bytes.Buffer
			kpmcli.SetLogWriter(&buf)

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

			assert.Equal(t, utils.RmNewline(tt.msg), utils.RmNewline(buf.String()))

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

func testAddRenameWithModSpec(t *testing.T) {
	testDir := getTestDir("add_with_mod_spec")
	pkgPath := filepath.Join(testDir, "rename_spec_only")

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
		WithAddModSpec(&downloader.ModSpec{
			Name:    "helloworld",
			Version: "0.1.2",
		}),
		WithAlias("newpkg"),
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

func testAddWithOnlyModSpec(t *testing.T) {
	testCases := []struct {
		testDir   string
		pkgSubDir string
		modSpec   *downloader.ModSpec
	}{
		{
			testDir:   "add_with_mod_spec",
			pkgSubDir: "spec_only",
			modSpec: &downloader.ModSpec{
				Name:    "helloworld",
				Version: "0.1.4",
			},
		},
		{
			testDir:   "add_with_mod_spec",
			pkgSubDir: "spec_only_no_ver",
			modSpec: &downloader.ModSpec{
				Name: "helloworld",
			},
		},
	}

	for _, tc := range testCases {
		testDir := getTestDir(tc.testDir)
		pkgPath := filepath.Join(testDir, tc.pkgSubDir)

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
			WithAddModSpec(tc.modSpec),
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
}

func testAddRenameWithNoSpec(t *testing.T) {
	testDir := getTestDir("add_with_mod_spec")
	pkgPath := filepath.Join(testDir, "rename_no_spec")

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
		WithAddSource(&downloader.Source{
			ModSpec: &downloader.ModSpec{
				Alias: "newpkg",
			},
			Oci: &downloader.Oci{
				Reg:  "ghcr.io",
				Repo: "kcl-lang/helloworld",
			},
		}),
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
