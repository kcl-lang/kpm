package runner

import (
	"fmt"
	"os/exec"

	"kusionstack.io/kpm/pkg/opt"
)

// CompileCmd denotes a KCL Compiler,
// it will call the kcl compiler with the command 'kclvm_cli'.
// If 'kclvm_cli' is not found in the environment variable,
// CompileCmd will not compile KCL properly and will throw an error.
type CompileCmd struct {
	kclOpts *opt.KclvmOptions
	cmd     *exec.Cmd
}

const KCLVM_CLI = "kclvm_cli"
const KCLVM_COMMAND_RUN = "run"

func NewCompileCmd(kclOpts *opt.KclvmOptions) *CompileCmd {
	return &CompileCmd{
		kclOpts: kclOpts,
		cmd:     exec.Command(KCLVM_CLI),
	}
}

func (cmd *CompileCmd) AddDepPath(depName string, depPath string) {
	cmd.kclOpts.Deps[depName] = depPath
}

// Call KCL Compiler and return the result.
func (cmd *CompileCmd) Run() string {
	var args []string
	args = append(args, KCLVM_COMMAND_RUN)
	args = append(args, cmd.kclOpts.Args()...)
	fmt.Printf("args: %v\n", args)
	cmd.cmd.Args = append(cmd.cmd.Args, args...)
	out, _ := cmd.cmd.CombinedOutput()
	return string(out)
}
