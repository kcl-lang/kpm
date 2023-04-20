package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSettingInit(t *testing.T) {
	home, _ := os.UserHomeDir()
	settings, err := Init()
	assert.Equal(t, err, nil)
	assert.Equal(t, settings.CredentialsFile, filepath.Join(home, CONFIG_JSON_PATH))
}
