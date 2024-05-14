package mvs

import (
	"fmt"

	"github.com/dominikbraun/graph"
	"github.com/hashicorp/go-version"
	"golang.org/x/mod/module"
	"kcl-lang.io/kpm/pkg/client"
	errInt "kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/git"
	"kcl-lang.io/kpm/pkg/oci"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/semver"
)

type ReqsGraph struct {
	graph.Graph[module.Version, module.Version]
	kpmClient *client.KpmClient
	kpmPkg    *pkg.KclPkg
}

func (r ReqsGraph) Max(path, v1, v2 string) string {
	version1, err := version.NewVersion(v1)
	if err != nil {
		reporter.Fatal(reporter.FailedParseVersion, err, fmt.Sprintf("failed to parse version %s for module %s", v1, path))
		return ""
	}
	version2, err := version.NewVersion(v2)
	if err != nil {
		reporter.Fatal(reporter.FailedParseVersion, err, fmt.Sprintf("failed to parse version %s for module %s", v2, path))
		return ""
	}
	if version1.GreaterThan(version2) {
		return v1
	}
	return v2
}

func (r ReqsGraph) Upgrade(m module.Version) (module.Version, error) {
	_, properties, err := r.VertexWithProperties(m)
	if err != nil {
		return module.Version{}, err
	}

	releases, err := getReleasesFromSource(properties)
	if err != nil {
		return module.Version{}, err
	}

	if releases == nil {
		return m, nil
	}

	m.Version, err = semver.LatestCompatibleVersion(releases, m.Version)
	if err != nil {
		return module.Version{}, err
	}
	_, err = r.Vertex(m)
	if err == graph.ErrVertexNotFound {
		d := pkg.Dependency{
			Name:    m.Path,
			Version: m.Version,
		}
		d.FullName = d.GenDepFullName()
		for sourceType, uri := range properties.Attributes {
			d.Source, err = pkg.GenSource(sourceType, uri, m.Version)
			if err != nil {
				return module.Version{}, err
			}
		}
		deps := pkg.Dependencies{
			Deps: map[string]pkg.Dependency{
				m.Path: d,
			},
		}
		lockDeps := pkg.Dependencies{
			Deps: make(map[string]pkg.Dependency),
		}
		_, err = r.kpmClient.DownloadDeps(deps, lockDeps, r.Graph, r.kpmPkg.HomePath, module.Version{})
		if err != nil {
			return module.Version{}, err
		}
	}
	return m, nil
}

func (r ReqsGraph) Previous(m module.Version) (module.Version, error) {
	_, properties, err := r.VertexWithProperties(m)
	if err != nil {
		return module.Version{}, err
	}

	releases, err := getReleasesFromSource(properties)
	if err != nil {
		return module.Version{}, err
	}

	if releases == nil {
		return m, nil
	}

	// copy the version to compare it later
	v := m.Version

	m.Version, err = semver.LeastOldCompatibleVersion(releases, m.Version)
	if err != nil && err != errInt.InvalidVersionFormat {
		return module.Version{}, err
	}

	if v == m.Version {
		return module.Version{Path: m.Path, Version: "none"}, nil
	}

	_, err = r.Vertex(m)
	if err == graph.ErrVertexNotFound {
		d := pkg.Dependency{
			Name:    m.Path,
			Version: m.Version,
		}
		d.FullName = d.GenDepFullName()
		for sourceType, uri := range properties.Attributes {
			d.Source, err = pkg.GenSource(sourceType, uri, m.Version)
			if err != nil {
				return module.Version{}, err
			}
		}
		deps := pkg.Dependencies{
			Deps: map[string]pkg.Dependency{
				m.Path: d,
			},
		}
		lockDeps := pkg.Dependencies{
			Deps: make(map[string]pkg.Dependency),
		}
		_, err = r.kpmClient.DownloadDeps(deps, lockDeps, r.Graph, r.kpmPkg.HomePath, module.Version{})
		if err != nil {
			return module.Version{}, err
		}
	}
	return m, nil
}

func (r ReqsGraph) Required(m module.Version) ([]module.Version, error) {
	adjMap, err := r.AdjacencyMap()
	if err != nil {
		return nil, err
	}
	var reqs []module.Version
	for v := range adjMap[m] {
		reqs = append(reqs, v)
	}
	return reqs, nil
}

func getReleasesFromSource(properties graph.VertexProperties) ([]string, error) {
	var releases []string
	var err error

	// there must be only one property depending on the download source type
	if len(properties.Attributes) != 1 {
		return nil, errInt.MultipleSources
	}

	for k, v := range properties.Attributes {
		switch k {
		case pkg.GIT:
			releases, err = git.GetAllGithubReleases(v)
		case pkg.OCI:
			releases, err = oci.GetAllImageTags(v)
		}
		if err != nil {
			return nil, err
		}
	}

	return releases, nil
}
