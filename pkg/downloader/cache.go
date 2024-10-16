package downloader

import "os"

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

// RemoveAll removes all the cache.
func (p *PkgCache) RemoveAll() error {
	if err := os.RemoveAll(p.cacheDir); err != nil {
		return err
	}
	return nil
}
