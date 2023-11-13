package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/opt"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

var _ = ginkgo.Describe("Kpm CLI Testing", func() {
	ginkgo.Context("testing 'exec kpm outside a kcl package'", func() {
		testSuites := LoadAllTestSuites(filepath.Join(filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "kpm"), "exec_outside_pkg"))
		testDataRoot := filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "test_data")

		for _, ts := range testSuites {
			// In the for loop, the variable ts is defined outside the scope of the ginkgo.It function.
			// This means that when the ginkgo.It function is executed,
			// it will always use the value of ts from the last iteration of the for loop.
			// To fix this issue, create a new variable inside the loop with the same value as ts,
			// and use that variable inside the ginkgo.It function.
			ts := ts
			ginkgo.It(ts.GetTestSuiteInfo(), func() {
				workspace := GetWorkspace()
				Copy(filepath.Join(testDataRoot, "kcl1.tar"), filepath.Join(workspace, "kcl1.tar"))
				Copy(filepath.Join(testDataRoot, "exist_but_not_tar"), filepath.Join(workspace, "exist_but_not_tar"))

				stdout, stderr, err := ExecKpmWithWorkDir(ts.Input, workspace)

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				expectedStdout := ReplaceAllKeyByValue(ts.ExpectStdout, "<workspace>", workspace)
				expectedStderr := ReplaceAllKeyByValue(ts.ExpectStderr, "<workspace>", workspace)

				// Using regular expressions may miss some cases where the string is empty.
				//
				// Since the login/logout-related test cases will output time information,
				// they cannot be compared with method 'Equal',
				// so 'ContainSubstring' is used to compare the results.
				gomega.Expect(stdout).To(gomega.ContainSubstring(expectedStdout))
				gomega.Expect(stderr).To(gomega.ContainSubstring(expectedStderr))
			})
		}
	})

	ginkgo.Context("testing 'exec kpm inside a kcl package'", func() {
		testSuitesRoot := filepath.Join(filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "kpm"), "exec_inside_pkg")
		testSuites := LoadAllTestSuites(testSuitesRoot)
		testDataRoot := filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "test_data")
		for _, ts := range testSuites {
			ts := ts
			ginkgo.It(ts.GetTestSuiteInfo(), func() {
				workspace := GetWorkspace()

				CopyDir(filepath.Join(testDataRoot, "test_kcl"), filepath.Join(workspace, "test_kcl"))
				CopyDir(filepath.Join(testDataRoot, "a_kcl_pkg_dep_one_pkg"), filepath.Join(workspace, "a_kcl_pkg_dep_one_pkg"))
				CopyDir(filepath.Join(testDataRoot, "a_kcl_pkg_dep_one_pkg_2"), filepath.Join(workspace, "a_kcl_pkg_dep_one_pkg_2"))
				CopyDir(filepath.Join(testDataRoot, "a_kcl_pkg_dep_one_pkg_3"), filepath.Join(workspace, "a_kcl_pkg_dep_one_pkg_3"))
				CopyDir(filepath.Join(testDataRoot, "a_kcl_pkg_dep_one_pkg_4"), filepath.Join(workspace, "a_kcl_pkg_dep_one_pkg_4"))
				CopyDir(filepath.Join(testDataRoot, "a_kcl_pkg_dep_one_pkg_5"), filepath.Join(workspace, "a_kcl_pkg_dep_one_pkg_5"))
				CopyDir(filepath.Join(testDataRoot, "an_invalid_kcl_pkg"), filepath.Join(workspace, "an_invalid_kcl_pkg"))

				input := ReplaceAllKeyByValue(ts.Input, "<workspace>", workspace)

				stdout, stderr, err := ExecKpmWithWorkDir(input, filepath.Join(workspace, "test_kcl"))

				expectedStdout := ReplaceAllKeyByValue(ts.ExpectStdout, "<workspace>", workspace)
				expectedStderr := ReplaceAllKeyByValue(ts.ExpectStderr, "<workspace>", workspace)

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				if strings.Contains(ts.ExpectStdout, "<un_ordered>") {
					expectedStdout := ReplaceAllKeyByValue(ts.ExpectStdout, "<un_ordered>", "")
					gomega.Expect(RemoveLineOrder(stdout)).To(gomega.ContainSubstring(RemoveLineOrder(expectedStdout)))
				} else if strings.Contains(ts.ExpectStderr, "<un_ordered>") {
					expectedStderr := ReplaceAllKeyByValue(ts.ExpectStderr, "<un_ordered>", "")
					gomega.Expect(RemoveLineOrder(stderr)).To(gomega.ContainSubstring(RemoveLineOrder(expectedStderr)))
				} else {
					gomega.Expect(stdout).To(gomega.ContainSubstring(expectedStdout))
					gomega.Expect(stderr).To(gomega.ContainSubstring(expectedStderr))
				}
			})
		}
	})

	ginkgo.Context("testing kpm workflows", func() {
		testSuitesRoot := filepath.Join(filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "kpm"), "workflows")

		files, _ := os.ReadDir(testSuitesRoot)
		for _, file := range files {
			ginkgo.It(filepath.Join(testSuitesRoot, file.Name()), func() {
				if file.IsDir() {
					testSuites := LoadAllTestSuites(filepath.Join(testSuitesRoot, file.Name()))
					for _, ts := range testSuites {
						ts := ts

						workspace := GetWorkspace()

						stdout, stderr, err := ExecKpmWithWorkDir(ts.Input, workspace)

						expectedStdout := ReplaceAllKeyByValue(ts.ExpectStdout, "<workspace>", workspace)
						expectedStderr := ReplaceAllKeyByValue(ts.ExpectStderr, "<workspace>", workspace)

						gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
						if !IsIgnore(expectedStdout) {
							gomega.Expect(stdout).To(gomega.ContainSubstring(expectedStdout))
						}
						if !IsIgnore(expectedStderr) {
							gomega.Expect(stderr).To(gomega.ContainSubstring(expectedStderr))
						}
					}
				}
			})
		}
	})

	ginkgo.Context("testing 'kpm run '", func() {
		testSuitesRoot := filepath.Join(filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "kpm"), "kpm_run")
		testSuites := LoadAllTestSuites(testSuitesRoot)
		testDataRoot := filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "test_data")
		for _, ts := range testSuites {
			ts := ts
			ginkgo.It(ts.GetTestSuiteInfo(), func() {
				workspace := GetWorkspace()

				CopyDir(filepath.Join(testDataRoot, ts.Name), filepath.Join(workspace, ts.Name))

				input := ReplaceAllKeyByValue(ts.Input, "<workspace>", filepath.Join(workspace, ts.Name))

				stdout, stderr, err := ExecKpmWithWorkDir(input, filepath.Join(workspace, ts.Name))

				expectedStdout := ReplaceAllKeyByValue(ts.ExpectStdout, "<workspace>", workspace)
				expectedStderr := ReplaceAllKeyByValue(ts.ExpectStderr, "<workspace>", workspace)

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(stdout).To(gomega.Equal(expectedStdout))
				gomega.Expect(stderr).To(gomega.Equal(expectedStderr))
			})
		}
	})

	ginkgo.Context("testing 'test oci '", func() {
		testSuitesRoot := filepath.Join(filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "kpm"), "test_oci")
		testSuites := LoadAllTestSuites(testSuitesRoot)
		testDataRoot := filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "test_data")
		for _, ts := range testSuites {
			ts := ts
			ginkgo.It(ts.GetTestSuiteInfo(), func() {
				// 1. Push a package with metadata in OCI manifest
				workspace := GetWorkspace()

				CopyDir(filepath.Join(testDataRoot, ts.Name), filepath.Join(workspace, ts.Name))

				input := ReplaceAllKeyByValue(ts.Input, "<workspace>", filepath.Join(workspace, ts.Name))

				stdout, stderr, err := ExecKpmWithWorkDir(input, filepath.Join(workspace, ts.Name))

				expectedStdout := ReplaceAllKeyByValue(ts.ExpectStdout, "<workspace>", workspace)
				expectedStderr := ReplaceAllKeyByValue(ts.ExpectStderr, "<workspace>", workspace)

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				if !IsIgnore(expectedStdout) {
					gomega.Expect(stdout).To(gomega.ContainSubstring(expectedStdout))
				}
				if !IsIgnore(expectedStderr) {
					gomega.Expect(stderr).To(gomega.ContainSubstring(expectedStderr))
				}

				bytes, err := os.ReadFile(filepath.Join(testDataRoot, "expected_oci_manifest.json"))
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				// 2. fetch the metadata in OCI manifest to check if the metadata is correct
				repo, err := remote.NewRepository("localhost:5001/test/test_push_with_oci_manifest")
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				repo.PlainHTTP = true
				repo.Client = &auth.Client{
					Client: retry.DefaultClient,
					Cache:  auth.DefaultCache,
					Credential: auth.StaticCredential("localhost:5001", auth.Credential{
						Username: "test",
						Password: "1234",
					}),
				}

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				_, manifestContent, err := oras.FetchBytes(context.Background(), repo, "0.0.1", oras.DefaultFetchBytesOptions)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				var manifest_got v1.Manifest
				err = json.Unmarshal([]byte(manifestContent), &manifest_got)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				var manifest_expect v1.Manifest
				err = json.Unmarshal(bytes, &manifest_expect)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(len(manifest_expect.Annotations)).To(gomega.Equal(len(manifest_got.Annotations)))
				gomega.Expect(manifest_expect.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_NAME]).
					To(gomega.Equal(manifest_got.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_NAME]))
				gomega.Expect(manifest_expect.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_VERSION]).
					To(gomega.Equal(manifest_got.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_VERSION]))
				gomega.Expect(manifest_expect.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_DESCRIPTION]).
					To(gomega.Equal(manifest_got.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_DESCRIPTION]))
				gomega.Expect(manifest_expect.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_SUM]).
					To(gomega.Equal(manifest_got.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_SUM]))
			})

			ginkgo.It("testing 'fetch api '", func() {
				kpmcli, err := client.NewKpmClient()
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				jsonstr, err := kpmcli.FetchOciManifestIntoJsonStr(opt.OciFetchOptions{
					FetchBytesOptions: oras.DefaultFetchBytesOptions,
					OciOptions: opt.OciOptions{
						Reg:  "localhost:5001",
						Repo: "test/kcl2",
						Tag:  "0.0.1",
					},
				})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				var manifest_expect v1.Manifest
				err = json.Unmarshal([]byte(jsonstr), &manifest_expect)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(len(manifest_expect.Annotations)).To(gomega.Equal(5))
				gomega.Expect(manifest_expect.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_NAME]).To(gomega.Equal("kcl2"))
				gomega.Expect(manifest_expect.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_VERSION]).To(gomega.Equal("0.0.1"))
				gomega.Expect(manifest_expect.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_DESCRIPTION]).To(gomega.Equal("This is the kcl package named kcl2"))
				gomega.Expect(manifest_expect.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_SUM]).To(gomega.Equal("Y/QXruiaxcJcmOnKWl4UEFuUqKTtbi4jTTeuEjeGV8s="))
			})
		}
	})
})
