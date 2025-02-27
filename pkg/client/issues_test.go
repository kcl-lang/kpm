package client

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/features"
	"kcl-lang.io/kpm/pkg/mock"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

func TestKclIssue1760(t *testing.T) {
	testPath := "github.com/kcl-lang/kcl/issues/1760"
	testCases := []struct {
		name  string
		setup func()
	}{
		{
			name: "Default",
			setup: func() {
				features.Disable(features.SupportNewStorage)
				features.Disable(features.SupportMVS)
			},
		},
		{
			name: "SupportNewStorage",
			setup: func() {
				features.Enable(features.SupportNewStorage)
				features.Disable(features.SupportMVS)
			},
		},
		{
			name: "SupportMVS",
			setup: func() {
				features.Disable(features.SupportNewStorage)
				features.Enable(features.SupportMVS)
			},
		},
		{
			name: "SupportNewStorageAndMVS",
			setup: func() {
				features.Enable(features.SupportNewStorage)
				features.Enable(features.SupportMVS)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		tc.setup()

		testFunc := func(t *testing.T, kpmcli *KpmClient) {
			rootPath := getTestDir("issues")
			mainKFilePath := filepath.Join(rootPath, testPath, "a", "main.k")
			var buf bytes.Buffer
			kpmcli.SetLogWriter(&buf)

			res, err := kpmcli.Run(
				WithRunSource(
					&downloader.Source{
						Local: &downloader.Local{
							Path: mainKFilePath,
						},
					},
				),
			)

			if err != nil {
				t.Fatal(err)
			}

			assert.Contains(t,
				utils.RmNewline(buf.String()),
				"downloading 'kcl-lang/fluxcd-source-controller:v1.3.2' from 'ghcr.io/kcl-lang/fluxcd-source-controller:v1.3.2'",
			)
			assert.Contains(t,
				utils.RmNewline(buf.String()),
				"downloading 'kcl-lang/k8s:1.31.2' from 'ghcr.io/kcl-lang/k8s:1.31.2'",
			)

			assert.Contains(t,
				utils.RmNewline(buf.String()),
				"downloading 'kcl-lang/fluxcd-helm-controller:v1.0.3' from 'ghcr.io/kcl-lang/fluxcd-helm-controller:v1.0.3'",
			)
			assert.Equal(t, res.GetRawYamlResult(), "The_first_kcl_program: Hello World!")
		}

		RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: tc.name, TestFunc: testFunc}})
	}
}

