package semver

import (
	"github.com/hashicorp/go-version"
	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/reporter"
)

func LatestVersion(versions []string) (string, error) {
	var latest *version.Version
	for _, v := range versions {
		ver, err := version.NewVersion(v)
		if err != nil {
			reporter.Report("kpm: failed to parse version", v, err)
			continue
		}
		if latest == nil || ver.GreaterThan(latest) {
			latest = ver
		}
	}

	if latest == nil {
		return "", errors.InvalidVersionFormat
	}

	return latest.String(), nil
}
