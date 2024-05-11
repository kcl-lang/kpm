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
	oldest, err := OldestVersion([]string{"1.2.3", "1.4.0", "1.3.5", "1.0.0"})
	assert.Equal(t, err, nil)
	assert.Equal(t, oldest, "1.0.0")

	oldest, err = OldestVersion([]string{})
	assert.Equal(t, err, errors.InvalidVersionFormat)
	assert.Equal(t, oldest, "")

	oldest, err = OldestVersion([]string{"invalid_version"})
	assert.Equal(t, err.Error(), "failed to parse version invalid_version\nMalformed version: invalid_version\n")
	assert.Equal(t, oldest, "")

	oldest, err = OldestVersion([]string{"1.2.3", "1.4.0", "1.3.5", "invalid_version"})
	assert.Equal(t, err.Error(), "failed to parse version invalid_version\nMalformed version: invalid_version\n")
	assert.Equal(t, oldest, "")
}

func TestOldestVersionWithVariousFormats(t *testing.T) {
	oldest, err := OldestVersion([]string{"2.2", "2.4.5", "2.3.9", "2.1.0", "2.0"})
	assert.Equal(t, err, nil)
	assert.Equal(t, oldest, "2.0")

	oldest, err = OldestVersion([]string{"0.1", "0.1.1", "0.1.2-beta", "0.0.9"})
	assert.Equal(t, err, nil)
	assert.Equal(t, oldest, "0.0.9")

	oldest, err = OldestVersion([]string{"3.3.3", "3.2", "3.1", "3.0.0"})
	assert.Equal(t, err, nil)
	assert.Equal(t, oldest, "3.0.0")
}
