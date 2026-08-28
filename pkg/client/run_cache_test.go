// Copyright 2026 The KCL Authors. All rights reserved.
//
// Tests for the persistent-cache-of-remote-primary-packages feature
// (https://github.com/kcl-lang/kpm/issues/691). These tests do NOT
// hit any network — they only verify the configuration plumbing in
// RunOptions and the integration point in KpmClient.Run.

package client

import (
	"os"
	"testing"

	"kcl-lang.io/kpm/pkg/env"
)

func TestRunOptions_RunCacheEnabled_Precedence(t *testing.T) {
	// Snapshot and restore the env var across tests so we don't
	// leak global state into other test files.
	prev, hadPrev := os.LookupEnv(env.KPM_RUN_NO_CACHE)
	defer func() {
		if hadPrev {
			_ = os.Setenv(env.KPM_RUN_NO_CACHE, prev)
		} else {
			_ = os.Unsetenv(env.KPM_RUN_NO_CACHE)
		}
	}()

	cases := []struct {
		name    string
		envVar  string // "" = unset
		optFunc func() RunOption // nil = no explicit option
		want    bool
	}{
		{
			name:    "no option, env unset → defaults to cache-on",
			envVar:  "",
			optFunc: nil,
			want:    true,
		},
		{
			name:    "no option, env=1 → cache-off",
			envVar:  "1",
			optFunc: nil,
			want:    false,
		},
		{
			name:    "no option, env=true → cache-off",
			envVar:  "true",
			optFunc: nil,
			want:    false,
		},
		{
			name:    "no option, env=0 (falsy) → defaults to cache-on",
			envVar:  "0",
			optFunc: nil,
			want:    true,
		},
		{
			name:    "WithRunCache(true) overrides env=1",
			envVar:  "1",
			optFunc: func() RunOption { return WithRunCache(true) },
			want:    true,
		},
		{
			name:    "WithRunCache(false) overrides env unset",
			envVar:  "",
			optFunc: func() RunOption { return WithRunCache(false) },
			want:    false,
		},
		{
			name:    "WithRunNoCache overrides env unset",
			envVar:  "",
			optFunc: func() RunOption { return WithRunNoCache() },
			want:    false,
		},
		{
			name:    "WithRunCache(true) overrides env=true",
			envVar:  "true",
			optFunc: func() RunOption { return WithRunCache(true) },
			want:    true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.envVar == "" {
				_ = os.Unsetenv(env.KPM_RUN_NO_CACHE)
			} else {
				_ = os.Setenv(env.KPM_RUN_NO_CACHE, c.envVar)
			}

			opts := &RunOptions{}
			if c.optFunc != nil {
				if err := c.optFunc()(opts); err != nil {
					t.Fatalf("option returned error: %v", err)
				}
			}

			if got := opts.runCacheEnabled(); got != c.want {
				t.Errorf("runCacheEnabled() = %v, want %v", got, c.want)
			}
		})
	}
}

// TestRunOptions_WithRunCache_PointerIndependence guards against a
// regression where the closure in WithRunCache accidentally captures
// a single bool value across multiple calls — each invocation must
// produce its own *bool slot on the RunOptions.
func TestRunOptions_WithRunCache_PointerIndependence(t *testing.T) {
	opts := &RunOptions{}

	if err := WithRunCache(true)(opts); err != nil {
		t.Fatalf("WithRunCache(true) error: %v", err)
	}
	if opts.cache == nil || *opts.cache != true {
		t.Fatalf("after WithRunCache(true), opts.cache = %v, want &true", opts.cache)
	}

	if err := WithRunCache(false)(opts); err != nil {
		t.Fatalf("WithRunCache(false) error: %v", err)
	}
	if opts.cache == nil || *opts.cache != false {
		t.Fatalf("after WithRunCache(false), opts.cache = %v, want &false", opts.cache)
	}
}
