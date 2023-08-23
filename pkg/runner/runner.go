package runner

import (
	"fmt"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/opt"
)

// The pattern of the external package argument.
const EXTERNAL_PKGS_ARG_PATTERN = "%s=%s"

// Compiler is a wrapper of kcl compiler.
type Compiler struct {
	compileOpts *opt.CompileOptions
}

// DefaultCompiler will create a default compiler.
func DefaultCompiler() *Compiler {
	return &Compiler{
		compileOpts: opt.DefaultCompileOptions(),
	}
}

func NewCompilerWithOpts(opts *opt.CompileOptions) *Compiler {
	return &Compiler{
		compileOpts: opts,
	}
}

// AddKFile will add a k file to the compiler.
func (compiler *Compiler) AddKFile(kFilePath string) *Compiler {
	compiler.compileOpts.Merge(kcl.WithKFilenames(kFilePath))
	return compiler
}

// AddKclOption will add a kcl option to the compiler.
func (compiler *Compiler) AddKclOption(opt kcl.Option) *Compiler {
	compiler.compileOpts.Merge(opt)
	return compiler
}

// AddDep will add a file path to the dependency list.
func (compiler *Compiler) AddDepPath(depName string, depPath string) *Compiler {
	compiler.compileOpts.Merge(kcl.WithExternalPkgs(fmt.Sprintf(EXTERNAL_PKGS_ARG_PATTERN, depName, depPath)))
	return compiler
}

// Call KCL Compiler and return the result.
func (compiler *Compiler) Run() (*kcl.KCLResultList, error) {
	return kcl.RunWithOpts(*compiler.compileOpts.Option)
}
