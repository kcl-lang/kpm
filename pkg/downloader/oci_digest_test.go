// Copyright 2026 The KCL Authors. All rights reserved.
//
// Tests for OCI digest pinning support
// (https://github.com/kcl-lang/kpm/issues/480).

package downloader

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestOciFromString_Digest(t *testing.T) {
	const sampleDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	cases := []struct {
		name       string
		in         string
		wantDigest string
		wantTag    string
		wantErr    bool
	}{
		{
			name:       "digest in query string",
			in:         "oci://ghcr.io/kcl-lang/helloworld?digest=" + sampleDigest,
			wantDigest: sampleDigest,
		},
		{
			name:    "tag in query string (unchanged behaviour)",
			in:      "oci://ghcr.io/kcl-lang/helloworld?tag=0.0.1",
			wantTag: "0.0.1",
		},
		{
			name:    "neither digest nor tag",
			in:      "oci://ghcr.io/kcl-lang/helloworld",
		},
		{
			name:    "both tag and digest set is rejected",
			in:      "oci://ghcr.io/kcl-lang/helloworld?tag=0.0.1&digest=" + sampleDigest,
			wantErr: true,
		},
		{
			name:    "digest with unsupported algorithm is rejected",
			in:      "oci://ghcr.io/kcl-lang/helloworld?digest=md5:abc",
			wantErr: true,
		},
		{
			name:       "empty digest in query string is treated as no digest",
			in:         "oci://ghcr.io/kcl-lang/helloworld?digest=",
			wantDigest: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var oci Oci
			err := oci.FromString(c.in)
			if c.wantErr {
				assert.Assert(t, err != nil, "expected error for %q", c.in)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, oci.Digest, c.wantDigest)
			assert.Equal(t, oci.Tag, c.wantTag)
		})
	}
}

func TestOciToString_Digest(t *testing.T) {
	const sampleDigest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	// url.Values.Encode() URL-encodes ":" — the parser
	// accepts both forms (it uses url.Query().Get which decodes
	// them back), but ToString always emits the encoded form.
	const encodedDigest = "sha256%3A1111111111111111111111111111111111111111111111111111111111111111"

	cases := []struct {
		name string
		oci  Oci
		want string
	}{
		{
			name: "tag-only URL round-trip",
			oci:  Oci{Reg: "ghcr.io", Repo: "kcl-lang/helloworld", Tag: "0.0.1"},
			want: "oci://ghcr.io/kcl-lang/helloworld?tag=0.0.1",
		},
		{
			name: "digest-only URL round-trip",
			oci:  Oci{Reg: "ghcr.io", Repo: "kcl-lang/helloworld", Digest: sampleDigest},
			want: "oci://ghcr.io/kcl-lang/helloworld?digest=" + encodedDigest,
		},
		{
			name: "both tag and digest → emitted as-is (parser rejects this combo)",
			oci:  Oci{Reg: "ghcr.io", Repo: "kcl-lang/helloworld", Tag: "0.0.1", Digest: sampleDigest},
			want: "oci://ghcr.io/kcl-lang/helloworld?digest=" + encodedDigest + "&tag=0.0.1",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.oci.ToString()
			assert.NilError(t, err)
			assert.Equal(t, got, c.want)
		})
	}
}

func TestOciNoRef(t *testing.T) {
	cases := []struct {
		name string
		oci  Oci
		want bool
	}{
		{"both empty", Oci{Reg: "ghcr.io", Repo: "kcl-lang/helloworld"}, true},
		{"tag set", Oci{Reg: "ghcr.io", Repo: "kcl-lang/helloworld", Tag: "0.0.1"}, false},
		{"digest set", Oci{Reg: "ghcr.io", Repo: "kcl-lang/helloworld", Digest: "sha256:abc"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.oci.NoRef(), c.want)
		})
	}
}

func TestOciResolveReference(t *testing.T) {
	const digest = "sha256:2222222222222222222222222222222222222222222222222222222222222222"

	cases := []struct {
		name string
		oci  Oci
		want string
	}{
		{"empty → latest", Oci{}, "latest"},
		{"tag only", Oci{Tag: "0.0.1"}, "0.0.1"},
		{"digest only", Oci{Digest: digest}, digest},
		{"tag wins over digest", Oci{Tag: "0.0.1", Digest: digest}, "0.0.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.oci.ResolveReference(), c.want)
		})
	}
}

func TestOciValidateDigest(t *testing.T) {
	const goodDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	cases := []struct {
		name    string
		digest  string
		wantErr bool
	}{
		{"empty is fine", "", false},
		{"sha256:hex is accepted", goodDigest, false},
		{"sha512:hex is rejected (only sha256 supported)", "sha512:0000000000000000000000000000000000000000000000000000000000000000", true},
		{"bare hex without algo is rejected", "0000000000000000000000000000000000000000000000000000000000000000", true},
		{"malformed reference is rejected", "sha256:not-hex", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := (&Oci{Digest: c.digest}).ValidateDigest()
			if c.wantErr {
				assert.Assert(t, err != nil, "expected error for digest %q", c.digest)
				return
			}
			assert.NilError(t, err)
		})
	}
}

func TestSourceLocalPath_Digest(t *testing.T) {
	const digest = "sha256:3333333333333333333333333333333333333333333333333333333333333333"

	tagSrc := &Source{Oci: &Oci{Reg: "ghcr.io", Repo: "org/helloworld", Tag: "0.0.1"}}
	digestSrc := &Source{Oci: &Oci{Reg: "ghcr.io", Repo: "org/helloworld", Digest: digest}}

	tagPath := tagSrc.LocalPath("/cache")
	digestPath := digestSrc.LocalPath("/cache")

	// Both should be under the same root.
	assert.Assert(t, len(tagPath) > len("/cache"))
	assert.Assert(t, len(digestPath) > len("/cache"))

	// They MUST differ — a digest pin and a tag pin are different
	// identities even if they happen to point at the same content.
	assert.Assert(t, tagPath != digestPath,
		"tag-based local path %q collides with digest-based path %q",
		tagPath, digestPath,
	)
}
