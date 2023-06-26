package runner

import (
	"fmt"
	"strings"

	"kusionstack.io/kclvm-go/pkg/kcl"
)

// The pattern of the external package argument.
const EXTERNAL_PKGS_ARG_PATTERN = "%s=%s"

// Compiler is a wrapper of kcl compiler.
type Compiler struct {
	kclCliArgs string
	kclOpts    *kcl.Option
}

// DefaultCompiler will create a default compiler.
func DefaultCompiler() *Compiler {
	return &Compiler{
		kclOpts: kcl.NewOption(),
	}
}

// AddKFile will add a k file to the compiler.
func (compiler *Compiler) AddKFile(kFilePath string) *Compiler {
	compiler.kclOpts.Merge(kcl.WithKFilenames(kFilePath))
	return compiler
}

// AddKclOption will add a kcl option to the compiler.
func (compiler *Compiler) AddKclOption(opt kcl.Option) *Compiler {
	compiler.kclOpts.Merge(opt)
	return compiler
}

// AddDep will add a file path to the dependency list.
func (compiler *Compiler) AddDepPath(depName string, depPath string) *Compiler {
	compiler.kclOpts.Merge(kcl.WithExternalPkgs(fmt.Sprintf(EXTERNAL_PKGS_ARG_PATTERN, depName, depPath)))
	return compiler
}

// SetKclCliArgs will set the kcl cli arguments.
func (compiler *Compiler) SetKclCliArgs(kclCliArgs string) *Compiler {
	compiler.kclCliArgs = kclCliArgs
	return compiler
}

// Call KCL Compiler and return the result.
func (compiler *Compiler) Run() (*kcl.KCLResultList, error) {
	// Parse all the kcl options.
	kclFlags, err := ParseArgs(strings.Fields(compiler.kclCliArgs))
	if err != nil {
		return nil, err
	}

	// Merge the kcl options from kcl.mod and kpm cli.
	compiler.kclOpts.Merge(kclFlags.IntoKclOptions())

	return kcl.RunWithOpts(*compiler.kclOpts)
}