func TestKpmIssue550(t *testing.T) {
	testPath := "github.com/kcl-lang/kpm/issues/550"
	testCases := []struct {
		name        string
		setup       func()
		expected    string
		winExpected string
	}{
		{
			name: "Default",
			setup: func() {
				features.Disable(features.SupportNewStorage)
				features.Disable(features.SupportMVS)
			},
			expected:    filepath.Join("flask-demo-kcl-manifests_test-branch-without-modfile", "aa", "cc"),
			winExpected: filepath.Join("flask-demo-kcl-manifests_test-branch-without-modfile", "aa", "cc"),
		},
		{
			name: "SupportNewStorage",
			setup: func() {
				features.Enable(features.SupportNewStorage)
				features.Disable(features.SupportMVS)
			},
			expected:    filepath.Join("git", "src", "200297ed26e4aeb7", "flask-demo-kcl-manifests", "test-branch-without-modfile", "aa", "cc"),
			winExpected: filepath.Join("git", "src", "3523a44a55384201", "flask-demo-kcl-manifests", "test-branch-without-modfile", "aa", "cc"),
		},
		{
			name: "SupportMVS",
			setup: func() {
				features.Disable(features.SupportNewStorage)
				features.Enable(features.SupportMVS)
			},
			expected:    filepath.Join("flask-demo-kcl-manifests_test-branch-without-modfile", "aa", "cc"),
			winExpected: filepath.Join("flask-demo-kcl-manifests_test-branch-without-modfile", "aa", "cc"),
		},
		{
			name: "SupportNewStorageAndMVS",
			setup: func() {
				features.Enable(features.SupportNewStorage)
				features.Enable(features.SupportMVS)
			},
			expected:    filepath.Join("git", "src", "200297ed26e4aeb7", "flask-demo-kcl-manifests", "test-branch-without-modfile", "aa", "cc"),
			winExpected: filepath.Join("git", "src", "3523a44a55384201", "flask-demo-kcl-manifests", "test-branch-without-modfile", "aa", "cc"),
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable

		tc.setup()

		testFunc := func(t *testing.T, kpmcli *KpmClient) {
			rootPath := getTestDir("issues")
			modPath := filepath.Join(rootPath, testPath, "pkg")
			var buf bytes.Buffer
			kpmcli.SetLogWriter(&buf)

			tmpKpmHome, err := os.MkdirTemp("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpKpmHome)

			kpmcli.homePath = tmpKpmHome

			kMod, err := pkg.LoadKclPkgWithOpts(
				pkg.WithPath(modPath),
			)

			if err != nil {
				t.Fatal(err)
			}

			res, err := kpmcli.ResolveDepsMetadataInJsonStr(kMod, true)

			if err != nil {
				t.Fatal(err)
			}

			expectedPath := filepath.Join(tmpKpmHome, tc.expected)
			if runtime.GOOS == "windows" {
				expectedPath = filepath.Join(tmpKpmHome, tc.winExpected)
				expectedPath = strings.ReplaceAll(expectedPath, "\\", "\\\\")
			}

			assert.Equal(t, res, fmt.Sprintf(
				`{"packages":{"cc":{"name":"cc","manifest_path":"%s"}}}`,
				expectedPath,
			))

			resMap, err := kpmcli.ResolveDepsIntoMap(kMod)

			if err != nil {
				t.Fatal(err)
			}
			fmt.Printf("buf.String(): %v\n", buf.String())
			assert.Contains(t,
				utils.RmNewline(buf.String()),
				"cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with branch 'test-branch-without-modfile'",
			)
			assert.Equal(t, len(resMap), 1)
			if runtime.GOOS == "windows" {
				assert.Equal(t, resMap["cc"], filepath.Join(tmpKpmHome, tc.winExpected))
			} else {
				assert.Equal(t, resMap["cc"], filepath.Join(tmpKpmHome, tc.expected))
			}
		}

		RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: tc.name, TestFunc: testFunc}})
	}
}

