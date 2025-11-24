package oci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"kcl-lang.io/kpm/pkg/settings"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func TestPull(t *testing.T) {
	type TestCase struct {
		Registry string
		Image    string
		Tag      string
	}

	testCases := []TestCase{
		{"ghcr.io", "kusionstack/opsrule", "0.0.9"},
		{"ghcr.io", "kcl-lang/helloworld", "0.1.2"},
	}

	defer func() {
		err := os.RemoveAll(getTestDir("test_pull"))
		assert.Equal(t, err, nil)
	}()

	for _, tc := range testCases {
		client, err := NewOciClient(tc.Registry, tc.Image, settings.GetSettings())
		if err != nil {
			t.Fatalf("%s", err.Error())
		}

		tmpPath := filepath.Join(getTestDir("test_pull"), tc.Tag)

		err = os.MkdirAll(tmpPath, 0755)
		assert.Equal(t, err, nil)

		err = client.Pull(tmpPath, tc.Tag)
		if err != nil {
			t.Errorf("%s", err.Error())
		}
	}
}
