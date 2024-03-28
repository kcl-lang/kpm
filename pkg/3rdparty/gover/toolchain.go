// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gover

import (
	"context"
	"errors"
	"strings"
)

// FromToolchain returns the Go version for the named toolchain,
// derived from the name itself (not by running the toolchain).
// A toolchain is named "goVERSION".
// A suffix after the VERSION introduced by a -, space, or tab is removed.
// Examples:
//
//	FromToolchain("go1.2.3") == "1.2.3"
//	FromToolchain("go1.2.3-bigcorp") == "1.2.3"
//	FromToolchain("invalid") == ""
func FromToolchain(name string) string {
	if strings.ContainsAny(name, "\\/") {
		// The suffix must not include a path separator, since that would cause
		// exec.LookPath to resolve it from a relative directory instead of from
		// $PATH.
		return ""
	}

	var v string
	if strings.HasPrefix(name, "go") {
		v = name[2:]
	} else {
		return ""
	}
	// Some builds use custom suffixes; strip them.
	if i := strings.IndexAny(v, " \t-"); i >= 0 {
		v = v[:i]
	}
	if !IsValid(v) {
		return ""
	}
	return v
}

func maybeToolchainVersion(name string) string {
	if IsValid(name) {
		return name
	}
	return FromToolchain(name)
}

// ToolchainMax returns the maximum of x and y interpreted as toolchain names,
// compared using Compare(FromToolchain(x), FromToolchain(y)).
// If x and y compare equal, Max returns x.
func ToolchainMax(x, y string) string {
	if Compare(FromToolchain(x), FromToolchain(y)) < 0 {
		return y
	}
	return x
}

// Startup records the information that went into the startup-time version switch.
// It is initialized by switchGoToolchain.
var Startup struct {
	GOTOOLCHAIN   string // $GOTOOLCHAIN setting
	AutoFile      string // go.mod or go.work file consulted
	AutoGoVersion string // go line found in file
	AutoToolchain string // toolchain line found in file
}

// A TooNewError explains that a module is too new for this version of Go.
type TooNewError struct {
	What      string
	GoVersion string
	Toolchain string // for callers if they want to use it, but not printed
}

var ErrTooNew = errors.New("module too new")

func (e *TooNewError) Is(err error) bool {
	return err == ErrTooNew
}

// A Switcher provides the ability to switch to a new toolchain in response to TooNewErrors.
// See [cmd/go/internal/toolchain.Switcher] for documentation.
type Switcher interface {
	Error(err error)
	Switch(ctx context.Context)
}
