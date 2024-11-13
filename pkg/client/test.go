package client

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
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

// Use a global variable to store the kpmcli instance.
func RunTestWithGlobalLockAndKpmCli(t *testing.T, name string, testFunc func(t *testing.T, kpmcli *KpmClient)) {
	t.Run(name, func(t *testing.T) {
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

		testFunc(t, kpmcli)
		fmt.Printf("%s completed\n", name)
	})
}
