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
	dependencies := mod.Dependencies.MarshalTOML()
	if dependencies != "" {
		sb.WriteString(NEWLINE)
		sb.WriteString(dependencies)
	}
	devDependencies := mod.DevDependencies.MarshalTOMLWithSection(DEV_DEPS_PATTERN)
	if devDependencies != "" {
		sb.WriteString(NEWLINE)
		sb.WriteString(devDependencies)
	}
	profiles := mod.Profiles.MarshalTOML()
	if profiles != "" {
		sb.WriteString(NEWLINE)
		sb.WriteString(profiles)
	}
	return sb.String()
}

const PACKAGE_PATTERN = "[package]"

func (pkg *Package) MarshalTOML() string {
	var sb strings.Builder
	sb.WriteString(PACKAGE_PATTERN)
	sb.WriteString(NEWLINE)
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(pkg); err != nil {
		return ""
	}
	sb.WriteString(buf.String())
	return sb.String()
}

const (
	DEPS_PATTERN     = "[dependencies]"
	DEV_DEPS_PATTERN = "[dev_dependencies]"
)

func (dep *Dependencies) MarshalTOML() string {
	return dep.MarshalTOMLWithSection(DEPS_PATTERN)
}

// MarshalTOMLWithSection renders the dependency set under the given TOML
// table header — use DEPS_PATTERN for `[dependencies]` and
// DEV_DEPS_PATTERN for `[dev_dependencies]`. An empty dep map produces
// an empty string so callers can leave optional sections out.
func (dep *Dependencies) MarshalTOMLWithSection(section string) string {
	var sb strings.Builder
	if dep.Deps != nil && dep.Deps.Len() != 0 {
		sb.WriteString(section)
		for _, depKeys := range dep.Deps.Keys() {
			dep, ok := dep.Deps.Get(depKeys)
			if !ok {
				break
			}
			sb.WriteString(NEWLINE)
			sb.WriteString(dep.MarshalTOML())
		}
		sb.WriteString(NEWLINE)
	}
	return sb.String()
}

const DEP_PATTERN = "%s = %s"

func (dep *Dependency) MarshalTOML() string {
	var sb strings.Builder

	depName := dep.Name
	if !dep.Source.ModSpec.IsNil() {
		if dep.Source.ModSpec.Alias != "" {
			depName = dep.Source.ModSpec.Alias
		}
	}

	sb.WriteString(fmt.Sprintf(DEP_PATTERN, depName, dep.Source.MarshalTOML()))
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
			return ""
		}
		sb.WriteString(buf.String())
	}
	return sb.String()
}

const (
	PACKAGE_FLAG  = "package"
	DEPS_FLAG     = "dependencies"
	DEV_DEPS_FLAG = "dev_dependencies"
	PROFILES_FLAG = "profile"
)

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

	deps := Dependencies{
		Deps: orderedmap.NewOrderedMap[string, Dependency](),
	}
	if v, ok := meta[DEPS_FLAG]; ok {
		err := deps.UnmarshalModTOML(v)
		if err != nil {
			return err
		}
	}
	mod.Dependencies = deps

	devDeps := Dependencies{
		Deps: orderedmap.NewOrderedMap[string, Dependency](),
	}
	if v, ok := meta[DEV_DEPS_FLAG]; ok {
		err := devDeps.UnmarshalModTOML(v)
		if err != nil {
			return err
		}
	}
	mod.DevDependencies = devDeps

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

const (
	NAME_FLAG        = "name"
	EDITION_FLAG     = "edition"
	VERSION_FLAG     = "version"
	DESCRIPTION_FLAG = "description"
	INCLUDE_FLAG     = "include"
	EXCLUDE_FLAG     = "exclude"
)

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
		if !dep.Source.ModSpec.IsNil() {
			if dep.Source.ModSpec.Name != dep.Name {
				dep.Source.ModSpec.Alias = dep.Name
			}
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

	if source.ModSpec != nil {
		version = source.ModSpec.Version
	}

	dep.FullName = fmt.Sprintf(PKG_NAME_PATTERN, dep.Name, version)
	dep.Version = version
	if dep.Source.ModSpec != nil && dep.Source.ModSpec.Name == "" {
		dep.Source.ModSpec.Name = dep.Name
	}
	return nil
}

// DependenciesUI mirrors the on-disk shape of kcl.mod.lock. Both regular and
// dev dependencies live in their own top-level tables so that the lock file
// stays self-describing and symmetrical with the two sections in kcl.mod.
// JSON keys are kept stable for backward compatibility (public CLI consumers);
// `omitempty` keeps the JSON shape unchanged for callers that never declared
// a dev dep.
type DependenciesUI struct {
	Deps    map[string]Dependency `json:"packages" toml:"dependencies,omitempty"`
	DevDeps map[string]Dependency `json:"dev_packages,omitempty" toml:"dev_dependencies,omitempty"`
}

