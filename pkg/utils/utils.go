package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"

	"kusionstack.io/kpm/pkg/reporter"
)

// HashDir computes the checksum of a directory by concatenating all files and
// hashing them by sha256.
func HashDir(dir string) string {
	hasher := sha256.New()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(hasher, f); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		reporter.ExitWithReport("kpm: internal bug, failed to hash directory")
	}

	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}

// StoreToFile will store 'data' into toml file under 'filePath'.
func StoreToFile(filePath string, dataStr string) error {
	file, err := os.Create(filePath)
	if err != nil {
		reporter.ExitWithReport("kpm: failed to create file: ", filePath, err)
		return err
	}
	defer file.Close()

	file, err = os.Create(filePath)

	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.WriteString(file, dataStr); err != nil {
		return err
	}
	return nil
}

// ParseRepoNameFromGitUrl get the repo name from git url,
// the repo name in 'https://github.com/xxx/kcl1.git' is 'kcl1'.
func ParseRepoNameFromGitUrl(gitUrl string) string {
	name := filepath.Base(gitUrl)
	return name[:len(name)-len(filepath.Ext(name))]
}

// CreateFileIfNotExist will check whether the file under a certain path 'filePath/fileName' exists,
// and return an error if it exists, and call the method 'storeFunc' to save the file if it does not exist.
func CreateFileIfNotExist(filePath string, storeFunc func() error) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		reporter.Report("kpm: creating new :", filePath)
		err := storeFunc()
		if err != nil {
			reporter.Report("kpm: failed to create: ,", filePath)
			return err
		}
	} else {
		reporter.Report("kpm: '" + filePath + "' already exists")
		return err
	}
	return nil
}

// Whether the file exists
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
