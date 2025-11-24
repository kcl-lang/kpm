package runner

import (
	"fmt"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kcl-go/scripts"
	"kcl-lang.io/kpm/pkg/opt"
)

// The pattern of the external package argument.
const EXTERNAL_PKGS_ARG_PATTERN = "%s=%s"

// Compiler is a wrapper of kcl compiler.
type Compiler struct {
	opts *opt.CompileOptions
}

// DefaultCompiler will create a default compiler.
func DefaultCompiler() *Compiler {
	return &Compiler{
		opts: opt.DefaultCompileOptions(),
	}
}

func NewCompilerWithOpts(opts *opt.CompileOptions) *Compiler {
	return &Compiler{
		opts,
	}
}

// AddKFile will add a k file to the compiler.
func (compiler *Compiler) AddKFile(kFilePath string) *Compiler {
	compiler.opts.Merge(kcl.WithKFilenames(kFilePath))
	return compiler
}

// AddKclOption will add a kcl option to the compiler.
func (compiler *Compiler) AddKclOption(opt kcl.Option) *Compiler {
	compiler.opts.Merge(opt)
	return compiler
}

// AddDep will add a file path to the dependency list.
func (compiler *Compiler) AddDepPath(depName string, depPath string) *Compiler {
	compiler.opts.Merge(kcl.WithExternalPkgs(fmt.Sprintf(EXTERNAL_PKGS_ARG_PATTERN, depName, depPath)))
	return compiler
}

// Call KCL Compiler and return the result.
func (compiler *Compiler) Run() (*kcl.KCLResultList, error) {
	return kcl.RunWithOpts(*compiler.opts.Option)
}

// GetKclVersion fetches the kcl version
func GetKclVersion() string {
	return string(scripts.KclVersionType_latest)
}
