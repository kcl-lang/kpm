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
	assert.Equal(t, err.Error(), "failed to login 'ghcr.io', please check registry, username and password is valid\nGet \"https://ghcr.io/v2/\": denied: denied\n")
}

func TestPull(t *testing.T) {
	type TestCase struct {
		Registry string
		Image    string
		Tag      string
	}

	testCases := []TestCase{
		{"ghcr.io", "kusionstack/opsrule", "0.0.9"},
		{"ghcr.io", "kcl-lang/helloworld", "0.1.1"},
	}

	defer func() {
		err := os.RemoveAll(getTestDir("test_pull"))
		assert.Equal(t, err, nil)
	}()

	for _, tc := range testCases {
		client, err := NewOciClient(tc.Registry, tc.Image, settings.GetSettings())
		if err != nil {
			t.Fatalf(err.Error())
		}

		tmpPath := filepath.Join(getTestDir("test_pull"), tc.Tag)

		err = os.MkdirAll(tmpPath, 0755)
		assert.Equal(t, err, nil)

		err = client.Pull(tmpPath, tc.Tag)
		if err != nil {
			t.Errorf(err.Error())
		}
	}
}
