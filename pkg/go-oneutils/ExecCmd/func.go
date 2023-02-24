package ExecCmd

import (
	"bytes"
	"errors"
	"os/exec"
)

// Run  运行命令
func Run(dir string, name string, args ...string) error {
	cmd := exec.Command(
		name, args...)
	cmd.Dir = dir
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

// RunWithStdout 运行命令并取得成功返回值
func RunWithStdout(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(
		name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Start()
	if err != nil {
		return "", err
	}
	err = cmd.Wait()
	if err != nil {
		return "", err
	}
	outStr, errStr := stdout.String(), stderr.String()
	if err != nil {
		return "", err
	}
	if errStr != "" {
		return "", errors.New(errStr)
	}
	return outStr, nil
}
