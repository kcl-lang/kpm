package client

import (
	"kcl-lang.io/kpm/pkg/downloader"
	pkg "kcl-lang.io/kpm/pkg/package"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"gotest.tools/v3/assert"
	"kcl-lang.io/kpm/pkg/features"
	"kcl-lang.io/kpm/pkg/utils"
)

func TestUpdate(t *testing.T) {
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestUpdateWithKclMod", TestFunc: testUpdateWithKclMod}})
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestUpdateWithKclModlock", TestFunc: testUpdateWithKclModlock}})
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestUpdateWithNoSumCheck", TestFunc: testUpdateWithNoSumCheck}})
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestUpdateDefaultRegistryDep", TestFunc: testUpdateDefaultRegistryDep}})
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestUpdateWithKclModAndLock", TestFunc: testUpdateKclModAndLock}})
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestUpdate", TestFunc: testUpdate}})
}

func testUpdate(t *testing.T, kpmcli *KpmClient) {
	features.Enable(features.SupportMVS)
	testDir := getTestDir("test_update_with_mvs")

	updates := []struct {
		name   string
		before func() error
	}{
		{
			name:   "update_0",
			before: func() error { return nil },
		},
		{
			name: "update_1",
			before: func() error {
				if err := copy.Copy(filepath.Join(testDir, "update_1", "pkg", "kcl.mod.bk"), filepath.Join(testDir, "update_1", "pkg", "kcl.mod")); err != nil {
					return err
				}
				if err := copy.Copy(filepath.Join(testDir, "update_1", "pkg", "kcl.mod.lock.bk"), filepath.Join(testDir, "update_1", "pkg", "kcl.mod.lock")); err != nil {
					return err
				}
				return nil
			},
		},
	}

	for _, update := range updates {
		if err := update.before(); err != nil {
			t.Fatal(err)
		}

		kpkg, err := kpmcli.LoadPkgFromPath(filepath.Join(testDir, update.name, "pkg"))
		if err != nil {
			t.Fatal(err)
		}

		_, err = kpmcli.Update(WithUpdatedKclPkg(kpkg))
		if err != nil {
			t.Fatal(err)
		}

		expectedMod, err := os.ReadFile(filepath.Join(testDir, update.name, "pkg", "kcl.mod.expect"))
		if err != nil {
			t.Fatal(err)
		}

		expectedModLock, err := os.ReadFile(filepath.Join(testDir, update.name, "pkg", "kcl.mod.lock.expect"))
		if err != nil {
			t.Fatal(err)
		}

		gotMod, err := os.ReadFile(filepath.Join(testDir, update.name, "pkg", "kcl.mod"))
		if err != nil {
			t.Fatal(err)
		}

		gotModLock, err := os.ReadFile(filepath.Join(testDir, update.name, "pkg", "kcl.mod.lock"))
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(string(expectedMod)), utils.RmNewline(string(gotMod)))
		assert.Equal(t, utils.RmNewline(string(expectedModLock)), utils.RmNewline(string(gotModLock)))
	}
}

