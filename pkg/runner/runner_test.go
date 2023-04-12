package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"kusionstack.io/kpm/pkg/opt"
)

func TestCompilerNotFound(t *testing.T) {
	// Set the entry file into compile options.
	compileOpts := opt.NewKclvmOpts()

	// Call the kclvm_cli.
	compiler := NewCompileCmd(compileOpts)
	compiler.cmd.Path = ""
	result := compiler.Run()
	assert.Equal(t, result, "")
}
