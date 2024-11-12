package client

import (
	"path/filepath"
	"testing"

	"golang.org/x/mod/module"
	"gotest.tools/v3/assert"
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

	assert.Equal(t, utils.RmNewline(graStr), "pkg@0.0.1 dep@0.0.1pkg@0.0.1 helloworld@0.1.4dep@0.0.1 helloworld@0.1.4")
}
