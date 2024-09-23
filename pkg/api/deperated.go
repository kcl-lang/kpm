package api

import (
	"path/filepath"

	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/opt"
)

// CompileWithOpt will compile the kcl program without kcl package.
// Deprecated: This method will not be maintained in the future. Use RunWithOpts instead.
func RunWithOpt(opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	// The entries will override the entries in the settings file.
	if opts.HasSettingsYaml() && len(opts.KFilenameList) > 0 && len(opts.Entries()) > 0 {
		opts.KFilenameList = []string{}
	}
	if len(opts.Entries()) > 0 {
		for _, entry := range opts.Entries() {
			if filepath.IsAbs(entry) {
				opts.Merge(kcl.WithKFilenames(entry))
			} else {
				opts.Merge(kcl.WithKFilenames(filepath.Join(opts.PkgPath(), entry)))
			}
		}
	} else if !opts.HasSettingsYaml() && len(opts.KFilenameList) == 0 {
		// If no entry, no kcl files and no settings files.
		opts.Merge(kcl.WithKFilenames(opts.PkgPath()))
	}
	opts.Merge(kcl.WithWorkDir(opts.PkgPath()))
	return kcl.RunWithOpts(*opts.Option)
}

// RunPkgWithOpt will compile the kcl package with the compile options.
// Deprecated: This method will not be maintained in the future. Use RunWithOpts instead.
func RunPkgWithOpt(opts *opt.CompileOptions) (*kcl.KCLResultList, error) {
	kpmcli, err := client.NewKpmClient()
	kpmcli.SetNoSumCheck(opts.NoSumCheck())
	if err != nil {
		return nil, err
	}
	return run(kpmcli, opts)
}