func TestKpmIssue226(t *testing.T) {
	testPath := "github.com/kcl-lang/kpm/issues/226"
	test_add_dep_with_git_commit := func(t *testing.T, kpmcli *KpmClient) {
		rootPath := getTestDir("issues")
		modPath := filepath.Join(rootPath, testPath, "add_with_commit")
		modFileBk := filepath.Join(modPath, "kcl.mod.bk")
		LockFileBk := filepath.Join(modPath, "kcl.mod.lock.bk")
		modFile := filepath.Join(modPath, "kcl.mod")
		LockFile := filepath.Join(modPath, "kcl.mod.lock")
		modFileExpect := filepath.Join(modPath, "kcl.mod.expect")
		LockFileExpect := filepath.Join(modPath, "kcl.mod.lock.expect")

		defer func() {
			_ = os.RemoveAll(modFile)
			_ = os.RemoveAll(LockFile)
		}()

		err := copy.Copy(modFileBk, modFile)
		if err != nil {
			t.Fatal(err)
		}
		err = copy.Copy(LockFileBk, LockFile)
		if err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		kpkg, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(modPath),
		)

		if err != nil {
			t.Fatal(err)
		}

		err = kpmcli.Add(
			WithAddKclPkg(kpkg),
			WithAddSource(
				&downloader.Source{
					Git: &downloader.Git{
						Url:    "https://github.com/kcl-lang/flask-demo-kcl-manifests.git",
						Commit: "ade147b",
					},
				},
			),
		)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(buf.String()),
			"cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit 'ade147b'"+
				"adding dependency 'flask_manifests'"+
				"add dependency 'flask_manifests:0.0.1' successfully")

		modFileContent, err := os.ReadFile(modFile)
		if err != nil {
			t.Fatal(err)
		}
		lockFileContent, err := os.ReadFile(LockFile)
		if err != nil {
			t.Fatal(err)
		}

		modFileExpectContent, err := os.ReadFile(modFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		lockFileExpectContent, err := os.ReadFile(LockFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(string(modFileContent)), utils.RmNewline(string(modFileExpectContent)))
		assert.Equal(t, utils.RmNewline(string(lockFileContent)), utils.RmNewline(string(lockFileExpectContent)))
	}

	test_update_with_git_commit := func(t *testing.T, kpmcli *KpmClient) {
		rootPath := getTestDir("issues")
		modPath := filepath.Join(rootPath, testPath, "update_check_version")
		modFileBk := filepath.Join(modPath, "kcl.mod.bk")
		LockFileBk := filepath.Join(modPath, "kcl.mod.lock.bk")
		modFile := filepath.Join(modPath, "kcl.mod")
		LockFile := filepath.Join(modPath, "kcl.mod.lock")
		modFileExpect := filepath.Join(modPath, "kcl.mod.expect")
		LockFileExpect := filepath.Join(modPath, "kcl.mod.lock.expect")

		defer func() {
			_ = os.RemoveAll(modFile)
			_ = os.RemoveAll(LockFile)
		}()

		err := copy.Copy(modFileBk, modFile)
		if err != nil {
			t.Fatal(err)
		}
		err = copy.Copy(LockFileBk, LockFile)
		if err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		kpkg, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(modPath),
		)

		if err != nil {
			t.Fatal(err)
		}

		_, err = kpmcli.Update(
			WithUpdatedKclPkg(kpkg),
		)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(buf.String()),
			"cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit 'ade147b'")

		modFileContent, err := os.ReadFile(modFile)
		if err != nil {
			t.Fatal(err)
		}
		lockFileContent, err := os.ReadFile(LockFile)
		if err != nil {
			t.Fatal(err)
		}

		modFileExpectContent, err := os.ReadFile(modFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		lockFileExpectContent, err := os.ReadFile(LockFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(string(modFileContent)), utils.RmNewline(string(modFileExpectContent)))
		assert.Equal(t, utils.RmNewline(string(lockFileContent)), utils.RmNewline(string(lockFileExpectContent)))
	}

	test_update_with_git_commit_invalid := func(t *testing.T, kpmcli *KpmClient) {
		rootPath := getTestDir("issues")
		modPath := filepath.Join(rootPath, testPath, "update_check_version_invalid")
		modFileBk := filepath.Join(modPath, "kcl.mod.bk")
		LockFileBk := filepath.Join(modPath, "kcl.mod.lock.bk")
		modFile := filepath.Join(modPath, "kcl.mod")
		LockFile := filepath.Join(modPath, "kcl.mod.lock")
		modFileExpect := filepath.Join(modPath, "kcl.mod.expect")
		LockFileExpect := filepath.Join(modPath, "kcl.mod.lock.expect")

		defer func() {
			_ = os.RemoveAll(modFile)
			_ = os.RemoveAll(LockFile)
		}()

		err := copy.Copy(modFileBk, modFile)
		if err != nil {
			t.Fatal(err)
		}
		err = copy.Copy(LockFileBk, LockFile)
		if err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		kpkg, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(modPath),
		)

		if err != nil {
			t.Fatal(err)
		}

		_, err = kpmcli.Update(
			WithUpdatedKclPkg(kpkg),
		)

		assert.Equal(t, err.Error(), "package 'flask_manifests:0.100.0' not found")

		assert.Equal(t, utils.RmNewline(buf.String()),
			"cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit 'ade147b'")

		modFileContent, err := os.ReadFile(modFile)
		if err != nil {
			t.Fatal(err)
		}
		lockFileContent, err := os.ReadFile(LockFile)
		if err != nil {
			t.Fatal(err)
		}

		modFileExpectContent, err := os.ReadFile(modFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		lockFileExpectContent, err := os.ReadFile(LockFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(string(modFileContent)), utils.RmNewline(string(modFileExpectContent)))
		assert.Equal(t, utils.RmNewline(string(lockFileContent)), utils.RmNewline(string(lockFileExpectContent)))
	}

	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "add_dep_with_git_commit", TestFunc: test_add_dep_with_git_commit}})
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "update_with_git_commit", TestFunc: test_update_with_git_commit}})
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "update_with_git_commit_invalid", TestFunc: test_update_with_git_commit_invalid}})
}

