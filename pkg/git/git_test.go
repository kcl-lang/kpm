package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateTagRef(t *testing.T) {
	assert.Equal(t, CreateTagRef("test"), "refs/tags/test")
}
