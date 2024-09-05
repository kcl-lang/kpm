package client

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
)

func TestResolver(t *testing.T) {
	kpmcli, err := NewKpmClient()
	if err != nil {
		t.Fatal(err)
	}

	resolve_path := getTestDir("test_resolve_graph")
	pkgPath := filepath.Join(resolve_path, "pkg")

	pkgSource, err := downloader.NewSourceFromStr(pkgPath)
	if err != nil {
		t.Fatal(err)
	}

	var res []string
	var buf bytes.Buffer

	kpmcli.SetLogWriter(&buf)
	resolver := NewDepsResolver(kpmcli)
	resolver.AddResolveFunc(func(dep *pkg.Dependency, parentPkg *pkg.KclPkg) error {
		res = append(res, fmt.Sprintf("%s -> %s", parentPkg.GetPkgName(), dep.Name))
		return nil
	})

	err = resolver.Resolve(
		WithEnableCache(true),
		WithResolveSource(pkgSource),
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
	assert.Equal(t, buf.String(), "")
}