func TestKclIssue1768(t *testing.T) {
	testPath := "github.com/kcl-lang/kcl/issues/1768"
	test_push_with_tag := func(t *testing.T, kpmcli *KpmClient) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping test on Windows")
		}
		err := mock.StartDockerRegistry()
		if err != nil {
			t.Errorf("Error starting docker registry: %v", err)
		}

		defer func() {
			err = mock.CleanTestEnv()
			if err != nil {
				t.Errorf("Error stopping docker registry: %v", err)
			}
		}()

		kpmcli.SetInsecureSkipTLSverify(true)
		err = kpmcli.LoginOci("localhost:5001", "test", "1234")
		if err != nil {
			t.Errorf("Error logging in to docker registry: %v", err)
		}

		rootPath := getTestDir("issues")
		pushedModPath := filepath.Join(rootPath, testPath, "pushed_mod")

		modPath := filepath.Join(rootPath, testPath, "depends_on_pushed_mod")
		modFileBk := filepath.Join(modPath, "kcl.mod.bk")
		LockFileBk := filepath.Join(modPath, "kcl.mod.lock.bk")
		modFile := filepath.Join(modPath, "kcl.mod")
		LockFile := filepath.Join(modPath, "kcl.mod.lock")
		modFileExpect := filepath.Join(modPath, "kcl.mod.expect")
		LockFileExpect := filepath.Join(modPath, "kcl.mod.lock.expect")

		defer func() {
			_ = os.RemoveAll(modFile)
			_ = os.RemoveAll(LockFile)
		}()

		err = copy.Copy(modFileBk, modFile)
		if err != nil {
			t.Fatal(err)
		}
		err = copy.Copy(LockFileBk, LockFile)
		if err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		err = kpmcli.Push(
			WithPushModPath(pushedModPath),
			WithPushSource(downloader.Source{
				Oci: &downloader.Oci{
					Reg:  "localhost:5001",
					Repo: "test/oci_pushed_mod",
					Tag:  "v9.9.9",
				},
			}),
		)

		if err != (*reporter.KpmEvent)(nil) {
			t.Errorf("Error pushing kcl package: %v", err)
		}

		assert.Contains(t, buf.String(), "package 'pushed_mod' will be pushed")
		assert.Contains(t, buf.String(), "pushed [registry] localhost:5001/test/oci_pushed_mod")
		assert.Contains(t, buf.String(), "digest: sha256:")

		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(modPath),
		)

		err = kpmcli.Add(
			WithAddKclPkg(kmod),
			WithAddSource(
				&downloader.Source{
					Oci: &downloader.Oci{
						Reg:  "localhost:5001",
						Repo: "test/oci_pushed_mod",
						Tag:  "v9.9.9",
					}},
			),
			WithAddModSpec(
				&downloader.ModSpec{
					Name:    "pushed_mod",
					Version: "0.0.1",
				},
			),
		)

		if err != nil {
			t.Errorf("Error adding dependency: %v", err)
		}

		modFileContent, err := os.ReadFile(modFile)
		if err != nil {
			t.Fatal(err)
		}
		lockFileContent, err := os.ReadFile(LockFile)
		if err != nil {
			t.Fatal(err)
		}

		modFileExpectContent, err := os.ReadFile(modFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		lockFileExpectContent, err := os.ReadFile(LockFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(string(modFileContent)), utils.RmNewline(string(modFileExpectContent)))
		assert.Equal(t, utils.RmNewline(string(lockFileContent)), utils.RmNewline(string(lockFileExpectContent)))
	}
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "test_push_with_tag", TestFunc: test_push_with_tag}})
}

