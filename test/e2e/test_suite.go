package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kcl-lang.io/kpm/pkg/reporter"
)

const TEST_SUITES_DIR = "test_suites"

const STDOUT_EXT = ".stdout"
const STDERR_EXT = ".stderr"
const INPUT_EXT = ".input"
const JSON_EXT = ".json"
const ENV_EXT = ".env"

type TestSuite struct {
	Name         string
	Envs         string
	Input        string
	ExpectStdout string
	ExpectStderr string
}

// / checkTestSuite Check that the file corresponding to each suffix can appear only once
func CheckTestSuite(testSuitePath string, name string) {
	files, err := os.ReadDir(testSuitePath)
	if err != nil {
		reporter.ExitWithReport("kpm_e2e: failed to check test suite.")
	}

	extFlags := map[string]*bool{
		STDOUT_EXT: new(bool),
		STDERR_EXT: new(bool),
		INPUT_EXT:  new(bool),
		ENV_EXT:    new(bool),
	}

	for _, file := range files {
		if !file.IsDir() {
			ext := filepath.Ext(file.Name())
			if flag, ok := extFlags[ext]; ok {
				if *flag {
					reporter.ExitWithReport("kpm_e2e: invalid test suite, duplicate '*" + ext + "' file.")
				}
				*flag = true
			} else {
				reporter.ExitWithReport("kpm_e2e: invalid test suite, unknown file :", file.Name())
			}
		}
	}

	if !*extFlags[INPUT_EXT] {
		reporter.Report("kpm_e2e: ignore test ", name)
	}
}

// LoadTestSuite load test suite from 'getWorkDir()/test_suites/name'.
func LoadTestSuite(testSuitePath, name string) TestSuite {
	reporter.Report("kpm_e2e: loading '", name, "' from ", testSuitePath)
	CheckTestSuite(testSuitePath, name)
	return TestSuite{
		Name:             name,
		ExpectStdout: LoadFirstFileWithExt(testSuitePath, STDOUT_EXT),
		ExpectStderr: LoadFirstFileWithExt(testSuitePath, STDERR_EXT),

		// Strip whitespace to ignore the leading and trailing new lines.
		Input: strings.TrimSpace(LoadFirstFileWithExt(testSuitePath, INPUT_EXT)),
		Envs:  LoadFirstFileWithExt(testSuitePath, ENV_EXT),
	}
}

// LoadAllTestSuites load all test suites from 'getWorkDir()/test_suites'.
func LoadAllTestSuites(testSuitesDir string) []TestSuite {
	testSuites := make([]TestSuite, 0)
	files, err := os.ReadDir(testSuitesDir)

	if err != nil {
		reporter.ExitWithReport("kpm_e2e: failed to read test suites dir.")
	}

	for _, file := range files {
		if file.IsDir() {
			testSuites = append(
				testSuites,
				LoadTestSuite(
					filepath.Join(testSuitesDir, file.Name()),
					file.Name(),
				),
			)
		}
	}

	return testSuites
}

// GetTestSuiteInfo return a info for a test suite "<name>:<info>:<env>"
func (ts *TestSuite) GetTestSuiteInfo() string {
	return fmt.Sprintf("%s:%s", ts.Name, strings.ReplaceAll(ts.Envs, "\n", ":"))
}
