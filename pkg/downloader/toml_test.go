package downloader

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/utils"
)

const testTomlDir = "test_data_toml"

func TestMarshalTOML(t *testing.T) {
	source := &Source{
		Git: &Git{
			Url:     "https://github.com/kcl-lang/flask-demo-kcl-manifests",
			Version: "v0.1.0",
		},
	}

	got_data := source.MarshalTOML("flask-demo-kcl-manifests")

	expected_data, _ := os.ReadFile(filepath.Join(getTestDir(testTomlDir), "expected.toml"))
	expected_toml := utils.RmNewline(string(expected_data))

	fmt.Printf("expected_toml: '%q'\n", expected_toml)

	fmt.Printf("modfile: '%q'\n", got_data)
	assert.Equal(t, utils.RmNewline(expected_toml), utils.RmNewline(got_data))
}
