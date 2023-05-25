package settings

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"kusionstack.io/kpm/pkg/env"
	"kusionstack.io/kpm/pkg/utils"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func TestSettingInit(t *testing.T) {
	kpmHome, err := env.GetAbsPkgPath()
	assert.Equal(t, err, nil)
	settings, err := GetSettings()
	assert.Equal(t, err, nil)
	assert.Equal(t, settings.CredentialsFile, filepath.Join(kpmHome, CONFIG_JSON_PATH))
}

func TestGetFullJsonPath(t *testing.T) {
	path, err := GetFullJsonPath("test.json")
	assert.Equal(t, err, nil)

	kpmHome, err := env.GetAbsPkgPath()
	assert.Equal(t, err, nil)
	assert.Equal(t, path, filepath.Join(kpmHome, "test.json"))
}

func TestDefaultKpmConf(t *testing.T) {
	settings := Settings{
		Conf: DefaultKpmConf(),
	}
	assert.Equal(t, settings.DefaultOciRegistry(), "ghcr.io")
	assert.Equal(t, settings.DefaultOciRepo(), "KusionStack")
}

func TestLoadOrCreateDefaultKpmJson(t *testing.T) {
	testDir := getTestDir("expected.json")
	kpmPath := filepath.Join(filepath.Join(filepath.Join(filepath.Dir(testDir), ".kpm"), "config"), "kpm.json")
	err := os.Setenv("KCL_PKG_PATH", filepath.Dir(testDir))

	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(kpmPath), false)

	kpmConf, err := loadOrCreateDefaultKpmJson()
	assert.Equal(t, kpmConf.DefaultOciRegistry, "ghcr.io")
	assert.Equal(t, kpmConf.DefaultOciRepo, "KusionStack")
	assert.Equal(t, err, nil)
	assert.Equal(t, utils.DirExists(kpmPath), true)

	expectedJson, err := ioutil.ReadFile(testDir)
	assert.Equal(t, err, nil)

	gotJson, err := ioutil.ReadFile(kpmPath)
	assert.Equal(t, err, nil)

	var expected interface{}
	err = json.Unmarshal(expectedJson, &expected)
	assert.Equal(t, err, nil)

	var got interface{}
	err = json.Unmarshal(gotJson, &got)
	assert.Equal(t, err, nil)

	assert.Equal(t, reflect.DeepEqual(expected, got), true)

	os.RemoveAll(kpmPath)
	assert.Equal(t, utils.DirExists(kpmPath), false)
}
