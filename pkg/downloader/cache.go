package downloader

import (
	"fmt"
	"os"
	"path/filepath"

	"kcl-lang.io/kpm/pkg/utils"
)

// Cache is the interface for the cache.
type Cache interface {
	// Update updates the cache with the source.
	// Due to the different action for the different source, the updateFunc is used to update the cache.
	Update(source Source, updateFunc func(cachePath string) error) error
	// Find finds the cache path for the source.
	Find(source Source) (string, error)
	// Remove removes the source from the cache.
	Remove(source Source) error
	// RemoveAll removes all the cache.
	RemoveAll() error
}

// PkgCache is used for some common cache operations and members which are same for Git and Oci.
type PkgCache struct {
	// The cache directory.
	cacheDir string
}

// Update updates the cache with the source.
func (p *PkgCache) Update(source Source, updateFunc func(cachePath string) error) error {
	cachePath, err := source.GenCachePath()
	if err != nil {
		return err
	}

	if err := updateFunc(filepath.Join(p.cacheDir, cachePath)); err != nil {
		return err
	}

	return nil
}

var PkgCacheNotFound = fmt.Errorf("package cache not found")

// Find finds the cache path for the source.
func (p *PkgCache) Find(source Source) (string, error) {
	srcPath, err := source.GenSrcCachePath()
	if err != nil {
		return "", err
	}

	srcFullPath := filepath.Join(p.cacheDir, srcPath)

	if utils.DirExists(srcFullPath) {
		return srcFullPath, nil
	} else {
		return "", PkgCacheNotFound
	}
}

// Remove removes the source from the cache.
func (p *PkgCache) Remove(source Source) error {
	cachePath, err := source.GenCachePath()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(p.cacheDir, cachePath)); err != nil {
		return err
	}

	srcCachePath, err := source.GenSrcCachePath()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(p.cacheDir, srcCachePath)); err != nil {
		return err
	}

	return nil
}

// RemoveAll removes all the cache.
func (p *PkgCache) RemoveAll() error {
	if err := os.RemoveAll(p.cacheDir); err != nil {
		return err
	}
	return nil
}
