package git

import (
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
)

/// Clone will clone from `repoURL` to `localPath` via git.
func Clone(repoURL string, localPath string) (*git.Repository, error) {
	repo, err := git.PlainClone(localPath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})
	return repo, err
}

// ParseRepoNameFromGitUrl get the repo name from git url,
// the repo name in 'https://github.com/xxx/kcl1.git' is 'kcl1'.
func ParseRepoNameFromGitUrl(gitUrl string) string {
	name := filepath.Base(gitUrl)
	return name[:len(name)-len(filepath.Ext(name))]
}
