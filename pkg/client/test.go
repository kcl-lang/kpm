package client

import (
	"os"
	"path/filepath"
	"testing"

	"kcl-lang.io/kpm/pkg/reporter"
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

func enableLocalRegistryPlainHTTP(t *testing.T, kpmcli *KpmClient) {
	t.Helper()

	prev, hadPrev := os.LookupEnv("OCI_REG_PLAIN_HTTP")
	if err := os.Setenv("OCI_REG_PLAIN_HTTP", "ON"); err != nil {
		t.Fatalf("failed to set OCI_REG_PLAIN_HTTP: %v", err)
	}
	t.Cleanup(func() {
		if hadPrev {
			_ = os.Setenv("OCI_REG_PLAIN_HTTP", prev)
			return
		}
		_ = os.Unsetenv("OCI_REG_PLAIN_HTTP")
	})

	_, evt := kpmcli.GetSettings().LoadSettingsFromEnv()
	if evt != (*reporter.KpmEvent)(nil) {
		t.Fatalf("failed to load settings from env: %v", evt)
	}
}
