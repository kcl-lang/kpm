package client

import (
	"fmt"
	"os"

	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// PushOptions contains the options for the Push method.
type PushOptions struct {
	// By default, the registry, repository, reference and tag are set to the default values
	// registry is set to the default registry `ghcr.io` in the settings
	// repository is set to the default repository `kcl-lang` in the settings
	// reference is set to the package name in `kcl.mod`
	// tag is set to the package version in `kcl.mod`
	// Force is set to false by default
	Source     downloader.Source
	ModPath    string
	VendorMode bool
	Force      bool
}

type PushOption func(*PushOptions) error

// WithPushSource sets the source for the Push method.
func WithPushSource(source downloader.Source) PushOption {
	return func(opts *PushOptions) error {
		opts.Source = source
		return nil
	}
}

// WithPushVendorMode sets the vendor mode for the Push method.
func WithPushVendorMode(vendorMode bool) PushOption {
	return func(opts *PushOptions) error {
		opts.VendorMode = vendorMode
		return nil
	}
}

// WithPushModPath sets the modPath for the Push method.
func WithPushModPath(modPath string) PushOption {
	return func(opts *PushOptions) error {
		if modPath == "" {
			return fmt.Errorf("modPath cannot be empty")
		}
		opts.ModPath = modPath
		return nil
	}
}

// WithPushForce sets the Force option for the Push method.
func WithPushForce(allowForce bool) PushOption {
	return func(opts *PushOptions) error {
		opts.Force = allowForce
		return nil
	}
}

// fillDefaultPushOptions will fill the default values for the PushOptions.
func (c *KpmClient) fillDefaultPushOptions(ociOpt *opt.OciOptions, kMod *pkg.KclPkg) {
	if ociOpt.Reg == "" {
		ociOpt.Reg = c.GetSettings().DefaultOciRegistry()
	}

	if ociOpt.Repo == "" {
		ociOpt.Repo = c.GetSettings().DefaultOciRepo()
	}

	if ociOpt.Tag == "" {
		ociOpt.Tag = kMod.ModFile.Pkg.Version
	}
}

// Push will push a kcl package to a registry.
func (c *KpmClient) Push(opts ...PushOption) error {
	pushOpts := &PushOptions{}
	for _, opt := range opts {
		if err := opt(pushOpts); err != nil {
			return err
		}
	}

	kMod, err := pkg.LoadKclPkgWithOpts(
		pkg.WithPath(pushOpts.ModPath),
		pkg.WithSettings(c.GetSettings()),
	)

	if err != nil {
		return err
	}

	source := pushOpts.Source
	ociUrl, err := source.ToString()
	if err != nil {
		return err
	}
	if source.Oci == nil {
		return fmt.Errorf("'%s' is not an oci source, only support oci source", ociUrl)
	}

	var tag string
	if source.Oci.Tag == "" {
		tag = kMod.ModFile.Pkg.Version
	} else {
		tag = source.Oci.Tag
	}

	ociOpts, err := c.ParseOciOptionFromString(ociUrl, tag)
	if err != nil {
		return err
	}
	c.fillDefaultPushOptions(ociOpts, kMod)

	ociOpts.Annotations, err = kMod.GenOciManifestFromPkg()
	if err != nil {
		return err
	}

	tarPath, err := c.PackagePkg(kMod, pushOpts.VendorMode)
	if err != nil {
		return err
	}

	// clean the tar path.
	defer func() {
		if kMod != nil && utils.DirExists(tarPath) {
			err = os.RemoveAll(tarPath)
			if err != nil {
				err = errors.InternalBug
			}
		}
	}()

	reporter.ReportMsgTo(fmt.Sprintf("package '%s' will be pushed", kMod.GetPkgName()), c.GetLogWriter())
	return c.pushToOci(tarPath, ociOpts, pushOpts)
}

// PushToOci will push a kcl package to oci registry.
func (c *KpmClient) pushToOci(localPath string, ociOpts *opt.OciOptions, pushOpts *PushOptions) error {
	repoPath := utils.JoinPath(ociOpts.Reg, ociOpts.Repo, ociOpts.Ref)
	cred, err := c.GetCredentials(ociOpts.Reg)
	if err != nil {
		return err
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
		oci.WithInsecureSkipTLSverify(c.insecureSkipTLSverify),
	)

	if err != nil {
		return err
	}

	ociCli.SetLogWriter(c.logWriter)

	exist, err := ociCli.ContainsTag(ociOpts.Tag)
	if err != (*reporter.KpmEvent)(nil) {
		return err
	}

	if exist {
		// Allow force when explicitly allowed
		if pushOpts.Force {
			reporter.ReportMsgTo(fmt.Sprintf("package version '%s' already exists, force pushing", ociOpts.Tag), c.GetLogWriter())
		} else {
			return reporter.NewErrorEvent(
				reporter.PkgTagExists,
				fmt.Errorf("package version '%s' already exists", ociOpts.Tag),
			)
		}
	}

	return ociCli.PushWithOciManifest(localPath, ociOpts.Tag, &opt.OciManifestOptions{
		Annotations: ociOpts.Annotations,
	})
}
