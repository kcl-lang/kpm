package runner

import (
	"github.com/alexflint/go-arg"
	"kusionstack.io/kclvm-go/pkg/kcl"
	"kusionstack.io/kclvm-go/pkg/spec/gpyrpc"
)

type Flag struct {
	ExternalPkg []string `arg:"-E,--external,separate"`
	Options     []string `arg:"-D,--argument,separate"`
	Overrides   []string `arg:"-O,--overrides,separate"`
	DisableNone bool     `arg:"-n,--disable-none" default:"false"`
	SortKeys    bool     `arg:"-k,--sort-keys" default:"false"`
	Settings    string   `arg:"-Y,--settings" default:""`
}

// ParseArgs parses the arguments and returns the options.
func ParseArgs(arguments []string) (Flag, error) {
	var flag Flag
	parser, err := arg.NewParser(arg.Config{}, &flag)
	if err != nil {
		return flag, err
	}
	err = parser.Parse(arguments)
	if err != nil {
		return flag, err
	}
	return flag, nil
}

// IntoKclvmOptions converts the flag into kclvm options.
func (flag *Flag) IntoKclOptions() kcl.Option {
	if flag == nil {
		return kcl.Option{}
	}

	opts := kcl.Option{
		ExecProgram_Args: new(gpyrpc.ExecProgram_Args),
	}
	if flag.ExternalPkg != nil {
		opts.Merge(kcl.WithExternalPkgs(flag.ExternalPkg...))
	}
	if flag.Options != nil {
		opts.Merge(kcl.WithOptions(flag.Options...))
	}
	if flag.Overrides != nil {
		opts.Merge(kcl.WithOverrides(flag.Overrides...))
	}

	opts.Merge(kcl.WithDisableNone(flag.DisableNone))
	opts.Merge(kcl.WithSortKeys(flag.SortKeys))

	if len(flag.Settings) != 0 {
		opts.Merge(kcl.WithSettings(flag.Settings))
	}
	return opts
}
