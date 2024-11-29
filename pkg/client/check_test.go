package client

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	pkg "kcl-lang.io/kpm/pkg/package"
)

func TestModCheckPass(t *testing.T) {
	testModCheckPass := func(t *testing.T, kpmcli *KpmClient) {
		testDir := filepath.Join(getTestDir("test_mod_check"), "pass")
		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(testDir),
		)

		if err != nil {
			t.Fatalf("failed to load kcl package: %v", err)
		}

		err = kpmcli.Check(
			WithCheckKclMod(kmod),
		)

		if err != nil {
			t.Fatalf("failed to check kcl package: %v", err)
		}
	}
	RunTestWithGlobalLockAndKpmCli(t, "test_mod_check_pass", testModCheckPass)
}

func TestModCheckNameFailed(t *testing.T) {
	testModCheckNameFailed := func(t *testing.T, kpmcli *KpmClient) {
		testDir := filepath.Join(getTestDir("test_mod_check"), "name_failed")
		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(testDir),
		)

		if err != nil {
			t.Fatalf("failed to load kcl package: %v", err)
		}

		err = kpmcli.Check(
			WithCheckKclMod(kmod),
		)

		assert.Equal(t, err.Error(), "invalid name: invalid/mod/name")
	}
	RunTestWithGlobalLockAndKpmCli(t, "test_mod_check_name_failed", testModCheckNameFailed)
}

func TestModCheckVersionFailed(t *testing.T) {
	testModCheckVersionFailed := func(t *testing.T, kpmcli *KpmClient) {
		testDir := filepath.Join(getTestDir("test_mod_check"), "version_failed")
		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(testDir),
		)
		if err != nil {
			t.Fatalf("failed to load kcl package: %v", err)
		}

		err = kpmcli.Check(
			WithCheckKclMod(kmod),
		)

		assert.Equal(t, err.Error(), "invalid version: invalid_version for version_failed")
	}
	RunTestWithGlobalLockAndKpmCli(t, "test_mod_check_version_failed", testModCheckVersionFailed)
}

func TestCheckDepSumPass(t *testing.T) {
	testDepSumFunc := func(t *testing.T, kpmcli *KpmClient) {
		pkgDir := filepath.Join(getTestDir("test_mod_check"), "sum")

		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(pkgDir),
		)

		if err != nil {
			t.Fatalf("failed to load kcl package: %v", err)
		}

		err = kpmcli.Check(
			WithCheckKclMod(kmod),
		)
		if err != nil {
			t.Fatalf("failed to check kcl package: %v", err)
		}
	}

	RunTestWithGlobalLockAndKpmCli(t, "TestCheckDepSumPass", testDepSumFunc)
}

func TestCheckDepSumFailed(t *testing.T) {
	testDepSumFunc := func(t *testing.T, kpmcli *KpmClient) {
		pkgDir := filepath.Join(getTestDir("test_mod_check"), "sum")

		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(pkgDir),
		)

		if err != nil {
			t.Fatalf("failed to load kcl package: %v", err)
		}

		dep := kmod.Dependencies.Deps.GetOrDefault("helloworld", pkg.TestPkgDependency)
		dep.Sum = "invalid_sum"
		ok := kmod.Dependencies.Deps.Set("helloworld", dep)
		assert.Assert(t, !ok)

		err = kpmcli.Check(
			WithCheckKclMod(kmod),
		)

		assert.Equal(t, err.Error(), "checksum verification failed for 'helloworld': "+
			"expected '9J9HOMhdypaDYf0J7PqtpGTdlkbxkN0HFEYhosHhf4U=', got 'invalid_sum'")
	}

	RunTestWithGlobalLockAndKpmCli(t, "TestCheckDepSumFailed", testDepSumFunc)
}
