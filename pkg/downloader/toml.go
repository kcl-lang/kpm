// Copyright 2022 The KCL Authors. All rights reserved.
package downloader

import (
	"fmt"
	"strings"

	"kcl-lang.io/kpm/pkg/constants"
)

const NEWLINE = "\n"
const SOURCE_PATTERN = "{ %s }"
const DEP_PATTERN = "%s = %s"
const SPEC_PATTERN = "%s = %q"

func (ps *ModSpec) MarshalTOML() string {
	var sb strings.Builder
	if ps != nil && len(ps.Version) != 0 && len(ps.Name) != 0 {
		sb.WriteString(fmt.Sprintf(SPEC_PATTERN, ps.Name, ps.Version))
		return sb.String()
	}

	return sb.String()
}

func (source *Source) MarshalTOML() string {
	var sb strings.Builder
	if source.SpecOnly {
		return source.ModSpec.MarshalTOML()
	} else {
		var pkgVersion string
		var tomlStr string
		if source.ModSpec != nil && len(source.ModSpec.Version) > 0 {
			pkgVersion = fmt.Sprintf(", version = %q", source.ModSpec.Version)
		}

		if source.Git != nil {
			tomlStr = source.Git.MarshalTOML()
			if len(tomlStr) != 0 {
				tomlStr = fmt.Sprintf(SOURCE_PATTERN, tomlStr+pkgVersion)
			}
		}

		if source.Oci != nil {
			tomlStr = source.Oci.MarshalTOML()
			if len(tomlStr) != 0 {
				if len(source.Oci.Reg) != 0 && len(source.Oci.Repo) != 0 {
					tomlStr = fmt.Sprintf(SOURCE_PATTERN, tomlStr+pkgVersion)
				}
			}
		}

		if source.Local != nil {
			tomlStr = source.Local.MarshalTOML()
			if len(tomlStr) != 0 {
				tomlStr = fmt.Sprintf(SOURCE_PATTERN, tomlStr+pkgVersion)
			}
		}

		if source.ModSpec != nil && len(source.ModSpec.Name) != 0 {
			sb.WriteString(fmt.Sprintf(DEP_PATTERN, source.ModSpec.Name, tomlStr))
		} else {
			sb.WriteString(tomlStr)
		}
	}

	return sb.String()
}

const GIT_URL_PATTERN = "git = \"%s\""
const TAG_PATTERN = "tag = \"%s\""
const GIT_COMMIT_PATTERN = "commit = \"%s\""
const GIT_BRANCH_PATTERN = "branch = \"%s\""
const VERSION_PATTERN = "version = \"%s\""
const GIT_PACKAGE = "package = \"%s\""
const SEPARATOR = ", "

func (git *Git) MarshalTOML() string {
	var sb strings.Builder
	if len(git.Url) != 0 {
		sb.WriteString(fmt.Sprintf(GIT_URL_PATTERN, git.Url))
	}
	if len(git.Tag) != 0 {
		sb.WriteString(SEPARATOR)
		sb.WriteString(fmt.Sprintf(TAG_PATTERN, git.Tag))
	}
	if len(git.Commit) != 0 {
		sb.WriteString(SEPARATOR)
		sb.WriteString(fmt.Sprintf(GIT_COMMIT_PATTERN, git.Commit))
	}

	if len(git.Branch) != 0 {
		sb.WriteString(SEPARATOR)
		sb.WriteString(fmt.Sprintf(GIT_BRANCH_PATTERN, git.Branch))
	}

	if len(git.Version) != 0 {
		sb.WriteString(SEPARATOR)
		sb.WriteString(fmt.Sprintf(VERSION_PATTERN, git.Version))
	}

	if len(git.Package) != 0 {
		sb.WriteString(SEPARATOR)
		sb.WriteString(fmt.Sprintf(GIT_PACKAGE, git.Package))
	}

	return sb.String()
}

const OCI_URL_PATTERN = "oci = \"%s\""

func (oci *Oci) MarshalTOML() string {
	var sb strings.Builder
	if len(oci.Reg) != 0 && len(oci.Repo) != 0 {
		sb.WriteString(fmt.Sprintf(OCI_URL_PATTERN, oci.IntoOciUrl()))
		if len(oci.Tag) != 0 {
			sb.WriteString(SEPARATOR)
			sb.WriteString(fmt.Sprintf(TAG_PATTERN, oci.Tag))
		}
	} else if len(oci.Reg) == 0 && len(oci.Repo) == 0 && len(oci.Tag) != 0 {
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
		} else if _, ok := meta["git"]; ok {
			git := Git{}
			err := git.UnmarshalModTOML(data)
			if err != nil {
				return err
			}
			source.Git = &git
		} else if _, ok := meta["oci"]; ok {
			oci := Oci{}
			err := oci.UnmarshalModTOML(data)
			if err != nil {
				return err
			}
			source.Oci = &oci
		}

		if v, ok := meta["version"].(string); ok {
			pSpec := ModSpec{}
			err := pSpec.UnmarshalModTOML(v)
			if err != nil {
				return err
			}
			source.ModSpec = &pSpec
		}
	}

	_, ok = data.(string)
	if ok {
		pSpec := ModSpec{}
		err := pSpec.UnmarshalModTOML(data)
		if err != nil {
			return err
		}
		source.ModSpec = &pSpec
	}

	if source.Git == nil &&
		source.Oci == nil &&
		source.Local == nil &&
		source.ModSpec != nil {
		source.SpecOnly = true
	}

	return nil
}

func (ps *ModSpec) UnmarshalModTOML(data interface{}) error {
	version, ok := data.(string)
	if ok {
		ps.Version = version
	}

	return nil
}

const GIT_URL_FLAG = "git"
const TAG_FLAG = "tag"
const GIT_COMMIT_FLAG = "commit"
const GIT_BRANCH_FLAG = "branch"
const GIT_PACKAGE_FLAG = "package"

func (git *Git) UnmarshalModTOML(data interface{}) error {
	meta, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map[string]interface{}, got %T", data)
	}

	if v, ok := meta[GIT_URL_FLAG].(string); ok {
		git.Url = v
	}

	if v, ok := meta[TAG_FLAG].(string); ok {
		git.Tag = v
	}

	if v, ok := meta[GIT_COMMIT_FLAG].(string); ok {
		git.Commit = v
	}

	if v, ok := meta[GIT_BRANCH_FLAG].(string); ok {
		git.Branch = v
	}

	if v, ok := meta[GIT_PACKAGE_FLAG].(string); ok {
		git.Package = v
	}

	return nil
}

func (oci *Oci) UnmarshalModTOML(data interface{}) error {
	if meta, ok := data.(map[string]interface{}); ok {
		if v, ok := meta[constants.OciScheme].(string); ok {
			err := oci.FromString(v)
			if err != nil {
				return err
			}
		}

		if v, ok := meta[TAG_FLAG].(string); ok {
			oci.Tag = v
		}
	}

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
