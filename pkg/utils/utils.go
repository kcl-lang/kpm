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
