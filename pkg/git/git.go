package git

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

/// Clone will clone from `repoURL` to `localPath` via git.
func Clone(repoURL string, tagName string, localPath string) (*git.Repository, error) {
	repo, err := git.PlainClone(localPath, false, &git.CloneOptions{
		URL:           repoURL,
		Progress:      os.Stdout,
		ReferenceName: plumbing.ReferenceName(CreateTagRef(tagName)),
	})
	return repo, err
}

const TAG_PREFIX = "refs/tags/%s"

func CreateTagRef(tagName string) string {
	return fmt.Sprintf(TAG_PREFIX, tagName)
}
