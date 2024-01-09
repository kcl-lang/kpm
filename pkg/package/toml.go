// Copyright 2022 The KCL Authors. All rights reserved.
//
// Because the same dependency package will be serialized
// into toml files in different formats in kcl.mod and kcl.mod.lock,
// the toml library 'github.com/BurntSushi/toml' is encapsulated in this file,
// and two different format are provided according to different files.
//
// In kcl.mod, the dependency toml looks like:
//
// <dependency_name> = { git = "<git_url>", tag = "<git_tag>" }
//
// In kcl.mod.lock, the dependency toml looks like:
//
// [dependencies.<dependency_name>]
// name = "<dependency_name>"
// full_name = "<dependency_fullname>"
// version = "<dependency_version>"
// sum = "yNADGqn3jclWtfpwvWMHBsgkAKzOaMWg/VYxfcOJs64="
// url = "https://github.com/xxxx"
// tag = "<dependency_tag>"
package pkg

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"kcl-lang.io/kpm/pkg/reporter"
)

const NEWLINE = "\n"

func (mod *ModFile) MarshalTOML() string {
	var sb strings.Builder
	sb.WriteString(mod.Pkg.MarshalTOML())
	sb.WriteString(mod.Dependencies.MarshalTOML())
	sb.WriteString(mod.Profiles.MarshalTOML())
	return sb.String()
}

const PACKAGE_PATTERN = "[package]"

func (pkg *Package) MarshalTOML() string {
	var sb strings.Builder
	sb.WriteString(PACKAGE_PATTERN)
	sb.WriteString(NEWLINE)
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(pkg); err != nil {
		fmt.Println(err)
		return ""
	}
	sb.WriteString(buf.String())
	sb.WriteString(NEWLINE)
	return sb.String()
}

const DEPS_PATTERN = "[dependencies]"

func (dep *Dependencies) MarshalTOML() string {
	var sb strings.Builder
	if len(dep.Deps) != 0 {
		sb.WriteString(DEPS_PATTERN)
		for _, dep := range dep.Deps {
			sb.WriteString(NEWLINE)
			sb.WriteString(dep.MarshalTOML())
		}
		sb.WriteString(NEWLINE)
	}
	return sb.String()
}

const DEP_PATTERN = "%s = %s"

func (dep *Dependency) MarshalTOML() string {
	source := dep.Source.MarshalTOML()
	var sb strings.Builder
	if len(source) != 0 {
		sb.WriteString(fmt.Sprintf(DEP_PATTERN, dep.Name, source))
	}
	return sb.String()
}

const SOURCE_PATTERN = "{ %s }"

func (source *Source) MarshalTOML() string {
	var sb strings.Builder
	if source.Git != nil {
		gitToml := source.Git.MarshalTOML()
		if len(gitToml) != 0 {
			sb.WriteString(fmt.Sprintf(SOURCE_PATTERN, gitToml))
		}
	}

	if source.Oci != nil {
		ociToml := source.Oci.MarshalTOML()
		if len(ociToml) != 0 {
			sb.WriteString(ociToml)
		}
	}

	if source.Local != nil {
		localPathToml := source.Local.MarshalTOML()
		if len(localPathToml) != 0 {
			sb.WriteString(fmt.Sprintf(SOURCE_PATTERN, localPathToml))
		}
	}

	return sb.String()
}

const GTI_URL_PATTERN = "git = \"%s\""
const GTI_TAG_PATTERN = "tag = \"%s\""
const GTI_COMMIT_PATTERN = "commit = \"%s\""
const SEPARATOR = ", "

func (git *Git) MarshalTOML() string {
	var sb strings.Builder
	if len(git.Url) != 0 {
		sb.WriteString(fmt.Sprintf(GTI_URL_PATTERN, git.Url))
	}
	if len(git.Tag) != 0 {
		sb.WriteString(SEPARATOR)
		sb.WriteString(fmt.Sprintf(GTI_TAG_PATTERN, git.Tag))
	}
	if len(git.Commit) != 0 {
		sb.WriteString(SEPARATOR)
		sb.WriteString(fmt.Sprintf(GTI_COMMIT_PATTERN, git.Commit))
	}
	return sb.String()
}

func (oci *Oci) MarshalTOML() string {
	var sb strings.Builder
	if len(oci.Tag) != 0 {
		sb.WriteString(fmt.Sprintf(`"%s"`, oci.Tag))
	}
	return sb.String()
}

const LOCAL_PATH_PATTERN = "path = %s"

func (local *Local) MarshalTOML() string {
	var sb strings.Builder
	if len(local.Path) != 0 {
		sb.WriteString(fmt.Sprintf(LOCAL_PATH_PATTERN, fmt.Sprintf("%q", local.Path)))
	}
	return sb.String()
}

const PROFILE_PATTERN = "[profile]"

func (p *Profile) MarshalTOML() string {
	var sb strings.Builder
	if p != nil {
		sb.WriteString(PROFILE_PATTERN)
		sb.WriteString(NEWLINE)
		var buf bytes.Buffer
		if err := toml.NewEncoder(&buf).Encode(p); err != nil {
			fmt.Println(err)
			return ""
		}
		sb.WriteString(buf.String())
		sb.WriteString(NEWLINE)
	}
	return sb.String()
}

