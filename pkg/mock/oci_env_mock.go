package mock

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to determine repo root")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")), nil
}

func repoScriptCommand(pathParts ...string) (*exec.Cmd, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(filepath.Join(append([]string{root}, pathParts...)...))
	cmd.Dir = root
	return cmd, nil
}

// StartDockerRegistry starts a local Docker registry by executing a shell script.
func StartDockerRegistry() error {
	cmd, err := repoScriptCommand("scripts", "reg.sh")
	if err != nil {
		return err
	}
	return cmd.Run()
}

// PushTestPkgToRegistry pushes the test package to the local Docker registry.
func PushTestPkgToRegistry() error {
	cmd, err := repoScriptCommand("pkg", "mock", "test_script", "push_pkg.sh")
	if err != nil {
		return err
	}
	return cmd.Run()
}

// CleanTestEnv cleans up the test environment by executing a cleanup script.
func CleanTestEnv() error {
	cmd, err := repoScriptCommand("pkg", "mock", "test_script", "cleanup_test_environment.sh")
	if err != nil {
		return err
	}
	return cmd.Run()
}
