package client

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/module"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/utils"
)

func testGraph(t *testing.T) {
	testPath := getTestDir("test_graph")
	modPath := filepath.Join(testPath, "pkg")

	kpmcli, err := NewKpmClient()
	if err != nil {
		t.Fatalf("failed to create kpm client: %v", err)
	}

	kMod, err := pkg.LoadKclPkgWithOpts(
		pkg.WithPath(modPath),
		pkg.WithSettings(kpmcli.GetSettings()),
	)

	if err != nil {
		t.Fatalf("failed to load kcl package: %v", err)
	}

	dGraph, err := kpmcli.Graph(
		WithGraphMod(kMod),
	)
	if err != nil {
		t.Fatalf("failed to create dependency graph: %v", err)
	}

	graStr, err := dGraph.DisplayGraphFromVertex(
		module.Version{Path: kMod.GetPkgName(), Version: kMod.GetPkgVersion()},
	)

	if err != nil {
		t.Fatalf("failed to display graph: %v", err)
	}

	assert.Contains(t, utils.RmNewline(graStr), "pkg@0.0.1 dep@0.0.1")
	assert.Contains(t, utils.RmNewline(graStr), "pkg@0.0.1 helloworld@0.1.4")
	assert.Contains(t, utils.RmNewline(graStr), "dep@0.0.1 helloworld@0.1.4")
}
