package mvs

import (
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/hashicorp/go-version"
	"golang.org/x/mod/module"
	"kcl-lang.io/kpm/pkg/3rdparty/mvs"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/downloader"
	errInt "kcl-lang.io/kpm/pkg/errors"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/semver"
)

type ReqsGraph struct {
	graph.Graph[module.Version, module.Version]
	KpmClient *client.KpmClient
	KpmPkg    *pkg.KclPkg
}

func (r ReqsGraph) Max(path, v1, v2 string) string {
	if v1 == "none" || v2 == "" {
		return v2
	}
	if v2 == "none" || v1 == "" {
		return v1
	}
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

func getRepoNameFromURL(gitURL string) string {
	// Remove the trailing ".git" if present
	gitURL = strings.TrimSuffix(gitURL, ".git")

	// Split the URL by the last forward slash
	parts := strings.Split(gitURL, "/")

	// The last part of the URL should be the repository name
	return parts[len(parts)-1]
}

func (r ReqsGraph) Upgrade(m module.Version) (module.Version, error) {
	_, properties, err := r.VertexWithProperties(m)
	if err != nil {
		return module.Version{}, err
	}

	// there must be only one property depending on the download source type
	if len(properties.Attributes) != 1 {
		return module.Version{}, errInt.MultipleSources
	}

	var releases []string
	for sourceType, uri := range properties.Attributes {
		releases, err = client.GetReleasesFromSource(sourceType, uri)
		if err != nil {
			return module.Version{}, err
		}
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
			if sourceType == "git" {
				repoName := getRepoNameFromURL(uri)
				if repoName == d.Name {
					continue
				}
				source := downloader.Source{}
				source.Git = &downloader.Git{
					Url:     uri,
					Tag:     m.Version,
					Package: d.Name,
				}
				d.Source = source
			}
		}
		mpp := orderedmap.NewOrderedMap[string, pkg.Dependency]()
		mpp.Set(m.Path, d)
		deps := pkg.Dependencies{
			Deps: mpp,
		}
		lockDeps := pkg.Dependencies{
			Deps: orderedmap.NewOrderedMap[string, pkg.Dependency](),
		}
		_, err = r.KpmClient.DownloadDeps(&deps, &lockDeps, r.Graph, r.KpmPkg.HomePath, module.Version{})
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

	// there must be only one property depending on the download source type
	if len(properties.Attributes) != 1 {
		return module.Version{}, errInt.MultipleSources
	}

	var releases []string
	for sourceType, uri := range properties.Attributes {
		releases, err = client.GetReleasesFromSource(sourceType, uri)
		if err != nil {
			return module.Version{}, err
		}
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
		mppDeps := orderedmap.NewOrderedMap[string, pkg.Dependency]()
		mppDeps.Set(m.Path, d)
		deps := pkg.Dependencies{
			Deps: mppDeps,
		}
		lockDeps := pkg.Dependencies{
			Deps: orderedmap.NewOrderedMap[string, pkg.Dependency](),
		}
		_, err = r.KpmClient.DownloadDeps(&deps, &lockDeps, r.Graph, r.KpmPkg.HomePath, module.Version{})
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

// UpdateBuildList decides whether to upgrade or downgrade based on modulesToUpgrade and modulesToDowngrade.
// if modulesToUpgrade is empty, upgrade all dependencies. if modulesToUpgrade is not empty, upgrade the dependencies.
// if modulesToDowngrade is not empty, downgrade the dependencies.
// if modulesToUpgrade and modulesToDowngrade are both empty, first apply upgrade operation and
// then downgrade the build list returned from previous operation.
func UpdateBuildList(target module.Version, modulesToUpgrade []module.Version, modulesToDowngrade []module.Version, reqs *ReqsGraph) ([]module.Version, error) {
	var (
		UpdBuildLists []module.Version
		err           error
	)

	if len(modulesToUpgrade) == 0 {
		UpdBuildLists, err = mvs.UpgradeAll(target, reqs)
	} else {
		UpdBuildLists, err = mvs.Upgrade(target, reqs, modulesToUpgrade...)
	}
	if err != nil {
		return []module.Version{}, err
	}

	if len(modulesToDowngrade) != 0 {
		UpdBuildLists, err = mvs.Downgrade(target, reqs, modulesToDowngrade...)
	}
	if err != nil {
		return []module.Version{}, err
	}

	return UpdBuildLists, nil
}
