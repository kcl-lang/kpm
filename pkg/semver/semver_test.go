package semver

import (
	"testing"

	"gotest.tools/v3/assert"
	"kusionstack.io/kpm/pkg/errors"
)

func TestLatestVersion(t *testing.T) {
	latest, err := LatestVersion([]string{"1.2.3", "1.4.0", "1.3.5", "1.0.0"})
	assert.Equal(t, err, nil)
	assert.Equal(t, latest, "1.4.0")

	latest, err = LatestVersion([]string{})
	assert.Equal(t, err, errors.InvalidVersionFormat)
	assert.Equal(t, latest, "")

	latest, err = LatestVersion([]string{"invalid_version"})
	assert.Equal(t, err, errors.InvalidVersionFormat)
	assert.Equal(t, latest, "")

	latest, err = LatestVersion([]string{"1.2.3", "1.4.0", "1.3.5", "invalid_version"})
	assert.Equal(t, err, nil)
	assert.Equal(t, latest, "1.4.0")
}
