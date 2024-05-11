package graph

import (
	"fmt"

	"github.com/dominikbraun/graph"
	"golang.org/x/mod/module"
	pkg "kcl-lang.io/kpm/pkg/package"
)

func ChangeGraphType(g graph.Graph[pkg.Dependency, pkg.Dependency]) (graph.Graph[module.Version, module.Version], error) {
	AdjacencyMap, err := g.AdjacencyMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get adjacency map: %w", err)
	}

	m := func(dep pkg.Dependency) module.Version {
		return module.Version{Path: dep.Name, Version: dep.Version}
	}

	moduleHash := func(m module.Version) module.Version {
		return m
	}

	depGraph := graph.New(moduleHash, graph.Directed(), graph.PreventCycles())
	for node, edges := range AdjacencyMap {
		err := depGraph.AddVertex(m(node))
		if err != nil && err != graph.ErrVertexAlreadyExists {
			return nil, fmt.Errorf("failed to add vertex: %w", err)
		}
		for edge := range edges {
			err := depGraph.AddVertex(m(edge))
			if err != nil && err != graph.ErrVertexAlreadyExists {
				return nil, fmt.Errorf("failed to add vertex: %w", err)
			}
			err = depGraph.AddEdge(m(node), m(edge))
			if err != nil {
				return nil, fmt.Errorf("failed to add edge: %w", err)
			}
		}
	}
	return depGraph, nil
}
