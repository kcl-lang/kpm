package downloader

import "os"

type Cache interface {
	Update(source Source, updateFunc func(cachePath string) error) error
	Find(source Source) (string, error)
	Remove(source Source) error
	RemoveAll() error
}

type PkgCache struct {
	cacheDir string
}

func (p *PkgCache) RemoveAll() error {
	if err := os.RemoveAll(p.cacheDir); err != nil {
		return err
	}
	return nil
}