func TestNeedCheckSum(t *testing.T) {
	type args struct {
		dep pkg.Dependency
	}
	tests := []struct {
		name string
		env  string
		args args
		want bool
	}{
		{
			name: "local_path_with_full_match",
			env:  "/path",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   nil,
						Local: &downloader.Local{Path: "/path"},
					},
				},
			},
			want: false,
		},
		{
			name: "local_path_with_prefix_match",
			env:  "/path",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   nil,
						Local: &downloader.Local{Path: "/path/pkg"},
					},
				},
			},
			want: false,
		},
		{
			name: "local_path_with_suffix_match",
			env:  "/path/*/pkg",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   nil,
						Local: &downloader.Local{Path: "/path/foo/pkg"},
					},
				},
			},
			want: false,
		},
		{
			name: "local_path_without_match",
			env:  "/foo/bar",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   nil,
						Local: &downloader.Local{Path: "/path/foo/bar"},
					},
				},
			},
			want: true,
		},
		{
			name: "oci_path_with_full_match",
			env:  "ghcr.io/kcl-lang/konfig",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   &downloader.Oci{Reg: "ghcr.io", Repo: "kcl-lang/konfig"},
						Local: nil,
					},
				},
			},
			want: false,
		},
		{
			name: "oci_path_with_prefix_match",
			env:  "ghcr.io/kcl-lang",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   &downloader.Oci{Reg: "ghcr.io", Repo: "kcl-lang/konfig"},
						Local: nil,
					},
				},
			},
			want: false,
		},
		{
			name: "oci_path_with_suffix_match",
			env:  "ghcr.io/kcl-lang/*",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   &downloader.Oci{Reg: "ghcr.io", Repo: "kcl-lang/konfig"},
						Local: nil,
					},
				},
			},
			want: false,
		},
		{
			name: "oci_path_with_suffix_match2",
			env:  "ghcr.io/*/konfig",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   &downloader.Oci{Reg: "ghcr.io", Repo: "kcl-lang/konfig"},
						Local: nil,
					},
				},
			},
			want: false,
		},
		{
			name: "oci_path_without_match",
			env:  "ghcr.io/kcl-lang/foo",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   &downloader.Oci{Reg: "ghcr.io", Repo: "kcl-lang/konfig"},
						Local: nil,
					},
				},
			},
			want: true,
		},
		{
			name: "oci_path_without_match2",
			env:  "docker.io/kcl-lang/foo",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   &downloader.Oci{Reg: "ghcr.io", Repo: "kcl-lang/konfig"},
						Local: nil,
					},
				},
			},
			want: true,
		},
		{
			name: "git_path_with_full_match",
			env:  "github.com/kcl-lang/konfig",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   &downloader.Git{Url: "github.com/kcl-lang/konfig"},
						Oci:   nil,
						Local: nil,
					},
				},
			},
			want: false,
		},
		{
			name: "git_path_with_prefix_match",
			env:  "github.com/kcl-lang",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   &downloader.Git{Url: "github.com/kcl-lang/konfig"},
						Oci:   nil,
						Local: nil,
					},
				},
			},
			want: false,
		},
		{
			name: "git_path_with_suffix_match",
			env:  "github.com/*/konfig",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   &downloader.Git{Url: "github.com/kcl-lang/konfig"},
						Oci:   nil,
						Local: nil,
					},
				},
			},
			want: false,
		},
		{
			name: "git_path_without_match",
			env:  "github.com/kcl-lang/foo",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   &downloader.Git{Url: "github.com/kcl-lang/konfig"},
						Oci:   nil,
						Local: nil,
					},
				},
			},
			want: true,
		},
		{
			name: "git_path_without_match2",
			env:  "gitea.com/kcl-lang/foo",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   &downloader.Git{Url: "github.com/kcl-lang/konfig"},
						Oci:   nil,
						Local: nil,
					},
				},
			},
			want: true,
		},
		{
			name: "multiple_path_with_match",
			env:  "github.com/kcl-lang/*,ghcr.io/kcl-lang/*,/path",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   &downloader.Git{Url: "github.com/kcl-lang/konfig"},
						Oci:   nil,
						Local: nil,
					},
				},
			},
			want: false,
		},
		{
			name: "multiple_path_with_match2",
			env:  "github.com/kcl-lang/*,ghcr.io/kcl-lang/*,/path",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   &downloader.Oci{Reg: "ghcr.io", Repo: "kcl-lang/konfig"},
						Local: nil,
					},
				},
			},
			want: false,
		},
		{
			name: "multiple_path_with_match3",
			env:  "github.com/kcl-lang/*,ghcr.io/kcl-lang/*,/path",
			args: args{
				dep: pkg.Dependency{
					Source: downloader.Source{
						Git:   nil,
						Oci:   nil,
						Local: &downloader.Local{Path: "/path/foo/bar"},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		err := os.Setenv("KPM_NO_SUM", tt.env)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NeedCheckSum(tt.args.dep), "NeedCheckSum(%v)", tt.args.dep)
		})
		err = os.Unsetenv("KPM_NO_SUM")
		if err != nil {
			t.Fatal(err)
		}
	}
}
