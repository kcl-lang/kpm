// Copyright 2026 The KCL Authors. All rights reserved.

package git

import (
	"testing"
)

func TestSplitSubdir(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantBase  string
		wantSub   string
	}{
		{
			name:     "no subdir",
			in:       "https://github.com/org/repo.git",
			wantBase: "https://github.com/org/repo.git",
			wantSub:  "",
		},
		{
			name:     "with subdir",
			in:       "https://github.com/org/repo.git//pkg/sub",
			wantBase: "https://github.com/org/repo.git",
			wantSub:  "pkg/sub",
		},
		{
			name:     "subdir with leading slash",
			in:       "https://github.com/org/repo.git///pkg/sub",
			wantBase: "https://github.com/org/repo.git",
			wantSub:  "pkg/sub",
		},
		{
			name:     "scheme prefix preserved",
			in:       "git::https://github.com/org/repo.git//pkg",
			wantBase: "git::https://github.com/org/repo.git",
			wantSub:  "pkg",
		},
		{
			name:     "scheme with ref and subdir",
			in:       "git::https://github.com/org/repo.git?ref=v1.0//pkg/sub",
			wantBase: "git::https://github.com/org/repo.git?ref=v1.0",
			wantSub:  "pkg/sub",
		},
		{
			name:     "query string on subdir",
			in:       "https://github.com/org/repo.git//pkg?ref=v1",
			wantBase: "https://github.com/org/repo.git?ref=v1",
			wantSub:  "pkg",
		},
		{
			name:     "fragment on subdir",
			in:       "https://github.com/org/repo.git//pkg#frag",
			wantBase: "https://github.com/org/repo.git#frag",
			wantSub:  "pkg",
		},
		{
			name:     "trailing slash subdir is empty",
			in:       "https://github.com/org/repo.git//",
			wantBase: "https://github.com/org/repo.git//",
			wantSub:  "",
		},
		{
			name:     "scheme delimiter https:// not split",
			in:       "https://github.com/org/repo",
			wantBase: "https://github.com/org/repo",
			wantSub:  "",
		},
		{
			name:     "ssh-like scheme not split",
			in:       "ssh://git@github.com/org/repo.git",
			wantBase: "ssh://git@github.com/org/repo.git",
			wantSub:  "",
		},
		{
			name:     "single subdir segment",
			in:       "https://example.com/repo//main",
			wantBase: "https://example.com/repo",
			wantSub:  "main",
		},
		{
			name:     "deep nested subdir",
			in:       "https://example.com/repo//a/b/c/d",
			wantBase: "https://example.com/repo",
			wantSub:  "a/b/c/d",
		},
		{
			// Regression: file:// URLs point at local bare repos and
			// contain a "///" right after the scheme delimiter. The
			// inner "//" must not be mistaken for a sub-dir marker,
			// otherwise the local path gets routed to go-getter as a
			// sub-directory of "file:" and the clone fails with
			// "ssh: Could not resolve hostname file".
			name:     "file scheme with absolute path is not split",
			in:       "file:///tmp/local_bare_repo",
			wantBase: "file:///tmp/local_bare_repo",
			wantSub:  "",
		},
		{
			name:     "file scheme with query is not split",
			in:       "file:///tmp/local_bare_repo?ref=4e59d5852cd76542f9f0ec65e5773ca9f4e02462",
			wantBase: "file:///tmp/local_bare_repo?ref=4e59d5852cd76542f9f0ec65e5773ca9f4e02462",
			wantSub:  "",
		},
		{
			name:     "git:: prefix on file scheme is not split",
			in:       "git::file:///tmp/local_bare_repo?ref=4e59d5852cd76542f9f0ec65e5773ca9f4e02462",
			wantBase: "git::file:///tmp/local_bare_repo?ref=4e59d5852cd76542f9f0ec65e5773ca9f4e02462",
			wantSub:  "",
		},
		{
			// Even with a file:// scheme, an explicit //subdir still
			// works because the parser looks for the SECOND "//"
			// after the scheme delimiter, not just any "//".
			name:     "file scheme with subdir",
			in:       "file:///tmp/repo//modules",
			wantBase: "file:///tmp/repo",
			wantSub:  "modules",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base, sub, err := SplitSubdir(c.in)
			if err != nil {
				t.Fatalf("SplitSubdir returned error: %v", err)
			}
			if base != c.wantBase {
				t.Errorf("base = %q, want %q", base, c.wantBase)
			}
			if sub != c.wantSub {
				t.Errorf("sub = %q, want %q", sub, c.wantSub)
			}
		})
	}
}

func TestSplitSubdir_Empty(t *testing.T) {
	base, sub, err := SplitSubdir("")
	if err != nil {
		t.Fatalf("SplitSubdir(\"\") returned error: %v", err)
	}
	if base != "" || sub != "" {
		t.Errorf("SplitSubdir(\"\") = (%q, %q), want (\"\", \"\")", base, sub)
	}
}

func TestIsLikelyScheme(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"git", true},
		{"http", true},
		{"git-lfs", true},
		{"git_lfs", true},
		{"git123", true},
		{"git::", false}, // colon not allowed
		{"", false},
		{"git repo", false},
		{"git.proto", false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := isLikelyScheme(c.in); got != c.want {
				t.Errorf("isLikelyScheme(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}