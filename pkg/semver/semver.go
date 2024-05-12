package semver

import (
	"fmt"

	"github.com/hashicorp/go-version"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/reporter"
)

func LatestVersion(versions []string) (string, error) {
	var latest *version.Version
	for _, v := range versions {
		ver, err := version.NewVersion(v)
		if err != nil {
			return "", reporter.NewErrorEvent(reporter.FailedParseVersion, err, fmt.Sprintf("failed to parse version %s", v))
		}
		if latest == nil || ver.GreaterThan(latest) {
			latest = ver
		}
	}

	if latest == nil {
		return "", errors.InvalidVersionFormat
	}

	return latest.Original(), nil
}

// LeastOldVersion returns the version that is most recent and less than the base version.
func LeastOldVersion(versions []string, baseVersion string) (string, error) {
	base, err := version.NewVersion(baseVersion)
	if err != nil {
		return "", fmt.Errorf("invalid base version: %v", err)
	}

	var leastOld *version.Version
	for _, v := range versions {
		ver, err := version.NewVersion(v)
		if err != nil {
			return "", reporter.NewErrorEvent(reporter.FailedParseVersion, err, fmt.Sprintf("failed to parse version %s", v))
		}

		// Only consider versions less than the base version
		if ver.LessThan(base) {
			if leastOld == nil || ver.GreaterThan(leastOld) {
				leastOld = ver
			}
		}
	}

	if leastOld == nil {
		return "", errors.InvalidVersionFormat
	}

	return leastOld.Original(), nil
}

func filterCompatibleVersions(versions []string, baseVersion string) ([]string, error) {
	base, err := version.NewVersion(baseVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid base version: %v", err)
	}
	var compatibleVersions []string
	for _, v := range versions {
		ver, err := version.NewVersion(v)
		if err != nil {
			continue // skip versions that fail to parse
		}
		if ver.Segments()[0] == base.Segments()[0] && ver.Prerelease() == "" {
			compatibleVersions = append(compatibleVersions, ver.Original())
		}
	}
	return compatibleVersions, nil
}

func LatestCompatibleVersion(versions []string, baseVersion string) (string, error) {
	compatibleVersions, err := filterCompatibleVersions(versions, baseVersion)
	if err != nil {
		return "", err
	}
	return LatestVersion(compatibleVersions)
}

func LeastOldCompatibleVersion(versions []string, baseVersion string) (string, error) {
	compatibleVersions, err := filterCompatibleVersions(versions, baseVersion)
	if err != nil {
		return "", err
	}
	return LeastOldVersion(compatibleVersions, baseVersion)
}
