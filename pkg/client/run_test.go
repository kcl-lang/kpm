package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/features"
	"kcl-lang.io/kpm/pkg/utils"
)

func testRunWithModSpecVersion(t *testing.T, kpmcli *KpmClient) {
	pkgPath := getTestDir("test_run_with_modspec_version")
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

	res, err := kpmcli.Run(
		WithRunSource(
			&downloader.Source{
				Local: &downloader.Local{
					Path: pkgPath,
				},
			},
		),
	)

	if err != nil {
		t.Errorf("Failed to run package: %v", err)
	}

	assert.Equal(t, res.GetRawYamlResult(), "res: Hello World!")
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

func TestRun(t *testing.T) {
	features.Enable(features.SupportNewStorage)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunWithOciDownloader", testRunWithOciDownloader)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunDefaultRegistryDep", testRunDefaultRegistryDep)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunInVendor", testRunInVendor)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunRemoteWithArgsInvalid", testRunRemoteWithArgsInvalid)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunRemoteWithArgs", testRunRemoteWithArgs)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunWithNoSumCheck", testRunWithNoSumCheck)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunWithGitPackage", testRunWithGitPackage)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunGit", testRunGit)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunOciWithSettingsFile", testRunOciWithSettingsFile)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunWithModSpecVersion", testRunWithModSpecVersion)

	features.Disable(features.SupportNewStorage)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunWithOciDownloader", testRunWithOciDownloader)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunDefaultRegistryDep", testRunDefaultRegistryDep)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunInVendor", testRunInVendor)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunRemoteWithArgsInvalid", testRunRemoteWithArgsInvalid)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunRemoteWithArgs", testRunRemoteWithArgs)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunWithNoSumCheck", testRunWithNoSumCheck)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunWithGitPackage", testRunWithGitPackage)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunGit", testRunGit)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunOciWithSettingsFile", testRunOciWithSettingsFile)
	RunTestWithGlobalLockAndKpmCli(t, "TestRunWithModSpecVersion", testRunWithModSpecVersion)
}
