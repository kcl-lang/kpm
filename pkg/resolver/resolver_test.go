package resolver

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/env"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/settings"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func TestResolver(t *testing.T) {
	resolve_path := getTestDir("test_resolve_graph")
	pkgPath := filepath.Join(resolve_path, "pkg")
	defaultCachePath, err := env.GetAbsPkgPath()
	if err != nil {
		t.Fatal(err)
	}

	var res []string
	var buf bytes.Buffer

	resolver := DepsResolver{
		Downloader:       &downloader.DepDownloader{},
		Settings:         settings.GetSettings(),
		LogWriter:        &buf,
		DefaultCachePath: defaultCachePath,
		ResolveFuncs: []resolveFunc{func(dep *pkg.Dependency, parentPkg *pkg.KclPkg) error {
			res = append(res, fmt.Sprintf("%s -> %s", parentPkg.GetPkgName(), dep.Name))
			return nil
		}},
	}

	err = resolver.Resolve(
		WithEnableCache(true),
		WithSourceUrl(pkgPath),
	)

	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"dep1 -> helloworld",
		"pkg -> dep1",
		"pkg -> helloworld",
	}

	sort.Strings(res)
	assert.Equal(t, len(res), 3)
	assert.Equal(t, res, expected)
}
