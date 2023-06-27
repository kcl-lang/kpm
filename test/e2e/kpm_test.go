package e2e

import (
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Kpm CLI Testing", func() {
	ginkgo.Context("testing no args", func() {
		testSuites := LoadAllTestSuites(filepath.Join(filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "kpm"), "no_args"))
		for _, ts := range testSuites {
			// In the for loop, the variable ts is defined outside the scope of the ginkgo.It function.
			// This means that when the ginkgo.It function is executed,
			// it will always use the value of ts from the last iteration of the for loop.
			// To fix this issue, create a new variable inside the loop with the same value as ts,
			// and use that variable inside the ginkgo.It function.
			ts := ts
			ginkgo.It(ts.GetTestSuiteInfo(), func() {
				workspace := GetWorkspace()
				stdout, stderr, err := ExecKpmWithWorkDir(ts.Input, workspace)

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				// Using regular expressions may miss some cases where the string is empty.
				//
				// Since the login/logout-related test cases will output time information,
				// they cannot be compared with method 'Equal',
				// so 'ContainSubstring' is used to compare the results.
				gomega.Expect(stdout).To(gomega.ContainSubstring(ts.ExpectStdout))
				gomega.Expect(stderr).To(gomega.ContainSubstring(ts.ExpectStderr))
			})
		}
	})

	ginkgo.Context("testing 'kpm run --input <entry_file> <tar_path>'", func() {
		testSuitesRoot := filepath.Join(filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "kpm"), "run_tar")
		testSuites := LoadAllTestSuites(testSuitesRoot)
		testDataRoot := filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "test_data")

		for _, ts := range testSuites {
			ts := ts
			ginkgo.It(ts.GetTestSuiteInfo(), func() {
				workspace := GetWorkspace()

				Copy(filepath.Join(testDataRoot, "kcl1.tar"), filepath.Join(workspace, "kcl1.tar"))
				Copy(filepath.Join(testDataRoot, "exist_but_not_tar"), filepath.Join(workspace, "exist_but_not_tar"))

				stdout, stderr, err := ExecKpmWithWorkDir(ts.Input, workspace)

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(stdout).To(gomega.Equal(ts.ExpectStdout))
				gomega.Expect(stderr).To(gomega.Equal(ts.ExpectStderr))
			})
		}
	})

	ginkgo.Context("testing 'kpm run --input <entry_file>'", func() {
		testSuitesRoot := filepath.Join(filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "kpm"), "run_pkg_not_tar")
		testSuites := LoadAllTestSuites(testSuitesRoot)
		testDataRoot := filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "test_data")
		for _, ts := range testSuites {
			ts := ts
			ginkgo.It(ts.GetTestSuiteInfo(), func() {
				workspace := GetWorkspace()

				CopyDir(filepath.Join(testDataRoot, "test_kcl"), filepath.Join(workspace, "test_kcl"))

				stdout, stderr, err := ExecKpmWithWorkDir(ts.Input, filepath.Join(workspace, "test_kcl"))

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(stdout).To(gomega.Equal(ts.ExpectStdout))
				gomega.Expect(stderr).To(gomega.Equal(ts.ExpectStderr))
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
							gomega.Expect(stdout).To(gomega.Equal(expectedStdout))
						}
						if !IsIgnore(expectedStderr) {
							gomega.Expect(stderr).To(gomega.Equal(expectedStderr))
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

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(stdout).To(gomega.Equal(ts.ExpectStdout))
				gomega.Expect(stderr).To(gomega.Equal(ts.ExpectStderr))
			})
		}
	})
})
