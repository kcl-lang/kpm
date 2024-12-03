package client

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// The PullOptions struct contains the options for pulling a package from the registry.
type PullOptions struct {
	// Source is the source of the package to be pulled.
	// Including git, oci, local.
	Source *downloader.Source
	// LocalPath is the local path to download the package.
	LocalPath string
}

type PullOption func(*PullOptions) error

func WithPullModSpec(modSpec *downloader.ModSpec) PullOption {
	return func(opts *PullOptions) error {
		if modSpec == nil {
			return errors.New("modSpec cannot be nil")
		}
		if opts.Source == nil {
			opts.Source = &downloader.Source{
				ModSpec: modSpec,
			}
		} else {
			opts.Source.ModSpec = modSpec
		}

		return nil
	}
}

func WithPullSourceUrl(sourceUrl string) PullOption {
	return func(opts *PullOptions) error {
		source, err := downloader.NewSourceFromStr(sourceUrl)
		if err != nil {
			return err
		}
		opts.Source = source
		return nil
	}
}

func WithPullSource(source *downloader.Source) PullOption {
	return func(opts *PullOptions) error {
		if source == nil {
			return errors.New("source cannot be nil")
		}
		opts.Source = source
		return nil
	}
}

func WithLocalPath(path string) PullOption {
	return func(opts *PullOptions) error {
		opts.LocalPath = path
		return nil
	}
}

func NewPullOptions(opts ...PullOption) *PullOptions {
	do := &PullOptions{}
	for _, opt := range opts {
		opt(do)
	}
	return do
}

func (c *KpmClient) Pull(options ...PullOption) (*pkg.KclPkg, error) {
	opts := &PullOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	if opts.Source.SpecOnly() {
		opts.Source.Oci = &downloader.Oci{
			Reg:  c.GetSettings().DefaultOciRegistry(),
			Repo: utils.JoinPath(c.GetSettings().DefaultOciRepo(), opts.Source.ModSpec.Name),
			Tag:  opts.Source.ModSpec.Version,
		}
	}

	sourceFilePath, err := opts.Source.ToFilePath()
	if err != nil {
		return nil, err
	}

	sourceStr, err := opts.Source.ToString()
	if err != nil {
		return nil, err
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("start to pull %s", sourceStr),
		c.GetLogWriter(),
	)

	pkgSource := opts.Source
	pulledFullPath := filepath.Join(opts.LocalPath, sourceFilePath)

	err = newVisitor(*pkgSource, c).Visit(pkgSource, func(kPkg *pkg.KclPkg) error {
		if !utils.DirExists(filepath.Dir(pulledFullPath)) {
			err := os.MkdirAll(filepath.Dir(pulledFullPath), os.ModePerm)
			if err != nil {
				return err
			}
		}
		err := utils.MoveOrCopy(kPkg.HomePath, pulledFullPath)
		if err != nil {
			return err
		}
		reporter.ReportMsgTo(
			fmt.Sprintf("pulled %s %s successfully", kPkg.GetPkgName(), kPkg.GetPkgVersion()),
			c.GetLogWriter(),
		)
		return nil
	})

	if err != nil {
		return nil, err
	}

	kPkg, err := pkg.LoadKclPkgWithOpts(
		pkg.WithPath(pulledFullPath),
		pkg.WithSettings(c.GetSettings()),
	)

	if err != nil {
		return nil, err
	}

	return kPkg, nil
}
