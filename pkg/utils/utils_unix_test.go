//go:build linux || darwin
// +build linux darwin

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateSymbolLink(t *testing.T) {
	testDir := getTestDir("test_link")
	need_linked := filepath.Join(testDir, "need_be_linked_v1")
	linkPath := filepath.Join(testDir, "linked")

	_ = os.Remove(linkPath)
	err := CreateSymlink(need_linked, linkPath)

	linkTarget, _ := os.Readlink(linkPath)
	assert.Equal(t, err, nil)
	assert.Equal(t, linkTarget, need_linked)
	_ = os.Remove(linkPath)
}
