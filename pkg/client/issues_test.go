package client

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/features"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/utils"
)

func TestKclIssue1760(t *testing.T) {
    testPath := "github.com/kcl-lang/kcl/issues/1760"
    testCases := []struct {
        name     string
        setup    func()
    }{
        {
            name: "Default",
            setup: func() {
                features.Disable(features.SupportNewStorage)
                features.Disable(features.SupportMVS)
            },
        },
        {
            name: "SupportNewStorage",
            setup: func() {
                features.Enable(features.SupportNewStorage)
                features.Disable(features.SupportMVS)
            },
        },
        {
            name: "SupportMVS",
            setup: func() {
                features.Disable(features.SupportNewStorage)
                features.Enable(features.SupportMVS)
            },
        },
        {
            name: "SupportNewStorageAndMVS",
            setup: func() {
                features.Enable(features.SupportNewStorage)
                features.Enable(features.SupportMVS)
            },
        },
    }

    for _, tc := range testCases {
        tc := tc // capture range variable
        t.Run(tc.name, func(t *testing.T) {
            tc.setup()

            testFunc := func(t *testing.T, kpmcli *KpmClient) {
                rootPath := getTestDir("issues")
                mainKFilePath := filepath.Join(rootPath, testPath, "a", "main.k")
                var buf bytes.Buffer
                kpmcli.SetLogWriter(&buf)

                res, err := kpmcli.Run(
                    WithRunSource(
                        &downloader.Source{
                            Local: &downloader.Local{
                                Path: mainKFilePath,
                            },
                        },
                    ),
                )

                if err != nil {
                    t.Fatal(err)
                }

                assert.Contains(t,
                    utils.RmNewline(buf.String()),
                    "downloading 'kcl-lang/fluxcd-source-controller:v1.3.2' from 'ghcr.io/kcl-lang/fluxcd-source-controller:v1.3.2'",
                )
                assert.Contains(t,
                    utils.RmNewline(buf.String()),
                    "downloading 'kcl-lang/k8s:1.31.2' from 'ghcr.io/kcl-lang/k8s:1.31.2'",
                )

                assert.Contains(t,
                    utils.RmNewline(buf.String()),
                    "downloading 'kcl-lang/fluxcd-helm-controller:v1.0.3' from 'ghcr.io/kcl-lang/fluxcd-helm-controller:v1.0.3'",
                )
                assert.Equal(t, res.GetRawYamlResult(), "The_first_kcl_program: Hello World!")
            }

            RunTestWithGlobalLockAndKpmCli(t, testPath, testFunc)
        })
    }
}

func TestKpmIssue550(t *testing.T) {
	testPath := "github.com/kcl-lang/kpm/issues/550"
	testCases := []struct {
		name     string
		setup    func()
		expected string
	}{
		{
			name: "Default",
			setup: func() {
				features.Disable(features.SupportNewStorage)
				features.Disable(features.SupportMVS)
			},
			expected: filepath.Join("flask-demo-kcl-manifests_test-branch-without-modfile", "aa", "cc"),
		},
		{
			name: "SupportNewStorage",
			setup: func() {
				features.Enable(features.SupportNewStorage)
				features.Disable(features.SupportMVS)
			},
			expected: filepath.Join("git", "src", "200297ed26e4aeb7", "flask-demo-kcl-manifests", "test-branch-without-modfile", "aa", "cc"),
		},
		{
			name: "SupportMVS",
			setup: func() {
				features.Disable(features.SupportNewStorage)
				features.Enable(features.SupportMVS)
			},
			expected: filepath.Join("flask-demo-kcl-manifests_test-branch-without-modfile", "aa", "cc"),
		},
		{
			name: "SupportNewStorageAndMVS",
			setup: func() {
				features.Enable(features.SupportNewStorage)
				features.Enable(features.SupportMVS)
			},
			expected: filepath.Join("git", "src", "200297ed26e4aeb7", "flask-demo-kcl-manifests", "test-branch-without-modfile", "aa", "cc"),
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			testFunc := func(t *testing.T, kpmcli *KpmClient) {
				rootPath := getTestDir("issues")
				modPath := filepath.Join(rootPath, testPath, "pkg")
				var buf bytes.Buffer
				kpmcli.SetLogWriter(&buf)

				tmpKpmHome, err := os.MkdirTemp("", "")
				if err != nil {
					t.Fatal(err)
				}
				defer os.RemoveAll(tmpKpmHome)

				kpmcli.homePath = tmpKpmHome

				kMod, err := pkg.LoadKclPkgWithOpts(
					pkg.WithPath(modPath),
				)

				if err != nil {
					t.Fatal(err)
				}

				res, err := kpmcli.ResolveDepsIntoMap(kMod)

				if err != nil {
					t.Fatal(err)
				}
				fmt.Printf("buf.String(): %v\n", buf.String())
				assert.Contains(t,
					utils.RmNewline(buf.String()),
					"cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with branch 'test-branch-without-modfile'",
				)
				assert.Equal(t, len(res), 1)
				assert.Equal(t, res["cc"], filepath.Join(tmpKpmHome, tc.expected))
			}

			RunTestWithGlobalLockAndKpmCli(t, testPath, testFunc)
		})
	}
}
