// Copyright 2024 The KCL Authors. All rights reserved.
package downloader

import (
	"testing"

	"gotest.tools/v3/assert"
)

// TestOciHostlessMarshal verifies that an Oci with no registry host (host-less
// dependency resolved from KPM_REG / DefaultOciRegistry at runtime) is serialised
// using the `repo = "..."` key instead of a full `oci = "oci://host/..."` URL,
// both when RegFromEnv was set at parse time and when Reg has been temporarily
// filled for the network call but RegFromEnv is still true.
func TestOciHostlessMarshal(t *testing.T) {
	// Basic host-less: Reg is empty, RegFromEnv set
	oci := &Oci{
		Reg:        "",
		Repo:       "myorg/kcl-templates/utils",
		Tag:        "0.4.0",
		RegFromEnv: true,
	}
	got := oci.MarshalTOML()
	assert.Equal(t, got, `repo = "myorg/kcl-templates/utils", tag = "0.4.0"`)

	// Reg temporarily filled (by resolution), but RegFromEnv is set → still host-less
	ociResolved := &Oci{
		Reg:        "672819064798.dkr.ecr.eu-west-1.amazonaws.com",
		Repo:       "myorg/kcl-templates/utils",
		Tag:        "0.4.0",
		RegFromEnv: true,
	}
	got = ociResolved.MarshalTOML()
	assert.Equal(t, got, `repo = "myorg/kcl-templates/utils", tag = "0.4.0"`)
}

// TestOciFullUrlMarshal verifies that existing full-URL behaviour is unchanged.
func TestOciFullUrlMarshal(t *testing.T) {
	oci := &Oci{
		Reg:  "ghcr.io",
		Repo: "kcl-lang/helloworld",
		Tag:  "0.1.0",
	}
	got := oci.MarshalTOML()
	assert.Equal(t, got, `oci = "oci://ghcr.io/kcl-lang/helloworld", tag = "0.1.0"`)
}

// TestOciShortNameMarshal verifies that the short-name form (no host, no repo) is unchanged.
func TestOciShortNameMarshal(t *testing.T) {
	oci := &Oci{Tag: "0.1.0"}
	got := oci.MarshalTOML()
	assert.Equal(t, got, `"0.1.0"`)
}

// TestOciHostlessUnmarshal verifies that `repo = "..."` in a kcl.mod map is
// parsed into Oci with Reg="", RegFromEnv=true and Repo/Tag set correctly.
func TestOciHostlessUnmarshal(t *testing.T) {
	data := map[string]interface{}{
		"repo": "myorg/kcl-templates/utils",
		"tag":  "0.4.0",
	}
	oci := &Oci{}
	err := oci.UnmarshalModTOML(data)
	assert.NilError(t, err)
	assert.Equal(t, oci.Repo, "myorg/kcl-templates/utils")
	assert.Equal(t, oci.Tag, "0.4.0")
	assert.Equal(t, oci.Reg, "")
	assert.Equal(t, oci.RegFromEnv, true)
}

// TestSourceHostlessUnmarshal verifies that Source.UnmarshalModTOML dispatches
// to OCI when the `repo` key is present and no `oci` key exists.
func TestSourceHostlessUnmarshal(t *testing.T) {
	data := map[string]interface{}{
		"repo":    "myorg/kcl-templates/utils",
		"tag":     "0.4.0",
		"version": "0.4.0",
	}
	src := &Source{}
	err := src.UnmarshalModTOML(data)
	assert.NilError(t, err)
	assert.Assert(t, src.Oci != nil, "expected Oci to be non-nil")
	assert.Equal(t, src.Oci.Repo, "myorg/kcl-templates/utils")
	assert.Equal(t, src.Oci.Tag, "0.4.0")
	assert.Equal(t, src.Oci.Reg, "")
	assert.Equal(t, src.Oci.RegFromEnv, true)
	assert.Assert(t, src.ModSpec != nil, "expected ModSpec to be non-nil")
	assert.Equal(t, src.ModSpec.Version, "0.4.0")
}

// TestOciHostlessMarshalRoundTrip verifies the full marshal→unmarshal round-trip
// for a host-less OCI dependency, including the Source wrapper with version.
func TestOciHostlessMarshalRoundTrip(t *testing.T) {
	src := &Source{
		Oci: &Oci{
			Repo:       "myorg/kcl-templates/utils",
			Tag:        "0.4.0",
			RegFromEnv: true,
		},
		ModSpec: &ModSpec{Version: "0.4.0"},
	}

	marshaled := src.MarshalTOML()
	// Expected: { repo = "myorg/kcl-templates/utils", tag = "0.4.0", version = "0.4.0" }
	assert.Equal(t, marshaled, `{ repo = "myorg/kcl-templates/utils", tag = "0.4.0", version = "0.4.0" }`)

	// Round-trip: unmarshal back
	data := map[string]interface{}{
		"repo":    "myorg/kcl-templates/utils",
		"tag":     "0.4.0",
		"version": "0.4.0",
	}
	src2 := &Source{}
	err := src2.UnmarshalModTOML(data)
	assert.NilError(t, err)
	assert.Assert(t, src2.Oci != nil)
	assert.Equal(t, src2.Oci.Repo, "myorg/kcl-templates/utils")
	assert.Equal(t, src2.Oci.Tag, "0.4.0")
	assert.Equal(t, src2.Oci.Reg, "")
	assert.Equal(t, src2.Oci.RegFromEnv, true)

	// Re-marshal: should produce the same string
	marshaled2 := src2.MarshalTOML()
	assert.Equal(t, marshaled, marshaled2)
}

// TestFromStringHostlessMarksRegFromEnv verifies that parsing an oci:///path URL
// (e.g. from --oci flag) sets RegFromEnv when host is empty.
func TestFromStringHostlessMarksRegFromEnv(t *testing.T) {
	oci := &Oci{}
	err := oci.FromString("oci:///myorg/kcl-templates/utils")
	assert.NilError(t, err)
	assert.Equal(t, oci.Reg, "")
	assert.Equal(t, oci.Repo, "myorg/kcl-templates/utils")
	assert.Equal(t, oci.RegFromEnv, true)
}

// TestFromStringFullUrlDoesNotMarkRegFromEnv verifies that a full oci://host/repo
// URL does NOT set RegFromEnv.
func TestFromStringFullUrlDoesNotMarkRegFromEnv(t *testing.T) {
	oci := &Oci{}
	err := oci.FromString("oci://ghcr.io/kcl-lang/helloworld")
	assert.NilError(t, err)
	assert.Equal(t, oci.Reg, "ghcr.io")
	assert.Equal(t, oci.Repo, "kcl-lang/helloworld")
	assert.Equal(t, oci.RegFromEnv, false)
}
