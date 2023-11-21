package oci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func TestLogin(t *testing.T) {
	testPath := getTestDir("test_login")
	testConfPath := filepath.Join(testPath, "config.json")

	// clean the test dir
	if utils.DirExists(testConfPath) {
		os.Remove(testConfPath)
	}

	settings := settings.Settings{
		CredentialsFile: testConfPath,
	}

	hostName := "ghcr.io"
	userName := "invalid_username"
	userPwd := "invalid_password"

	err := Login(hostName, userName, userPwd, &settings)
	assert.Equal(t, err.Error(), "error: failed to login 'ghcr.io', please check registry, username and password is valid\nerror: Get \"https://ghcr.io/v2/\": denied: denied\n")
}
