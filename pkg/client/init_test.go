package client

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/runner"
	"kcl-lang.io/kpm/pkg/utils"
)

// test 'kcl mod init <mod_path>'
func TestModInitPath(t *testing.T) {
	testFunc := func(t *testing.T, kpmcli *KpmClient) {
		testPath := getTestDir("test_init")
		workDir := filepath.Join(testPath, "init")
		defer os.RemoveAll(workDir)
		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		err := kpmcli.Init(
			WithInitModPath(workDir),
		)

		assert.Nil(t, err)
		assert.Equal(t, utils.RmNewline(buf.String()),
			fmt.Sprintf("creating new :%s", filepath.Join(workDir, "kcl.mod"))+
				fmt.Sprintf("creating new :%s", filepath.Join(workDir, "kcl.mod.lock"))+
				fmt.Sprintf("creating new :%s", filepath.Join(workDir, "main.k")))
		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(workDir),
		)

		assert.Nil(t, err)
		assert.Equal(t, kmod.ModFile.Pkg.Name, "init")
		assert.Equal(t, kmod.ModFile.Pkg.Version, "0.0.1")
		assert.Equal(t, kmod.ModFile.Pkg.Edition, runner.GetKclVersion())
	}

	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestModInitPath", TestFunc: testFunc}})
}

// test 'kcl mod init <mod_name>'
func TestModInitName(t *testing.T) {
	testFunc := func(t *testing.T, kpmcli *KpmClient) {
		testPath := getTestDir("test_init")
		workDir := filepath.Join(testPath, "init_0")
		defer os.RemoveAll(filepath.Join(workDir, "InitModName"))

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		err := kpmcli.Init(
			WithInitWorkDir(workDir),
			WithInitModName("InitModName"),
		)

		assert.Nil(t, err)
		assert.Equal(t, utils.RmNewline(buf.String()),
			fmt.Sprintf("creating new :%s", filepath.Join(workDir, "InitModName", "kcl.mod"))+
				fmt.Sprintf("creating new :%s", filepath.Join(workDir, "InitModName", "kcl.mod.lock"))+
				fmt.Sprintf("creating new :%s", filepath.Join(workDir, "InitModName", "main.k")))

		assert.True(t, utils.DirExists(filepath.Join(workDir, "InitModName", "kcl.mod")))
		assert.True(t, utils.DirExists(filepath.Join(workDir, "InitModName", "kcl.mod.lock")))

		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(filepath.Join(workDir, "InitModName")),
		)

		assert.Nil(t, err)
		assert.Equal(t, kmod.ModFile.Pkg.Name, "InitModName")
		assert.Equal(t, kmod.ModFile.Pkg.Version, "0.0.1")
		assert.Equal(t, kmod.ModFile.Pkg.Edition, runner.GetKclVersion())
	}

	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestModInitName", TestFunc: testFunc}})
}

// test 'kcl mod init --version <mod_version>'
func TestModInitOnlyVersion(t *testing.T) {
	testFunc := func(t *testing.T, kpmcli *KpmClient) {
		testPath := getTestDir("test_init")
		workDir := filepath.Join(testPath, "init_1")
		defer func() {
			_ = os.Remove(filepath.Join(workDir, "kcl.mod"))
			_ = os.Remove(filepath.Join(workDir, "kcl.mod.lock"))
			_ = os.Remove(filepath.Join(workDir, "main.k"))
		}()

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		err := kpmcli.Init(
			WithInitWorkDir(workDir),
			WithInitModVersion("0.1.100"),
		)

		assert.Nil(t, err)
		assert.Equal(t, utils.RmNewline(buf.String()),
			fmt.Sprintf("creating new :%s", filepath.Join(workDir, "kcl.mod"))+
				fmt.Sprintf("creating new :%s", filepath.Join(workDir, "kcl.mod.lock"))+
				fmt.Sprintf("creating new :%s", filepath.Join(workDir, "main.k")))

		assert.True(t, utils.DirExists(filepath.Join(workDir, "kcl.mod")))
		assert.True(t, utils.DirExists(filepath.Join(workDir, "kcl.mod.lock")))

		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(workDir),
		)

		assert.Nil(t, err)
		assert.Equal(t, kmod.ModFile.Pkg.Name, "init_1")
		assert.Equal(t, kmod.ModFile.Pkg.Version, "0.1.100")
		assert.Equal(t, kmod.ModFile.Pkg.Edition, runner.GetKclVersion())
	}
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestModInitOnlyVersion", TestFunc: testFunc}})
}

// test 'kcl mod init <mod_name> --version <mod_version>'
func TestModInitNameVersion(t *testing.T) {
	testFunc := func(t *testing.T, kpmcli *KpmClient) {
		testPath := getTestDir("test_init")
		workDir := filepath.Join(testPath, "init_1")
		modPath := filepath.Join(workDir, "InitModName")
		defer func() {
			os.RemoveAll(modPath)
		}()

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		err := kpmcli.Init(
			WithInitWorkDir(workDir),
			WithInitModName("InitModName"),
			WithInitModVersion("0.1.100"),
		)

		assert.Nil(t, err)
		assert.Equal(t, utils.RmNewline(buf.String()),
			fmt.Sprintf("creating new :%s", filepath.Join(modPath, "kcl.mod"))+
				fmt.Sprintf("creating new :%s", filepath.Join(modPath, "kcl.mod.lock"))+
				fmt.Sprintf("creating new :%s", filepath.Join(modPath, "main.k")))

		assert.True(t, utils.DirExists(filepath.Join(modPath, "kcl.mod")))
		assert.True(t, utils.DirExists(filepath.Join(modPath, "kcl.mod.lock")))

		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(modPath),
		)

		assert.Nil(t, err)
		assert.Equal(t, kmod.ModFile.Pkg.Name, "InitModName")
		assert.Equal(t, kmod.ModFile.Pkg.Version, "0.1.100")
		assert.Equal(t, kmod.ModFile.Pkg.Edition, runner.GetKclVersion())
	}

	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestModInitNameVersion", TestFunc: testFunc}})
}

