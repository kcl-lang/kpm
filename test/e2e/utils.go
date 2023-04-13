package e2e

import (
	"io/ioutil"
	"path/filepath"

	"kusionstack.io/kpm/pkg/reporter"
)

// LoadFirstFileWithExt read the first file with extention 'ext' in 'dir' and return the content.
func LoadFirstFileWithExt(dir string, ext string) string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		reporter.ExitWithReport("kpm_e2e: failed to load file, the dir not exists.")
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ext {
			data, err := ioutil.ReadFile(filepath.Join(dir, file.Name()))
			if err != nil {
				reporter.ExitWithReport("kpm_e2e: the file exists, but failed to read file.")
			}
			return string(data)
		}
	}

	reporter.ExitWithReport("kpm_e2e: failed to load file, the file not exists.")
	return ""
}
