package e2e

import (
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Kpm CLI Testing", func() {
	ginkgo.Context("testing kclvm_cli", func() {
		testSuites := LoadAllTestSuites(filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "kclvm_cli"))
		for _, ts := range testSuites {
			ts := ts
			ginkgo.It(ts.GetTestSuiteInfo(), func() {
				stdout, stderr, err := Exec(ts.Input)

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(stdout).To(gomega.MatchRegexp(ts.ExpectStdout))
				gomega.Expect(stderr).To(gomega.MatchRegexp(ts.ExpectStderr))
			})
		}
	})

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
				gomega.Expect(stdout).To(gomega.MatchRegexp(ts.ExpectStdout))
				gomega.Expect(stderr).To(gomega.MatchRegexp(ts.ExpectStderr))
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
})
