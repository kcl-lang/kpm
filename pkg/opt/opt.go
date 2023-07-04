// Copyright 2023 The KCL Authors. All rights reserved.

package opt

import (
	"net/url"
	"path/filepath"
	"strings"

	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
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
	}
	return nil
}

// OciOptions for download oci packages.
// kpm will download packages from oci registry by '{Reg}/{Repo}/{PkgName}:{Tag}'.
type OciOptions struct {
	Reg     string
	Repo    string
	Tag     string
	PkgName string
}

func (opts *OciOptions) Validate() error {
	if len(opts.Repo) == 0 {
		return errors.InvalidAddOptionsInvalidOciRepo
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
		return ociOpt, nil
	} else if err == errors.NotOciUrl {
		return nil, err
	}
	ociOpt.Tag = tag

	return ociOpt, nil
}

// ParseOciOptionFromOciUrl will parse oci url into an 'OciOptions'.
// If the 'tag' is empty, ParseOciOptionFromOciUrl will take 'latest' as the default tag.
func ParseOciOptionFromOciUrl(url, tag string) (*OciOptions, error) {
	ociOpt, err := ParseOciUrl(url)
	if err != nil {
		return nil, err
	}
	ociOpt.Tag = tag
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
		return &OciOptions{
			Reg:  settings.DefaultOciRegistry(),
			Repo: oci_address[0],
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

	if u.Scheme != "oci" {
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
