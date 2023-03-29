// Copyright 2021 The KCL Authors. All rights reserved.

package opt

// Input options of 'kpm init'.
type InitOptions struct {
	Name     string
	InitPath string
}

type AddOptions struct {
	LocalPath    string
	RegistryOpts RegistryOption
}

type RegistryOption struct {
	Git *GitOption
}

type GitOption struct {
	Url    string
	Branch string
	Commit string
	Tag    string
}
