package path

import (
	"runtime"
	"testing"
)

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Path with null character",
			input:    "test\x00file",
			expected: "test_file",
		},
	}

	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name     string
			input    string
			expected string
		}{
			name:     "Windows style path",
			input:    "C:\\Program Files\\Test<:>*|",
			expected: "C:\\Program Files\\Test_____",
		},
		)
	} else {
		tests = append(tests, struct {
			name     string
			input    string
			expected string
		}{
			name:     "Path without invalid characters",
			input:    "/usr/local/bin/test",
			expected: "/usr/local/bin/test",
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := SanitizePath(tt.input)
			if output != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, output)
			}
		})
	}
}