// marshalDepsInternal is a shared helper used by both MarshalLockTOML paths
// to apply the host-less Reg stripping policy before encoding.
func marshalDepsInternal(deps *orderedmap.OrderedMap[string, Dependency]) map[string]Dependency {
	out := make(map[string]Dependency)
	if deps == nil {
		return out
	}
	for _, depKey := range deps.Keys() {
		dep, ok := deps.Get(depKey)
		if !ok {
			break
		}
		// For host-less OCI dependencies (registry resolved from KPM_REG at runtime),
		// clear Reg before writing so that reg,omitempty omits it from the lock.
		// This produces a single lock file valid across all registry accounts.
		if dep.Source.Oci != nil && dep.Source.Oci.RegFromEnv {
			depCopy := dep
			ociCopy := *dep.Source.Oci
			ociCopy.Reg = ""
			depCopy.Source.Oci = &ociCopy
			out[depKey] = depCopy
		} else {
			out[depKey] = dep
		}
	}
	return out
}

// MarshalLockDepsTOML renders the regular `[dependencies]` table of a
// kcl.mod.lock file. MarshalLockDevDepsTOML does the same for the dev
// section.
func (dep *Dependencies) MarshalLockDepsTOML() (string, error) {
	ui := DependenciesUI{
		Deps: marshalDepsInternal(dep.Deps),
	}
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(&ui); err != nil {
		return "", reporter.NewErrorEvent(reporter.FailedLoadKclModLock, err, "failed to lock dependencies version")
	}
	return buf.String(), nil
}

// MarshalLockDevDepsTOML writes only the [dev_dependencies] table.
func (dep *Dependencies) MarshalLockDevDepsTOML() (string, error) {
	ui := DependenciesUI{
		DevDeps: marshalDepsInternal(dep.Deps),
	}
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(&ui); err != nil {
		return "", reporter.NewErrorEvent(reporter.FailedLoadKclModLock, err, "failed to lock dev dependencies version")
	}
	return buf.String(), nil
}

// MarshalLockTOML is kept as a thin wrapper for backward compatibility — only
// renders the regular `[dependencies]` section.
func (dep *Dependencies) MarshalLockTOML() (string, error) {
	return dep.MarshalLockDepsTOML()
}

// unmarshalDepsInternal applies the RegFromEnv sentinel restoration and
// inserts keys (sorted) into the supplied ordered map.
func unmarshalDepsInternal(ui *DependenciesUI, depMap *orderedmap.OrderedMap[string, Dependency]) {
	keys := make([]string, 0, len(ui.Deps))
	for k := range ui.Deps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		d := ui.Deps[k]
		// Restore the RegFromEnv sentinel for host-less entries (no reg in the lock)
		// so that subsequent marshal operations continue to emit them as host-less.
		if d.Source.Oci != nil && d.Source.Oci.Reg == "" && d.Source.Oci.Repo != "" {
			d.Source.Oci.RegFromEnv = true
		}
		depMap.Set(k, d)
	}
}

// UnmarshalLockDepsTOML parses the regular `[dependencies]` table from a
// kcl.mod.lock buffer and stores entries into dep.Deps.
func (dep *Dependencies) UnmarshalLockDepsTOML(data string) error {
	if dep.Deps == nil {
		dep.Deps = orderedmap.NewOrderedMap[string, Dependency]()
	}
	ui := DependenciesUI{Deps: make(map[string]Dependency)}
	if _, err := toml.NewDecoder(strings.NewReader(data)).Decode(&ui); err != nil {
		return reporter.NewErrorEvent(reporter.FailedLoadKclModLock, err, "failed to load kcl.mod.lock")
	}
	unmarshalDepsInternal(&ui, dep.Deps)
	return nil
}

// UnmarshalLockTOML parses both `[dependencies]` and `[dev_dependencies]`
// tables from a kcl.mod.lock buffer. The `deps` argument receives regular
// entries; pass a freshly-initialized Dependencies struct for `devDeps` if
// the caller wants dev entries populated, or nil to skip them.
func UnmarshalLockTOML(data string, deps, devDeps *Dependencies) error {
	if deps != nil && deps.Deps == nil {
		deps.Deps = orderedmap.NewOrderedMap[string, Dependency]()
	}
	if devDeps != nil && devDeps.Deps == nil {
		devDeps.Deps = orderedmap.NewOrderedMap[string, Dependency]()
	}
	ui := DependenciesUI{
		Deps:    make(map[string]Dependency),
		DevDeps: make(map[string]Dependency),
	}
	if _, err := toml.NewDecoder(strings.NewReader(data)).Decode(&ui); err != nil {
		return reporter.NewErrorEvent(reporter.FailedLoadKclModLock, err, "failed to load kcl.mod.lock")
	}
	if deps != nil {
		unmarshalDepsInternal(&ui, deps.Deps)
	}
	if devDeps != nil {
		unmarshalDepsInternal(&DependenciesUI{Deps: ui.DevDeps}, devDeps.Deps)
	}
	return nil
}

// UnmarshalLockTOML is kept as the single-section method for backward
// compatibility — only populates dep.Deps from the `[dependencies]` table.
func (dep *Dependencies) UnmarshalLockTOML(data string) error {
	return dep.UnmarshalLockDepsTOML(data)
}
