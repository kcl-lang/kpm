package semver

import (
	"testing"

	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/errors"
)

func TestLatestVersion(t *testing.T) {
	latest, err := LatestVersion([]string{"1.2.3", "1.4.0", "1.3.5", "1.0.0"})
	assert.Equal(t, err, nil)
	assert.Equal(t, latest, "1.4.0")

	latest, err = LatestVersion([]string{})
	assert.Equal(t, err, errors.InvalidVersionFormat)
	assert.Equal(t, latest, "")

	latest, err = LatestVersion([]string{"invalid_version"})
	assert.Equal(t, err.Error(), "failed to parse version invalid_version\nMalformed version: invalid_version\n")
	assert.Equal(t, latest, "")

	latest, err = LatestVersion([]string{"1.2.3", "1.4.0", "1.3.5", "invalid_version"})
	assert.Equal(t, err.Error(), "failed to parse version invalid_version\nMalformed version: invalid_version\n")
	assert.Equal(t, latest, "")
}

func TestTheLatestTagWithMissingVersion(t *testing.T) {
	latest, err := LatestVersion([]string{"1.2", "1.4", "1.3", "1.0", "5"})
	assert.Equal(t, err, nil)
	assert.Equal(t, latest, "5")

	latest, err = LatestVersion([]string{"1.2", "1.4", "1.3", "1.0", "5.5.5"})
	assert.Equal(t, err, nil)
	assert.Equal(t, latest, "5.5.5")

	latest, err = LatestVersion([]string{"1.2", "1.4", "1.3", "1.0", "5.5"})
	assert.Equal(t, err, nil)
	assert.Equal(t, latest, "5.5")
}

func TestOldestVersion(t *testing.T) {
	oldest, err := OldestVersion([]string{"1.2.3", "1.4.0", "2.0.0", "1.3.5", "1.0.0"})
	assert.Equal(t, err, nil)
	assert.Equal(t, oldest, "1.0.0")

	oldest, err = OldestVersion([]string{"2.2.0", "2.4.0", "3.0.0", "2.3.5"})
	assert.Equal(t, err, nil)
	assert.Equal(t, oldest, "2.2.0")
}

func TestFilterCompatibleVersions(t *testing.T) {
	compatible, err := filterCompatibleVersions([]string{"1.2.3", "1.4.0", "2.0.0", "1.3.5", "1.0.0"}, "1.2.0")
	assert.Equal(t, err, nil)
	expCompatible := []string{"1.2.3", "1.4.0", "1.3.5", "1.0.0"}
	for i, v := range compatible {
		assert.Equal(t, v, expCompatible[i])
	}

	compatible, err = filterCompatibleVersions([]string{"2.2.0", "2.4.0", "3.0.0", "2.3.5"}, "2.0.0")
	assert.Equal(t, err, nil)
	expCompatible = []string{"2.2.0", "2.4.0", "2.3.5"}
	for i, v := range compatible {
		assert.Equal(t, v, expCompatible[i])
	}
}
