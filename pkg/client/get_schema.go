package client

import pkg "kcl-lang.io/kpm/pkg/package"

type GetSchemaTypeOptions struct {
	kmod *pkg.KclPkg
}

type GetSchemaTypeOption func(*GetSchemaTypeOptions) error

func WithKmod(kmod *pkg.KclPkg) GetSchemaTypeOption {
	return func(opts *GetSchemaTypeOptions) error {
		opts.kmod = kmod
		return nil
	}
}

func (c *KpmClient) GetSchemaType(options ...GetSchemaTypeOption) error {
	opts := &GetSchemaTypeOptions{}
	for _, option := range options {
		if err := option(opts); err != nil {
			return err
		}
	}

	// Do something to get schema type
	return nil
}
