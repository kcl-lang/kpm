package e2e

import (
	"fmt"
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Kpm CLI Testing", func() {
	ginkgo.Context("testing no args", func() {
		testSuites := LoadAllTestSuites(filepath.Join(filepath.Join(GetWorkDir(), TEST_SUITES_DIR), "no_args"))
		for _, ts := range testSuites {
			ginkgo.It(ts.GetTestSuiteInfo(), func() {
				workspace := GetWorkspace()
				stdout, stderr, err := ExecKpmWithWorkDir(ts.Input, workspace)

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				fmt.Printf("stderr: %v\n", stderr)
				fmt.Printf("stdout: %v\n", stdout)
				gomega.Expect(stdout).To(gomega.MatchRegexp(ts.ExpectStdout))
				gomega.Expect(stderr).To(gomega.MatchRegexp(ts.ExpectStderr))
			})
		}
	})
})
