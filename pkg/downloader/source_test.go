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
