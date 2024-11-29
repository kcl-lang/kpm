package client

import (
	"fmt"

	"kcl-lang.io/kpm/pkg/checker"
	pkg "kcl-lang.io/kpm/pkg/package"
)

type CheckOptions struct {
	KclMod *pkg.KclPkg
}

type CheckOption func(*CheckOptions) error

func WithCheckKclMod(kclMod *pkg.KclPkg) CheckOption {
	return func(opts *CheckOptions) error {
		if kclMod == nil {
			return fmt.Errorf("kclMod cannot be nil")
		}
		opts.KclMod = kclMod
		return nil
	}
}

func (c *KpmClient) Check(options ...CheckOption) error {
	opts := &CheckOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return err
		}
	}

	kmod := opts.KclMod
	if kmod == nil {
		return fmt.Errorf("kclMod cannot be nil")
	}

	// Init the ModChecker, name and version checkers are required.
	if c.ModChecker == nil || c.ModChecker.CheckersSize() == 0 {
		c.ModChecker = checker.NewModChecker(
			checker.WithCheckers(
				checker.NewIdentChecker(),
				checker.NewVersionChecker(),
				checker.NewSumChecker(),
			),
		)
	}

	// Check the module and the dependencies
	err := c.ModChecker.Check(*kmod)
	if err != nil {
		return err
	}

	return err
}
