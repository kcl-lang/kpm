package client

import pkg "kcl-lang.io/kpm/pkg/package"

// Construct dependency graph
type DependencyGraph map[string][]string

// Function to construct dependency graph by parsing kcl.mod file
func ConstructDependencyGraphFromModFile(kpmClient *KpmClient, kclPkg *pkg.KclPkg) (DependencyGraph, error) {
	dependencies, err := kpmClient.ParseKclModFile(kclPkg)
	if err != nil {
		return nil, err
	}
	return ConstructDependencyGraph(dependencies), nil
}

// Function to construct dependency graph from dependency map
func ConstructDependencyGraph(dependencies map[string][]string) DependencyGraph {
	graph := make(DependencyGraph)
	for dependency, versions := range dependencies {
		graph[dependency] = versions
	}
	return graph
}

// Traverse dependency graph using depth-first search (DFS)
func DFS(graph DependencyGraph, dependency string, visited map[string]bool, result []string) []string {
	// Mark current dependency as visited
	visited[dependency] = true

	// Add current dependency to result
	result = append(result, dependency)

	// Recursively traverse dependencies
	for _, dep := range graph[dependency] {
		if !visited[dep] {
			result = DFS(graph, dep, visited, result)
		}
	}

	return result
}

// Output dependencies in the same format as go mod graph
func OutputDependencies(graph DependencyGraph) []string {
	result := make([]string, 0)
	visited := make(map[string]bool)

	// Traverse each dependency using DFS
	for dependency := range graph {
		if !visited[dependency] {
			result = DFS(graph, dependency, visited, result)
		}
	}

	return result
}
