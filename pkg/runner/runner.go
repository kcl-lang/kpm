package runner

import (
	"os"
	"strings"

	"kusionstack.io/kclvm-go/pkg/kcl"
	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/opt"
)

// Compiler is a wrapper of kclvm compiler.
type Compiler struct {
	kclOpts *opt.KclvmOptions
}

// NewCompiler will create a new compiler.
func NewCompiler(kclOpts *opt.KclvmOptions) *Compiler {
	return &Compiler{
		kclOpts,
	}
}

// AddDep will add a file path to the dependency list.
func (compiler *Compiler) AddDepPath(depName string, depPath string) {
	compiler.kclOpts.AddDep(depName, depPath)
}

// Call KCL Compiler and return the result.
func (compiler *Compiler) Run() (*kcl.KCLResultList, error) {
	// Parse all the kclvm options.
	kclFlags, err := ParseArgs(strings.Fields(compiler.kclOpts.KclvmCliArgs))
	if err != nil {
		return nil, err
	}

	// Transform the flags into kclvm options.
	kclOpts := kclFlags.IntoKclOptions()

	entry := compiler.kclOpts.EntryFile
	info, err := os.Stat(entry)
	if err != nil {
		return nil, err
	}

	// If the entry is a k file, then compile it directly.
	if !info.IsDir() && strings.HasSuffix(entry, ".k") {
		return kcl.Run(entry, kclOpts)
	} else if info.IsDir() {
		// If the entry is a directory, then compile all the k files in it.
		workDiropt := kcl.WithWorkDir(entry)
		workDiropt.Merge(kclOpts)
		return kcl.Run("", workDiropt)
	}
	return nil, errors.CompileFailed
}
