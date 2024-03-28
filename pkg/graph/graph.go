package graph

import (
	"fmt"

	"github.com/dominikbraun/graph"
	"golang.org/x/mod/module"
)

// ToAdjacencyList converts graph to adjacency list
func ToAdjacencyList(g graph.Graph[module.Version, module.Version]) (map[module.Version][]module.Version, error) {
	AdjacencyMap, err := g.AdjacencyMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get adjacency map: %w", err)
	}

	adjList := make(map[module.Version][]module.Version)
	for from, v := range AdjacencyMap {
		for to := range v {
			adjList[from] = append(adjList[from], to)
		}
	}
	return adjList, nil
}