// test 'kcl mod init <mod_name> --version <mod_version> --path <mod_path> '
func TestModInitNameVersionPath(t *testing.T) {
	testFunc := func(t *testing.T, kpmcli *KpmClient) {
		testPath := getTestDir("test_init")
		workDir := filepath.Join(testPath, "init_2")
		modPath := filepath.Join(workDir, "InitModName")
		defer func() {
			_ = os.RemoveAll(modPath)
		}()

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		err := kpmcli.Init(
			WithInitModPath(workDir),
			WithInitModName("InitModName"),
			WithInitModVersion("0.1.100"),
		)

		assert.Nil(t, err)
		assert.Equal(t, utils.RmNewline(buf.String()),
			fmt.Sprintf("creating new :%s", filepath.Join(modPath, "kcl.mod"))+
				fmt.Sprintf("creating new :%s", filepath.Join(modPath, "kcl.mod.lock"))+
				fmt.Sprintf("creating new :%s", filepath.Join(modPath, "main.k")))

		assert.True(t, utils.DirExists(filepath.Join(modPath, "kcl.mod")))
		assert.True(t, utils.DirExists(filepath.Join(modPath, "kcl.mod.lock")))

		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(modPath),
		)

		assert.Nil(t, err)
		assert.Equal(t, kmod.ModFile.Pkg.Name, "InitModName")
		assert.Equal(t, kmod.ModFile.Pkg.Version, "0.1.100")
		assert.Equal(t, kmod.ModFile.Pkg.Edition, runner.GetKclVersion())
	}
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestModInitNameVersionPath", TestFunc: testFunc}})
}

// test 'kcl mod init'
func TestModInitPwd(t *testing.T) {
	testFunc := func(t *testing.T, kpmcli *KpmClient) {
		testPath := getTestDir("test_init")
		workDir := filepath.Join(testPath, "init_4")
		defer func() {
			_ = os.Remove(filepath.Join(workDir, "kcl.mod"))
			_ = os.Remove(filepath.Join(workDir, "kcl.mod.lock"))
			_ = os.Remove(filepath.Join(workDir, "main.k"))
		}()

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		err := kpmcli.Init(
			WithInitWorkDir(workDir),
		)

		assert.Nil(t, err)
		assert.Equal(t, utils.RmNewline(buf.String()),
			fmt.Sprintf("creating new :%s", filepath.Join(workDir, "kcl.mod"))+
				fmt.Sprintf("creating new :%s", filepath.Join(workDir, "kcl.mod.lock"))+
				fmt.Sprintf("creating new :%s", filepath.Join(workDir, "main.k")))

		assert.True(t, utils.DirExists(filepath.Join(workDir, "kcl.mod")))
		assert.True(t, utils.DirExists(filepath.Join(workDir, "kcl.mod.lock")))

		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(workDir),
		)

		assert.Nil(t, err)
		assert.Equal(t, kmod.ModFile.Pkg.Name, "init_4")
		assert.Equal(t, kmod.ModFile.Pkg.Version, "0.0.1")
		assert.Equal(t, kmod.ModFile.Pkg.Edition, runner.GetKclVersion())
	}
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestModInitPwd", TestFunc: testFunc}})
}

// test 'kcl mod init' with existing mod file
func TestModInitWithExistFile(t *testing.T) {
	testFunc := func(t *testing.T, kpmcli *KpmClient) {
		testPath := getTestDir("test_init")
		workDir := filepath.Join(testPath, "init_5")

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		err := kpmcli.Init(
			WithInitWorkDir(workDir),
		)

		assert.Nil(t, err)
		assert.Equal(t, utils.RmNewline(buf.String()),
			fmt.Sprintf("creating new :%s", filepath.Join(workDir, "kcl.mod"))+
				fmt.Sprintf("'%s' already exists", filepath.Join(workDir, "kcl.mod"))+
				fmt.Sprintf("creating new :%s", filepath.Join(workDir, "kcl.mod.lock"))+
				fmt.Sprintf("'%s' already exists", filepath.Join(workDir, "kcl.mod.lock"))+
				fmt.Sprintf("creating new :%s", filepath.Join(workDir, "main.k"))+
				fmt.Sprintf("'%s' already exists", filepath.Join(workDir, "main.k")))

		assert.True(t, utils.DirExists(filepath.Join(workDir, "kcl.mod")))
		assert.True(t, utils.DirExists(filepath.Join(workDir, "kcl.mod.lock")))

		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(workDir),
		)

		assert.Nil(t, err)
		assert.Equal(t, kmod.ModFile.Pkg.Name, "init_5_exist")
		assert.Equal(t, kmod.ModFile.Pkg.Version, "0.1.1")
		assert.Equal(t, kmod.ModFile.Pkg.Edition, runner.GetKclVersion())
		assert.Equal(t, kmod.ModFile.Dependencies.Deps.Len(), 1)
		assert.Equal(t, kmod.Dependencies.Deps.Len(), 1)

		res, err := kpmcli.Run(
			WithRunSource(&downloader.Source{
				Local: &downloader.Local{
					Path: workDir,
				},
			}),
		)

		assert.Nil(t, err)
		assert.Equal(t, utils.RmNewline(res.GetRawYamlResult()),
			"The_first_kcl_program: Hello World!"+"The_second_kcl_program: test")
	}

	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestModInitWithExitModFile", TestFunc: testFunc}})
}
