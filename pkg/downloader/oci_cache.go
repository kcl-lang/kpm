package downloader

import (
	"fmt"
	"os"
	"path/filepath"
)

type OciCache struct {
	*PkgCache
}

func NewOciCacheWithCachePath(path string) *OciCache {
	return &OciCache{
		PkgCache: &PkgCache{cacheDir: path},
	}
}

func (o *OciCache) cachePath(s *Oci) (string, error) {
	var packageFilename string
	if s.Tag == "" {
		packageFilename = filepath.Base(s.Repo)
	} else {
		packageFilename = fmt.Sprintf("%s_%s", filepath.Base(s.Repo), s.Tag)
	}
	hash, err := s.Hash()
	if err != nil {
		return "", err
	}
	return filepath.Join(o.cacheDir, hash, packageFilename), nil
}

func (o *OciCache) Update(source Source, updateFunc func(cachePath string) error) error {
	if source.Oci == nil {
		return fmt.Errorf("oci source is nil")
	}

	cachePath, err := o.cachePath(source.Oci)
	if err != nil {
		return err
	}

	if err := updateFunc(cachePath); err != nil {
		return err
	}

	return nil
}

var PkgCacheNotFound = fmt.Errorf("package cache not found")

func (o *OciCache) Find(source Source) (string, error) {
	if source.Oci == nil {
		return "", fmt.Errorf("oci source is nil")
	}

	cachePath, err := o.cachePath(source.Oci)
	if err != nil {
		return "", err
	}

	matches, _ := filepath.Glob(filepath.Join(cachePath, "*.tar"))
	if matches == nil || len(matches) != 1 {
		// then try to glob tgz file
		matches, _ = filepath.Glob(filepath.Join(cachePath, "*.tgz"))
		if matches == nil || len(matches) != 1 {
			return "", fmt.Errorf("failed to find the downloaded kcl package tar file in '%s': %w", cachePath, PkgCacheNotFound)
		}
	}

	return matches[0], nil
}

func (o *OciCache) Remove(source Source) error {
	cachePath, err := o.cachePath(source.Oci)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(cachePath); err != nil {
		return err
	}

	return nil
}

func (o *OciCache) RemoveAll() error {
	return o.PkgCache.RemoveAll()
}