const PACKAGE_FLAG = "package"
const DEPS_FLAG = "dependencies"
const PROFILES_FLAG = "profile"

func (mod *ModFile) UnmarshalTOML(data interface{}) error {
	meta, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map[string]interface{}, got %T", data)
	}

	if v, ok := meta[PACKAGE_FLAG]; ok {
		pkg := Package{}
		err := pkg.UnmarshalTOML(v)
		if err != nil {
			return err
		}
		mod.Pkg = pkg
	}

	if v, ok := meta[DEPS_FLAG]; ok {
		deps := Dependencies{
			Deps: make(map[string]Dependency),
		}
		err := deps.UnmarshalModTOML(v)
		if err != nil {
			return err
		}
		mod.Dependencies = deps
	}

	if v, ok := meta[PROFILES_FLAG]; ok {
		p := NewProfile()
		var buf bytes.Buffer
		if err := toml.NewEncoder(&buf).Encode(v); err != nil {
			return err
		}
		err := toml.Unmarshal(buf.Bytes(), &p)

		if err != nil {
			return err
		}
		mod.Profiles = &p
	}
	return nil
}

const NAME_FLAG = "name"
const EDITION_FLAG = "edition"
const VERSION_FLAG = "version"
const DESCRIPTION_FLAG = "description"

func (pkg *Package) UnmarshalTOML(data interface{}) error {
	meta, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map[string]interface{}, got %T", data)
	}

	if v, ok := meta[NAME_FLAG].(string); ok {
		pkg.Name = v
	}

	if v, ok := meta[EDITION_FLAG].(string); ok {
		pkg.Edition = v
	}

	if v, ok := meta[VERSION_FLAG].(string); ok {
		pkg.Version = v
	}

	if v, ok := meta[DESCRIPTION_FLAG].(string); ok {
		pkg.Description = v
	}
	return nil
}

func (deps *Dependencies) UnmarshalModTOML(data interface{}) error {
	meta, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map[string]interface{}, got %T", data)
	}

	for k, v := range meta {
		dep := Dependency{}
		dep.Name = k

		err := dep.UnmarshalModTOML(v)
		if err != nil {
			return err
		}
		deps.Deps[k] = dep
	}

	return nil
}

func (dep *Dependency) UnmarshalModTOML(data interface{}) error {
	source := Source{}
	err := source.UnmarshalModTOML(data)
	if err != nil {
		return err
	}

	dep.Source = source
	var version string
	if source.Git != nil {
		version, err = source.Git.GetValidGitReference()
		if err != nil {
			return err
		}
	}
	if source.Oci != nil {
		version = source.Oci.Tag
	}

	dep.FullName = fmt.Sprintf(PKG_NAME_PATTERN, dep.Name, version)
	dep.Version = version
	return nil
}

func (source *Source) UnmarshalModTOML(data interface{}) error {
	meta, ok := data.(map[string]interface{})
	if ok {
		if _, ok := meta[LOCAL_PATH_FLAG].(string); ok {
			localPath := Local{}
			err := localPath.UnmarshalModTOML(data)
			if err != nil {
				return err
			}
			source.Local = &localPath
		} else {
			git := Git{}
			err := git.UnmarshalModTOML(data)
			if err != nil {
				return err
			}
			source.Git = &git
		}
	}

	_, ok = data.(string)
	if ok {
		oci := Oci{}
		err := oci.UnmarshalModTOML(data)
		if err != nil {
			return err
		}
		source.Oci = &oci
	}

	return nil
}

const GTI_URL_FLAG = "git"
const GTI_TAG_FLAG = "tag"
const GTI_COMMIT_FLAG = "commit"

func (git *Git) UnmarshalModTOML(data interface{}) error {
	meta, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map[string]interface{}, got %T", data)
	}

	if v, ok := meta[GTI_URL_FLAG].(string); ok {
		git.Url = v
	}

	if v, ok := meta[GTI_TAG_FLAG].(string); ok {
		git.Tag = v
	}

	if v, ok := meta[GTI_COMMIT_FLAG].(string); ok {
		git.Commit = v
	}

	return nil
}

func (oci *Oci) UnmarshalModTOML(data interface{}) error {
	meta, ok := data.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", data)
	}

	oci.Tag = meta

	return nil
}

const LOCAL_PATH_FLAG = "path"

func (local *Local) UnmarshalModTOML(data interface{}) error {
	meta, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map[string]interface{}, got %T", data)
	}

	if v, ok := meta[LOCAL_PATH_FLAG].(string); ok {
		local.Path = v
	}

	return nil
}

func (dep *Dependencies) MarshalLockTOML() (string, error) {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(dep); err != nil {
		return "", reporter.NewErrorEvent(reporter.FailedLoadKclModLock, err, "failed to lock dependencies version")
	}
	return buf.String(), nil
}

func (dep *Dependencies) UnmarshalLockTOML(data string) error {
	if _, err := toml.NewDecoder(strings.NewReader(data)).Decode(dep); err != nil {
		return reporter.NewErrorEvent(reporter.FailedLoadKclModLock, err, "failed to load kcl.mod.lock")
	}

	return nil
}
