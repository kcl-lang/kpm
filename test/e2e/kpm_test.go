package e2e

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
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

				stdout, stderr, err := ExecKpmWithWorkDir(ts.Input, filepath.Join(workspace, ts.Name))

				expectedStdout := ReplaceAllKeyByValue(ts.ExpectStdout, "<workspace>", workspace)
				expectedStderr := ReplaceAllKeyByValue(ts.ExpectStderr, "<workspace>", workspace)

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(stdout).To(gomega.Equal(expectedStdout))
				gomega.Expect(stderr).To(gomega.Equal(expectedStderr))
			})
		}
	})
})
