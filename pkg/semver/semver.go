package semver

import (
	"fmt"

	"github.com/hashicorp/go-version"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/reporter"
)

func LatestVersion(versions []string) (string, error) {
	var latest *version.Version
	for _, v := range versions {
		// Do not support the latest version.
		if v == constants.LATEST {
			continue
		}
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

func OldestVersion(versions []string) (string, error) {
	var oldest *version.Version
	for _, v := range versions {
		ver, err := version.NewVersion(v)
		if err != nil {
			return "", reporter.NewErrorEvent(reporter.FailedParseVersion, err, fmt.Sprintf("failed to parse version %s", v))
		}
		if oldest == nil || ver.LessThan(oldest) {
			oldest = ver
		}
	}

	if oldest == nil {
		return "", errors.InvalidVersionFormat
	}

	return oldest.Original(), nil
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
	return OldestVersion(compatibleVersions)
}
