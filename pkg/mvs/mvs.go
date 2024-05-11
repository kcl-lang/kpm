package mvs

import (
	"fmt"

	"github.com/dominikbraun/graph"
	"github.com/hashicorp/go-version"
	"golang.org/x/mod/module"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/git"
	"kcl-lang.io/kpm/pkg/oci"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/semver"
)

type ReqsGraph struct {
	graph.Graph[module.Version, module.Version]
}

func (r ReqsGraph) Max(_, v1, v2 string) string {
	version1, err := version.NewVersion(v1)
	if err != nil {
		reporter.NewErrorEvent(reporter.FailedParseVersion, err, fmt.Sprintf("failed to parse version %s", v1))
		return ""
	}
	version2, err := version.NewVersion(v2)
	if err != nil {
		reporter.NewErrorEvent(reporter.FailedParseVersion, err, fmt.Sprintf("failed to parse version %s", v2))
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

	m.Version, err = semver.LatestVersion(releases)
	if err != nil {
		return module.Version{}, err
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

	m.Version, err = semver.OldestVersion(releases)
	if err != nil {
		return module.Version{}, err
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
		return nil, errors.MultipleSources
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
