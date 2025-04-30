package client

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"kcl-lang.io/kpm/pkg/mock"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func initTestDir(subDir string) string {
	testDir := getTestDir(subDir)
	// clean the test data
	_ = os.RemoveAll(testDir)
	_ = os.Mkdir(testDir, 0755)

	return testDir
}

type TestSuite struct {
	Name     string
	TestFunc func(t *testing.T, kpmcli *KpmClient)
}

// Use a global variable to store the kpmcli instance.
func RunTestWithGlobalLockAndKpmCli(t *testing.T, testSuites []TestSuite) {
	kpmcli, err := NewKpmClient()
	if err != nil {
		t.Errorf("Error acquiring lock: %v", err)
	}
	err = kpmcli.AcquirePackageCacheLock()
	if err != nil {
		t.Errorf("Error acquiring lock: %v", err)
	}

	defer func() {
		err = kpmcli.ReleasePackageCacheLock()
		if err != nil {
			t.Errorf("Error acquiring lock: %v", err)
		}
	}()

	// create a tmp dir as kpm home for test
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Errorf("Error acquiring lock: %v", err)
	}
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)
	kpmcli.SetHomePath(tmpDir)

	for _, testSuite := range testSuites {
		t.Run(testSuite.Name, func(t *testing.T) {
			testSuite.TestFunc(t, kpmcli)
		})
	}
}

func WithMockRegistry(t *testing.T, kpmcli *KpmClient, testBody func()) {
	if isWindows() {
		t.Skip("Skipping test on Windows")
	}

	if err := mock.StartDockerRegistry(); err != nil {
		t.Fatalf("Failed to start mock registry: %v", err)
	}
	defer func() {
		if err := mock.CleanTestEnv(); err != nil {
			t.Errorf("Failed to clean up test environment: %v", err)
		}
	}()
	kpmcli.SetInsecureSkipTLSverify(true)
	if err := kpmcli.LoginOci("localhost:5001", "test", "1234"); err != nil {
		t.Fatalf("Failed to login to mock registry: %v", err)
	}
	testBody()
}
func isWindows() bool {
	return runtime.GOOS == "windows"
}
