// Copyright 2023 The KCL Authors. All rights reserved.

package opt

import (
	"fmt"

	"kusionstack.io/kpm/pkg/errors"
)

// Input options of 'kpm init'.
type InitOptions struct {
	Name     string
	InitPath string
}

func (opts *InitOptions) Validate() error {
	if len(opts.Name) == 0 {
		return errors.InvalidInitOptions
	} else if len(opts.InitPath) == 0 {
		return errors.InternalBug
	}
	return nil
}

type AddOptions struct {
	LocalPath    string
	RegistryOpts RegistryOptions
}

func (opts *AddOptions) Validate() error {
	if len(opts.LocalPath) == 0 {
		return errors.InternalBug
	} else if opts.RegistryOpts.Git != nil {
		return opts.RegistryOpts.Git.Validate()
	} else if opts.RegistryOpts.Git == nil {
		return errors.InvalidAddOptionsWithoutRegistry
	}
	return nil
}

type RegistryOptions struct {
	Git *GitOptions
}

type GitOptions struct {
	Url    string
	Branch string
	Commit string
	Tag    string
}

func (opts *GitOptions) Validate() error {
	if len(opts.Url) == 0 {
		return errors.InvalidAddOptionsInvalidGitUrl
	} else if len(opts.Tag) == 0 {
		return errors.InvalidAddOptionsInvalidGitTag
	}
	return nil
}

// The parameters needed to compile the kcl program.
type KclvmOptions struct {
	Deps       map[string]string
	EntryFiles []string
	// todo: add all kclvm options.
}

func (opts *KclvmOptions) Validate() error {
	if len(opts.EntryFiles) == 0 {
		return errors.InvalidRunOptionsWithoutEntryFiles
	}
	return nil
}

func NewKclvmOpts() *KclvmOptions {
	return &KclvmOptions{
		Deps:       make(map[string]string),
		EntryFiles: make([]string, 0),
	}
}

// Generate the kcl compile command arguments based on 'KclvmOptions'.
func (kclOpts *KclvmOptions) Args() []string {
	var args []string
	args = append(args, kclOpts.EntryFiles...)
	args = append(args, kclOpts.PkgPathMapArgs()...)
	return args
}

const EXTERNAL_ARG = "-E"
const EXTERNAL_ARG_PATTERN = "%s=%s"

// Generate the kcl compile command arguments '-E <pkg_name>=<pkg_path>'.
func (kclOpts *KclvmOptions) PkgPathMapArgs() []string {
	var args []string
	for k, v := range kclOpts.Deps {
		args = append(args, EXTERNAL_ARG)
		args = append(args, fmt.Sprintf(EXTERNAL_ARG_PATTERN, k, v))
	}
	return args
}
