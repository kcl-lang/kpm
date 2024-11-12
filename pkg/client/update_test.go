package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/features"
	"kcl-lang.io/kpm/pkg/utils"
)

func testUpdate(t *testing.T) {
	features.Enable(features.SupportMVS)
	testDir := getTestDir("test_update_with_mvs")
	kpmcli, err := NewKpmClient()
	if err != nil {
		t.Fatal(err)
	}

	updates := []struct {
		name   string
		before func() error
	}{
		{
			name:   "update_0",
			before: func() error { return nil },
		},
		{
			name: "update_1",
			before: func() error {
				if err := copy.Copy(filepath.Join(testDir, "update_1", "pkg", "kcl.mod.bk"), filepath.Join(testDir, "update_1", "pkg", "kcl.mod")); err != nil {
					return err
				}
				if err := copy.Copy(filepath.Join(testDir, "update_1", "pkg", "kcl.mod.lock.bk"), filepath.Join(testDir, "update_1", "pkg", "kcl.mod.lock")); err != nil {
					return err
				}
				return nil
			},
		},
	}

	for _, update := range updates {
		if err := update.before(); err != nil {
			t.Fatal(err)
		}

		kpkg, err := kpmcli.LoadPkgFromPath(filepath.Join(testDir, update.name, "pkg"))
		if err != nil {
			t.Fatal(err)
		}

		_, err = kpmcli.Update(WithUpdatedKclPkg(kpkg))
		if err != nil {
			t.Fatal(err)
		}

		expectedMod, err := os.ReadFile(filepath.Join(testDir, update.name, "pkg", "kcl.mod.expect"))
		if err != nil {
			t.Fatal(err)
		}

		expectedModLock, err := os.ReadFile(filepath.Join(testDir, update.name, "pkg", "kcl.mod.lock.expect"))
		if err != nil {
			t.Fatal(err)
		}

		gotMod, err := os.ReadFile(filepath.Join(testDir, update.name, "pkg", "kcl.mod"))
		if err != nil {
			t.Fatal(err)
		}

		gotModLock, err := os.ReadFile(filepath.Join(testDir, update.name, "pkg", "kcl.mod.lock"))
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(string(expectedMod)), utils.RmNewline(string(gotMod)))
		assert.Equal(t, utils.RmNewline(string(expectedModLock)), utils.RmNewline(string(gotModLock)))
	}
}
