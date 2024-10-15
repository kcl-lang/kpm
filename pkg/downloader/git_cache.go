package downloader

type GitCache struct {
	*PkgCache
}

func (g *GitCache) Update(source Source, updateFunc func(cachePath string) error) error {
	// TODO: implement this method
	return nil
}

func (g *GitCache) Find(source Source) (string, error) {
	// TODO: implement this method
	return "", nil
}

func (g *GitCache) Remove(source Source) error {
	return nil
}

func (g *GitCache) RemoveAll() error {
	return g.PkgCache.RemoveAll()
}
