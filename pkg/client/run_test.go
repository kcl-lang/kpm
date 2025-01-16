package client

import (
	"bytes"
	"os"
	"testing"

	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/utils"
)

func TestFixAddGitDep(t *testing.T) {
	modPath := getTestDir("test_add_git_dep")
	kpmcli, err := NewKpmClient()
	if err != nil {
		t.Fatal(err)
	}

	tmpKpmHome, err := os.MkdirTemp("", "kpm_home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpKpmHome)

	kpmcli.SetHomePath(tmpKpmHome)

	var buf bytes.Buffer
	kpmcli.SetLogWriter(&buf)

	res, err := kpmcli.Run(
		WithRunSource(
			&downloader.Source{
				Local: &downloader.Local{
					Path: modPath,
				},
			},
		),
	)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, utils.RmNewline(res.GetRawYamlResult()), "name: flask_manifestsreplicas: 1labels:  app: flask_manifests")
	assert.Equal(t, buf.String(), "cloning 'git://github.com/kcl-lang/flask-demo-kcl-manifests.git' with tag 'v0.1.0'\n")
}