func TestKclIssue1788(t *testing.T) {
	testPath := "github.com/kcl-lang/kcl/issues/1788"

	test_run_only_file_not_generate_mod := func(t *testing.T, kpmcli *KpmClient) {
		rootPath := getTestDir("issues")
		kfilePath := filepath.Join(rootPath, testPath, "mod", "main.k")
		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		res, err := kpmcli.Run(
			WithRunSource(
				&downloader.Source{
					Local: &downloader.Local{
						Path: kfilePath,
					},
				},
			),
		)

		if err != nil {
			t.Fatal(err)
		}

		modFilePath := filepath.Join(testPath, "kcl.mod")
		modLockFilePath := filepath.Join(testPath, "kcl.mod.lock")

		assert.Equal(t, res.GetRawYamlResult(), "The_first_kcl_program: Hello World!")
		assert.Equal(t, buf.String(), "")
		assert.Equal(t, utils.DirExists(modFilePath), false)
		assert.Equal(t, utils.DirExists(modLockFilePath), false)
	}

	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "test_run_only_file_not_generate_mod", TestFunc: test_run_only_file_not_generate_mod}})
}

func TestKpmIssue587(t *testing.T) {
	testPath := "github.com/kcl-lang/kpm/issues/587"

	test_download_with_git_dep := func(t *testing.T, kpmcli *KpmClient) {
		rootPath := getTestDir("issues")
		kfilePath := filepath.Join(rootPath, testPath)
		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		res, err := kpmcli.Run(
			WithRunSource(
				&downloader.Source{
					Local: &downloader.Local{
						Path: kfilePath,
					},
				},
			),
		)

		if err != nil {
			t.Fatal(err)
		}

		modFilePath := filepath.Join(testPath, "kcl.mod")
		modLockFilePath := filepath.Join(testPath, "kcl.mod.lock")

		assert.Equal(t, res.GetRawYamlResult(), "The_first_kcl_program: Hello World!")
		assert.Equal(t, buf.String(), "cloning 'git://github.com/kcl-lang/flask-demo-kcl-manifests.git' with tag 'v0.1.0'\n")
		assert.Equal(t, utils.DirExists(modFilePath), false)
		assert.Equal(t, utils.DirExists(modLockFilePath), false)
	}

	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "test_download_with_git_dep", TestFunc: test_download_with_git_dep}})
}

func TestKpmIssue605(t *testing.T) {
	testPath := "github.com/kcl-lang/kpm/issues/605"

	test_run_with_exist_checksum := func(t *testing.T, kpmcli *KpmClient) {
		rootPath := getTestDir("issues")
		kfilePath := filepath.Join(rootPath, testPath, "pkg")
		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		// Run the first time，download the dependency to cache and generate the checksum
		_, err := kpmcli.Run(
			WithRunSource(
				&downloader.Source{
					Local: &downloader.Local{
						Path: kfilePath,
					},
				},
			),
		)

		if err != nil {
			t.Fatal(err)
		}

		// Run the second time, this time must be finished in 1 second
		startTime := time.Now()
		_, err = kpmcli.Run(
			WithRunSource(
				&downloader.Source{
					Local: &downloader.Local{
						Path: kfilePath,
					},
				},
			),
		)
		duration := time.Since(startTime)

		if err != nil {
			t.Fatal(err)
		}
		assert.LessOrEqual(t, duration.Seconds(), 0.7, "The second run should finish in 1 second")
	}

	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "test_run_with_exist_checksum", TestFunc: test_run_with_exist_checksum}})
}
