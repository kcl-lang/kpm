// Copyright 2023 The KCL Authors. All rights reserved.

package opt

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/settings"
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
	} else if opts.RegistryOpts.Oci != nil {
		return opts.RegistryOpts.Oci.Validate()
	}
	return nil
}

type RegistryOptions struct {
	Git *GitOptions
	Oci *OciOptions
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
		return errors.InvalidAddOptionsInvalidTag
	}
	return nil
}

const DEFAULT_OCI_TAG = "latest"

func GetDefaultOCITag() string {
	return DEFAULT_OCI_TAG
}

type OciOptions struct {
	Reg     string
	Repo    string
	Tag     string
	PkgName string
}

func (opts *OciOptions) Validate() error {
	if len(opts.Repo) == 0 {
		return errors.InvalidAddOptionsInvalidOciRepo
	} else if len(opts.Tag) == 0 {
		return errors.InvalidAddOptionsInvalidTag
	}
	return nil
}

const OCI_SEPARATOR = ":"

// ParseOciOptionFromString will parser '<repo_name>:<repo_tag>' into an 'OciOptions' with an OCI registry.
// the default OCI registry is 'docker.io'.
// if the 'ociUrl' is only '<repo_name>', ParseOciOptionFromString will take 'latest' as the default tag.
func ParseOciOptionFromString(oci string, tag string) (*OciOptions, error) {
	ociOpt, err := ParseOciUrl(oci)
	if err == errors.IsOciRef {
		ociOpt, err = ParseOciRef(oci)
		if err != nil {
			return nil, err
		}
		if len(tag) != 0 {
			reporter.Report("kpm: kpm get version from oci reference '<repo_name>:<repo_tag>'")
			reporter.Report("kpm: arg '--tag' is invalid for oci reference")
		}
	} else if err == errors.NotOciUrl {
		return nil, err
	} else {
		if len(tag) == 0 {
			reporter.Report("kpm: using default tag: latest")
			ociOpt.Tag = GetDefaultOCITag()
		} else {
			ociOpt.Tag = tag
		}
	}
	return ociOpt, nil
}

// ParseOciOptionFromOciUrl will parse oci url into an 'OciOptions'.
// If the 'tag' is empty, ParseOciOptionFromOciUrl will take 'latest' as the default tag.
func ParseOciOptionFromOciUrl(url, tag string) (*OciOptions, error) {
	ociOpt, err := ParseOciUrl(url)
	if err != nil {
		return nil, err
	} else {
		if len(tag) == 0 {
			reporter.Report("kpm: using default tag: latest")
			ociOpt.Tag = GetDefaultOCITag()
		} else {
			ociOpt.Tag = tag
		}
	}
	return ociOpt, nil
}

// ParseOciRef will parse 'repoName:repoTag' into OciOptions,
// with default registry host 'docker.io'.
func ParseOciRef(ociRef string) (*OciOptions, error) {
	oci_address := strings.Split(ociRef, OCI_SEPARATOR)
	settings, err := settings.GetSettings()
	if err != nil {
		return nil, err
	}
	if len(oci_address) == 1 {
		reporter.Report("kpm: using default tag: latest")
		return &OciOptions{
			Reg:  settings.DefaultOciRegistry(),
			Repo: oci_address[0],
			Tag:  GetDefaultOCITag(),
		}, nil
	} else if len(oci_address) == 2 {
		return &OciOptions{
			Reg:  settings.DefaultOciRegistry(),
			Repo: oci_address[0],
			Tag:  oci_address[1],
		}, nil
	} else {
		return nil, errors.InvalidOciRef
	}
}

// ParseOciUrl will parse 'oci://hostName/repoName:repoTag' into OciOptions without tag.
func ParseOciUrl(ociUrl string) (*OciOptions, error) {
	u, err := url.Parse(ociUrl)
	if err != nil {
		return nil, errors.IsOciRef
	}

	if len(u.Scheme) != 0 && u.Scheme != "oci" {
		return nil, errors.NotOciUrl
	} else if len(u.Scheme) == 0 {
		return nil, errors.IsOciRef
	}

	return &OciOptions{
		Reg:  u.Host,
		Repo: u.Path,
	}, nil
}

// AddStoragePathSuffix will take 'Registry/Repo/Tag' as a path suffix.
// e.g. Take '/usr/test' as input,
// and oci options is
//
//	OciOptions {
//	  Reg: 'docker.io',
//	  Repo: 'test/testRepo',
//	  Tag: 'v0.0.1'
//	}
//
// You will get a path '/usr/test/docker.io/test/testRepo/v0.0.1'.
func (oci *OciOptions) AddStoragePathSuffix(pathPrefix string) string {
	return filepath.Join(filepath.Join(filepath.Join(pathPrefix, oci.Reg), oci.Repo), oci.Tag)
}

// The parameters needed to compile the kcl program.
type KclvmOptions struct {
	Deps         map[string]string
	EntryFiles   []string
	KclvmCliArgs string
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
	if len(kclOpts.KclvmCliArgs) != 0 {
		args = append(args, strings.Split(kclOpts.KclvmCliArgs, " ")...)
	}

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
