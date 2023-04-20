package settings

import (
	"os"
	"path/filepath"

	"kusionstack.io/kpm/pkg/errors"
)

// The config.json used to persist user information
const CONFIG_JSON_PATH = ".kpm/config/config.json"

type Settings struct {
	CredentialsFile string
}

// GetConfigJsonPath returns config.json file path under '$HOME/.kpm/config/config.json'
func GetConfigJsonPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.InternalBug
	}

	return filepath.Join(home, CONFIG_JSON_PATH), nil
}

// Init returns default kpm settings.
func Init() (*Settings, error) {
	credentialsFile, err := GetConfigJsonPath()
	if err != nil {
		return nil, err
	}
	return &Settings{
		CredentialsFile: credentialsFile,
	}, nil
}
