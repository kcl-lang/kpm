package runner

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kcl-go/pkg/kcl"
)

func TestKclRun(t *testing.T) {
	absPath, err := filepath.Abs("./testdata_external/external/")
	assert.Equal(t, err, nil)
	absPath1, err := filepath.Abs("./testdata_external/external_1/")
	assert.Equal(t, err, nil)
	opt := kcl.WithExternalPkgs("external="+absPath, "external_1="+absPath1)
	result, err := kcl.Run("./testdata/import_external.k", opt)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "{\"a\":\"Hello External World!\",\"a1\":\"Hello External_1 World!\"}\n", result.GetRawJsonResult())
	assert.Equal(t, "a: Hello External World!\na1: Hello External_1 World!", result.GetRawYamlResult())
}
