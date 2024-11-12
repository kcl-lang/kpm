package client

import (
	"fmt"
	"path/filepath"

	"github.com/dominikbraun/graph"
	"golang.org/x/mod/module"
	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/resolver"
)

// GraphOptions is the options for creating a dependency graph.
type GraphOptions struct {
	kMod *pkg.KclPkg
}

type GraphOption func(*GraphOptions) error

// WithGraphMod sets the kMod for creating a dependency graph.
func WithGraphMod(kMod *pkg.KclPkg) GraphOption {
	return func(o *GraphOptions) error {
		o.kMod = kMod
		return nil
	}
}

// DepGraph is the dependency graph.
type DepGraph struct {
	gra graph.Graph[module.Version, module.Version]
}

// NewDepGraph creates a new dependency graph.
func NewDepGraph() *DepGraph {
	return &DepGraph{
		gra: graph.New(
			func(m module.Version) module.Version { return m },
			graph.Directed(),
			graph.PreventCycles(),
		),
	}
}

// AddVertex adds a vertex to the dependency graph.
func (g *DepGraph) AddVertex(name, version string) (*module.Version, error) {
	root := module.Version{Path: name, Version: version}
	err := g.gra.AddVertex(root)
	if err != nil && err != graph.ErrVertexAlreadyExists {
		return nil, err
	}
	return &root, nil
}

// AddEdge adds an edge to the dependency graph.
func (g *DepGraph) AddEdge(parent, child module.Version) error {
	err := g.gra.AddEdge(parent, child)
	if err != nil {
		return err
	}
	return nil
}

// DisplayGraphFromVertex displays the dependency graph from the start vertex to string.
func (g *DepGraph) DisplayGraphFromVertex(startVertex module.Version) (string, error) {
	var res string
	adjMap, err := g.gra.AdjacencyMap()
	if err != nil {
		return "", err
	}

	// Print the dependency graph to string.
	err = graph.BFS(g.gra, startVertex, func(source module.Version) bool {
		for target := range adjMap[source] {
			res += fmt.Sprint(format(source), " ", format(target)) + "\n"
		}
		return false
	})
	if err != nil {
		return "", err
	}

	return res, nil
}

// format formats the module version to string.
func format(m module.Version) string {
	formattedMsg := m.Path
	if m.Version != "" {
		formattedMsg += "@" + m.Version
	}
	return formattedMsg
}

// Graph creates a dependency graph for the given KCL Module.
func (c *KpmClient) Graph(opts ...GraphOption) (*DepGraph, error) {
	options := &GraphOptions{}
	for _, o := range opts {
		err := o(options)
		if err != nil {
			return nil, err
		}
	}

	kMod := options.kMod

	if kMod == nil {
		return nil, fmt.Errorf("kMod is required")
	}

	// Create the dependency graph.
	dGraph := NewDepGraph()
	// Take the current KCL module as the start vertex
	dGraph.AddVertex(kMod.GetPkgName(), kMod.GetPkgVersion())

	modDeps := kMod.ModFile.Dependencies.Deps
	if modDeps == nil {
		return nil, fmt.Errorf("kcl.mod dependencies is nil")
	}

	// ResolveFunc is the function for resolving each dependency when traversing the dependency graph.
	resolverFunc := func(dep *pkg.Dependency, parentPkg *pkg.KclPkg) error {
		if dep != nil && parentPkg != nil {
			// Set the dep as a vertex into graph.
			depVertex, err := dGraph.AddVertex(dep.Name, dep.Version)
			if err != nil && err != graph.ErrVertexAlreadyExists {
				return err
			}

			// Create the vertex for the parent package.
			parent := module.Version{
				Path:    parentPkg.GetPkgName(),
				Version: parentPkg.GetPkgVersion(),
			}

			// Add the edge between the parent and the dependency.
			err = dGraph.AddEdge(parent, *depVertex)
			if err != nil {
				if err == graph.ErrEdgeCreatesCycle {
					return fmt.Errorf("adding %s as a dependency results in a cycle", depVertex)
				}
				return err
			}
		}

		return nil
	}

	// Create a new dependency resolver
	depResolver := resolver.DepsResolver{
		DefaultCachePath:      c.homePath,
		InsecureSkipTLSverify: c.insecureSkipTLSverify,
		Downloader:            c.DepDownloader,
		Settings:              &c.settings,
		LogWriter:             c.logWriter,
	}
	depResolver.ResolveFuncs = append(depResolver.ResolveFuncs, resolverFunc)

	for _, depName := range modDeps.Keys() {
		dep, ok := modDeps.Get(depName)
		if !ok {
			return nil, fmt.Errorf("failed to get dependency %s", depName)
		}

		// Check if the dependency is a local path and it is not an absolute path.
		// If it is not an absolute path, transform the path to an absolute path.
		var depSource *downloader.Source
		if dep.Source.IsLocalPath() && !filepath.IsAbs(dep.Source.Local.Path) {
			depSource = &downloader.Source{
				Local: &downloader.Local{
					Path: filepath.Join(kMod.HomePath, dep.Source.Local.Path),
				},
			}
		} else {
			depSource = &dep.Source
		}

		err := resolverFunc(&dep, kMod)
		if err != nil {
			return nil, err
		}

		err = depResolver.Resolve(
			resolver.WithEnableCache(true),
			resolver.WithSource(depSource),
		)
		if err != nil {
			return nil, err
		}
	}

	return dGraph, nil
}
