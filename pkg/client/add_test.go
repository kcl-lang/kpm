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

func TestAddWithModSpec(t *testing.T) {
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
			msg: "downloading 'kcl-lang/helloworld:0.1.4' from 'ghcr.io/kcl-lang/helloworld:0.1.4'" +
				"adding dependency 'subhelloworld'" +
				"add dependency 'subhelloworld:0.0.1' successfully",
		},
		{
			name:       "TestAddGitWithModSpec",
			pkgSubPath: "git",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git?commit=8308200&mod=cc:0.0.1",
			msg: "cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit '8308200'" +
				"adding dependency 'cc'" +
				"add dependency 'cc:0.0.1' successfully",
		},
		{
			name:       "TestAddGitWithModSpec",
			pkgSubPath: "git_mod_0",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git?commit=8308200&mod=cc",
			msg: "cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit '8308200'" +
				"adding dependency 'cc'" +
				"add dependency 'cc:0.0.1' successfully",
		},
		{
			name:       "TestAddGitWithoutModFileWithModSpec",
			pkgSubPath: "git_mod_1",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git?commit=5ab0fff&mod=cc",
			msg: "cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit '5ab0fff'" +
				"adding dependency 'cc'" +
				"add dependency 'cc:0.0.1' successfully",
		},
		{
			name:       "TestAddGitWithoutModFileWithModSpec",
			pkgSubPath: "git_mod_2",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git?commit=3adfc81&mod=cc",
			msg: "cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit '3adfc81'" +
				"adding dependency 'cc'" +
				"add dependency 'cc:0.0.2' successfully",
		},
		{
			name:       "TestAddGitWithoutModFileWithModSpec",
			pkgSubPath: "git_mod_3",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git?commit=3adfc81&mod=cc:0.0.1",
			msg: "cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit '3adfc81'" +
				"adding dependency 'cc'" +
				"add dependency 'cc:0.0.1' successfully",
		},
		{
			name:       "TestAddLocalWithModSpec",
			pkgSubPath: filepath.Join("local", "pkg"),
			sourceUrl:  "../dep?mod=sub:0.0.1",
			msg: "adding dependency 'sub'" +
				"add dependency 'sub:0.0.1' successfully",
		},
		{
			name:       "TestAddOciWithEmptyVersion",
			pkgSubPath: "empty_version",
			sourceUrl:  "oci://ghcr.io/kcl-lang/helloworld?tag=0.1.4&mod=subhelloworld",
			msg: "downloading 'kcl-lang/helloworld:0.1.4' from 'ghcr.io/kcl-lang/helloworld:0.1.4'" +
				"adding dependency 'subhelloworld'" +
				"add dependency 'subhelloworld:0.0.1' successfully",
		},
		{
			name:       "TestAddOciWithNoSpec",
			pkgSubPath: "no_spec",
			sourceUrl:  "oci://ghcr.io/kcl-lang/helloworld?tag=0.1.4",
			msg: "downloading 'kcl-lang/helloworld:0.1.4' from 'ghcr.io/kcl-lang/helloworld:0.1.4'" +
				"adding dependency 'helloworld'" +
				"add dependency 'helloworld:0.1.4' successfully",
		},
		{
			name:       "TestAddOciWithNoTag",
			pkgSubPath: "no_oci_ref",
			sourceUrl:  "oci://ghcr.io/kcl-lang/helloworld",
			msg: "the latest version '0.1.4' will be downloaded" +
				"downloading 'kcl-lang/helloworld:0.1.4' from 'ghcr.io/kcl-lang/helloworld:0.1.4'" +
				"adding dependency 'helloworld'" +
				"add dependency 'helloworld:0.1.4' successfully",
		},
		{
			name:       "TestAddGitWithNoTag",
			pkgSubPath: "no_git_ref",
			sourceUrl:  "git://github.com/kcl-lang/flask-demo-kcl-manifests.git",
			msg: "the latest version 'ade147b' will be downloaded" +
				"cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit 'ade147b'" +
				"adding dependency 'flask_manifests'" +
				"add dependency 'flask_manifests:0.0.1' successfully",
		},
	}

	for _, tt := range tests {
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

		testFunc := func(t *testing.T, kpmcli *KpmClient) {
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
		}

		RunTestWithGlobalLockAndKpmCli(t, tt.name, testFunc)

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

func TestAddRenameWithModSpec(t *testing.T) {
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

	testFunc := func(t *testing.T, kpmcli *KpmClient) {
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
			WithAddModSpec(&downloader.ModSpec{
				Name:    "helloworld",
				Version: "0.1.2",
			}),
			WithAlias("newpkg"),
		)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(
			"downloading 'kcl-lang/helloworld:0.1.2' from 'ghcr.io/kcl-lang/helloworld:0.1.2'"+
				"adding dependency 'helloworld'"+
				"add dependency 'helloworld:0.1.2' successfully",
		), utils.RmNewline(buf.String()))
	}

	RunTestWithGlobalLockAndKpmCli(t, "TestAddRenameWithModSpec", testFunc)

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

func TestAddWithOnlyModSpec(t *testing.T) {
	testCases := []struct {
		name      string
		testDir   string
		pkgSubDir string
		msg       string
		modSpec   *downloader.ModSpec
	}{
		{
			name:      "TestAddWithOnlyModSpec",
			testDir:   "add_with_mod_spec",
			pkgSubDir: "spec_only",
			msg: "downloading 'kcl-lang/helloworld:0.1.4' from 'ghcr.io/kcl-lang/helloworld:0.1.4'" +
				"adding dependency 'helloworld'" +
				"add dependency 'helloworld:0.1.4' successfully",
			modSpec: &downloader.ModSpec{
				Name:    "helloworld",
				Version: "0.1.4",
			},
		},
		{
			name:      "TestAddWithOnlyModSpecButNoVersion",
			testDir:   "add_with_mod_spec",
			pkgSubDir: "spec_only_no_ver",
			msg: "the latest version '0.1.4' will be downloaded" +
				"downloading 'kcl-lang/helloworld:0.1.4' from 'ghcr.io/kcl-lang/helloworld:0.1.4'" +
				"adding dependency 'helloworld'" +
				"add dependency 'helloworld:0.1.4' successfully",
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

		testFunc := func(t *testing.T, kpmcli *KpmClient) {
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
				WithAddModSpec(tc.modSpec),
			)

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, utils.RmNewline(tc.msg), utils.RmNewline(buf.String()))
		}

		RunTestWithGlobalLockAndKpmCli(t, tc.name, testFunc)

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

func TestAddRenameWithNoSpec(t *testing.T) {
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

	testFunc := func(t *testing.T, kpmcli *KpmClient) {

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

		assert.Equal(t,
			"the latest version '0.1.4' will be downloaded"+
				"downloading 'kcl-lang/helloworld:0.1.4' from 'ghcr.io/kcl-lang/helloworld:0.1.4'"+
				"adding dependency 'helloworld'"+
				"add dependency 'helloworld:0.1.4' successfully",
			utils.RmNewline(buf.String()),
		)

		if err != nil {
			t.Fatal(err)
		}
	}

	RunTestWithGlobalLockAndKpmCli(t, "TestAddRenameWithNoSpec", testFunc)

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
