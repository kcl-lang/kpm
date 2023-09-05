package runner

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/reporter"
)

func TestFindModRootFrom(t *testing.T) {
	// find root from local file
	absPath, err := filepath.Abs("./testdata_external/external/main.k")
	assert.Equal(t, err, nil)
	root, err := FindModRootFrom(absPath)
	assert.Equal(t, err, (*reporter.KpmEvent)(nil))
	assert.Equal(t, root, filepath.Dir(absPath))

	// find root from dir
	absPath, err = filepath.Abs("./testdata_external/external/")
	assert.Equal(t, err, nil)
	root, err = FindModRootFrom(absPath)
	assert.Equal(t, err, (*reporter.KpmEvent)(nil))
	assert.Equal(t, root, absPath)

	// find root from kfile parent
	absPath, err = filepath.Abs("./testdata/test_find_mod/sub/main.k")
	assert.Equal(t, err, nil)
	root, err = FindModRootFrom(absPath)
	assert.Equal(t, err, (*reporter.KpmEvent)(nil))
	assert.Equal(t, root, filepath.Dir(filepath.Dir(absPath)))

	// find root from kfile parent
	absPath, err = filepath.Abs("./testdata/test_find_mod/sub")
	assert.Equal(t, err, nil)
	root, err = FindModRootFrom(absPath)
	assert.Equal(t, err, (*reporter.KpmEvent)(nil))
	assert.Equal(t, root, filepath.Dir(absPath))
}

func TestGetSourceKindFrom(t *testing.T) {
	assert.Equal(t, GetSourceKindFrom("./testdata_external/external/main.k"), constants.FileEntry)
	assert.Equal(t, GetSourceKindFrom("main.tar"), constants.TarEntry)
	assert.Equal(t, GetSourceKindFrom("oci://test_url"), constants.UrlEntry)
	assert.Equal(t, GetSourceKindFrom("test_ref:0.0.1"), constants.RefEntry)
	assert.Equal(t, GetSourceKindFrom("invalid input"), "")
}

func TestFindRunEntryFrom(t *testing.T) {
	res, err := FindRunEntryFrom([]string{"./testdata_external/external/main.k", "./testdata_external/external"})
	assert.Equal(t, err, (*reporter.KpmEvent)(nil))
	assert.Equal(t, res.packageSource, "./testdata_external/external")

	res, err = FindRunEntryFrom([]string{"./testdata_external/external/main.k", "./testdata_external/external", "./testdata/test_find_mod/sub/main.k"})
	assert.Equal(t, err.Type(), reporter.CompileFailed)
	assert.Equal(t, strings.Contains(err.Error(), "cannot compile multiple packages"), true)
	assert.Equal(t, res, (*Entry)(nil))
}
