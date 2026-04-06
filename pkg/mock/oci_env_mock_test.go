package mock

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoRoot(t *testing.T) {
	root, err := repoRoot()
	assert.NoError(t, err)

	assert.DirExists(t, root)
	assert.FileExists(t, filepath.Join(root, "go.mod"))
}

func TestRepoScriptCommand(t *testing.T) {
	root, err := repoRoot()
	assert.NoError(t, err)

	tests := []struct {
		name  string
		parts []string
	}{
		{
			name:  "registry setup script",
			parts: []string{"scripts", "reg.sh"},
		},
		{
			name:  "registry cleanup script",
			parts: []string{"pkg", "mock", "test_script", "cleanup_test_environment.sh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := repoScriptCommand(tt.parts...)
			assert.NoError(t, err)
			assert.Equal(t, root, cmd.Dir)
			assert.Equal(t, filepath.Join(append([]string{root}, tt.parts...)...), cmd.Path)
		})
	}
}
