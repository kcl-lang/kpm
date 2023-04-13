package runner

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/opt"
)

func TestCompilerNotFound(t *testing.T) {
	// Set the entry file into compile options.
	compileOpts := opt.NewKclvmOpts()

	// Call the kclvm_cli.
	compiler, err := NewCompileCmd(compileOpts)
	assert.Equal(t, err, nil)
	compiler.cmd.Path = ""
	result := compiler.Run()
	assert.Equal(t, result, "")
}

func TestCompilerValidate(t *testing.T) {
	os.Setenv("PATH", "")
	// Set the entry file into compile options.
	compileOpts := opt.NewKclvmOpts()

	// Call the kclvm_cli.
	compiler, err := NewCompileCmd(compileOpts)
	assert.Equal(t, err, errors.CompileFailed)
	err = compiler.Validate()
	assert.Equal(t, err, errors.CompileFailed)
}
