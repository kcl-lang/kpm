package graph

import (
	"fmt"
	"github.com/dominikbraun/graph"
)

// Union combines two given graphs into a new graph. The vertex hashes in both
// graphs are expected to be unique. The two input graphs will remain unchanged.
//
// Both graphs should be either directed or undirected. All traits for the new
// graph will be derived from g.
//
// If the same vertex/edge happens to be in both g and h, then an error will not be
// thrown as happens in original Union function and successful operation takes place.
func Union[K comparable, T any](g, h graph.Graph[K, T]) (graph.Graph[K, T], error) {
	union, err := g.Clone()
	if err != nil {
		return union, fmt.Errorf("failed to clone g: %w", err)
	}

	adjacencyMap, err := h.AdjacencyMap()
	if err != nil {
		return union, fmt.Errorf("failed to get adjacency map: %w", err)
	}

	addedEdges := make(map[K]map[K]struct{})

	for currentHash := range adjacencyMap {
		vertex, err := h.Vertex(currentHash)
		if err != nil {
			return union, fmt.Errorf("failed to get vertex %v: %w", currentHash, err)
		}

		err = union.AddVertex(vertex)
		if err != nil && err != graph.ErrVertexAlreadyExists {
			return union, fmt.Errorf("failed to add vertex %v: %w", currentHash, err)
		}
	}

	for _, adjacencies := range adjacencyMap {
		for _, edge := range adjacencies {
			if _, sourceOK := addedEdges[edge.Source]; sourceOK {
				if _, targetOK := addedEdges[edge.Source][edge.Target]; targetOK {
					// If the edge addedEdges[source][target] exists, the edge
					// has already been created and thus can be skipped here.
					continue
				}
			}

			err = union.AddEdge(edge.Source, edge.Target)
			if err != nil && err != graph.ErrEdgeAlreadyExists {
				return union, fmt.Errorf("failed to add edge (%v, %v): %w", edge.Source, edge.Target, err)
			}

			if _, ok := addedEdges[edge.Source]; !ok {
				addedEdges[edge.Source] = make(map[K]struct{})
			}
			addedEdges[edge.Source][edge.Target] = struct{}{}
		}
	}

	return union, nil
}

func FindSources[K comparable, T any](g graph.Graph[K, T]) ([]K, error) {
	if !g.Traits().IsDirected {
		return nil, fmt.Errorf("cannot find source of a non-DAG graph ")
	}

	predecessorMap, err := g.PredecessorMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get predecessor map: %w", err)
	}

	var sources []K
	for vertex, predecessors := range predecessorMap {
		if len(predecessors) == 0 {
			sources = append(sources, vertex)
		}
	}
	return sources, nil
}