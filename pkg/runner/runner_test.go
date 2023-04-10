package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/opt"
)

func TestCompilerNotFound(t *testing.T) {
	// Set the entry file into compile options.
	compileOpts := opt.NewKclvmOpts()

	// Call the kclvm_cli.
	compiler := NewCompileCmd(compileOpts)
	compiler.cmd.Env = make([]string, 0)
	result, err := compiler.Run()
	assert.Equal(t, err.Error(), errors.CompileFailed.Error())
	assert.Equal(t, result, "")
}
