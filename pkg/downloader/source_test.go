package downloader

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseModSpecFromStr(t *testing.T) {
	tests := []struct {
		input    string
		expected ModSpec
	}{
		{"subhelloworld:0.0.1", ModSpec{Name: "subhelloworld", Version: "0.0.1"}},
		{"subhelloworld", ModSpec{Name: "subhelloworld", Version: ""}},
		{"subhelloworld:", ModSpec{Name: "subhelloworld", Version: ""}},
		{":0.0.1", ModSpec{Name: "", Version: "0.0.1"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			modspec := ModSpec{}
			err := modspec.FromString(tt.input)
			assert.NilError(t, err)
			assert.Equal(t, modspec.Name, tt.expected.Name)
			assert.Equal(t, modspec.Version, tt.expected.Version)
		})
	}
}

func TestLocalArchiveSourceDetection(t *testing.T) {
	tarSource := &Source{Local: &Local{Path: "/tmp/test.tar"}}
	assert.Equal(t, tarSource.IsLocalTarPath(), true)
	assert.Equal(t, tarSource.IsLocalTgzPath(), false)
	assert.Equal(t, tarSource.IsPackaged(), true)

	tgzSource := &Source{Local: &Local{Path: "/tmp/test.tgz"}}
	assert.Equal(t, tgzSource.IsLocalTarPath(), false)
	assert.Equal(t, tgzSource.IsLocalTgzPath(), true)
	assert.Equal(t, tgzSource.IsPackaged(), true)
}
