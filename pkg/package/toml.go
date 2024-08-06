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
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	orderedmap "github.com/elliotchance/orderedmap/v2"

	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/reporter"
)

const NEWLINE = "\n"

func (mod *ModFile) MarshalTOML() string {
	var sb strings.Builder
	sb.WriteString(mod.Pkg.MarshalTOML())
	sb.WriteString(NEWLINE)
	sb.WriteString(mod.Dependencies.MarshalTOML())
	sb.WriteString(NEWLINE)
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
	return sb.String()
}

const DEPS_PATTERN = "[dependencies]"

func (dep *Dependencies) MarshalTOML() string {
	var sb strings.Builder
	if dep.Deps != nil && dep.Deps.Len() != 0 {
		sb.WriteString(DEPS_PATTERN)
		for _, depKeys := range dep.Deps.Keys() {
			dep, ok := dep.Deps.Get(depKeys)
			if !ok {
				break
			}
			sb.WriteString(NEWLINE)
			sb.WriteString(dep.MarshalTOML())
		}
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
			Deps: orderedmap.NewOrderedMap[string, Dependency](),
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
const INCLUDE_FLAG = "include"
const EXCLUDE_FLAG = "exclude"

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

	convertToStringArray := func(v interface{}) []string {
		var arr []string
		for _, item := range v.([]interface{}) {
			arr = append(arr, item.(string))
		}
		return arr
	}

	if v, ok := meta[INCLUDE_FLAG].([]interface{}); ok {
		pkg.Include = convertToStringArray(v)
	}

	if v, ok := meta[EXCLUDE_FLAG].([]interface{}); ok {
		pkg.Exclude = convertToStringArray(v)
	}

	return nil
}

func (deps *Dependencies) UnmarshalModTOML(data interface{}) error {
	meta, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map[string]interface{}, got %T", data)
	}

	var keys []string
	for k := range meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := meta[k]
		dep := Dependency{}
		dep.Name = k

		err := dep.UnmarshalModTOML(v)
		if err != nil {
			return err
		}
		deps.Deps.Set(k, dep)
	}

	return nil
}

func (dep *Dependency) UnmarshalModTOML(data interface{}) error {
	source := downloader.Source{}
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
	if source.Registry != nil {
		version = source.Registry.Version
	}

	dep.FullName = fmt.Sprintf(PKG_NAME_PATTERN, dep.Name, version)
	dep.Version = version
	return nil
}

type DependenciesUI struct {
	Deps map[string]Dependency `json:"packages" toml:"dependencies,omitempty"`
}

func (dep *Dependencies) MarshalLockTOML() (string, error) {

	marshaledDeps := make(map[string]Dependency)
	for _, depKey := range dep.Deps.Keys() {
		dep, ok := dep.Deps.Get(depKey)
		if !ok {
			break
		}
		marshaledDeps[depKey] = dep
	}

	lockDepdenciesUI := DependenciesUI{
		Deps: marshaledDeps,
	}

	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(&lockDepdenciesUI); err != nil {
		return "", reporter.NewErrorEvent(reporter.FailedLoadKclModLock, err, "failed to lock dependencies version")
	}
	return buf.String(), nil
}

func (dep *Dependencies) UnmarshalLockTOML(data string) error {

	if dep.Deps == nil {
		dep.Deps = orderedmap.NewOrderedMap[string, Dependency]()
	}

	lockDepdenciesUI := DependenciesUI{
		Deps: make(map[string]Dependency),
	}

	if _, err := toml.NewDecoder(strings.NewReader(data)).Decode(&lockDepdenciesUI); err != nil {
		return reporter.NewErrorEvent(reporter.FailedLoadKclModLock, err, "failed to load kcl.mod.lock")
	}

	var keys []string
	for k := range lockDepdenciesUI.Deps {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		dep.Deps.Set(k, lockDepdenciesUI.Deps[k])
	}

	return nil
}
