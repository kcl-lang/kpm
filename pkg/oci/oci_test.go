package oci

import (
	"archive/tar"
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

func TestPrepareArtifactLayer(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "test-module-0.1.0.tar")
	layerFile := "main.k"
	layerContent := "value = 1\n"

	tarFile, err := os.Create(tarPath)
	assert.NoError(t, err)

	tarWriter := tar.NewWriter(tarFile)
	assert.NoError(t, tarWriter.WriteHeader(&tar.Header{
		Name: layerFile,
		Mode: 0600,
		Size: int64(len(layerContent)),
	}))
	_, err = tarWriter.Write([]byte(layerContent))
	assert.NoError(t, err)
	assert.NoError(t, tarWriter.Close())
	assert.NoError(t, tarFile.Close())

	layerPath, cleanup, err := prepareArtifactLayer(tarPath)
	assert.NoError(t, err)
	defer cleanup()

	assert.Equal(t, filepath.Join(tmpDir, "test-module-0.1.0.tgz"), layerPath)

	extractDir := filepath.Join(tmpDir, "extracted")
	assert.NoError(t, utils.ExtractTarball(layerPath, extractDir))

	got, err := os.ReadFile(filepath.Join(extractDir, layerFile))
	assert.NoError(t, err)
	assert.Equal(t, layerContent, string(got))
}
