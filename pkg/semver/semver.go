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
