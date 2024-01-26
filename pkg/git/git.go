package git

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// CloneOptions is a struct for specifying options for cloning a git repository
type CloneOptions struct {
	RepoURL   string
	Commit    string
	Tag       string
	Branch    string
	LocalPath string
	Writer    io.Writer
}

// CloneOption is a function that modifies CloneOptions
type CloneOption func(*CloneOptions)

func NewCloneOptions(repoUrl, commit, tag, branch, localpath string, Writer io.Writer) *CloneOptions {
	return &CloneOptions{
		RepoURL:   repoUrl,
		Commit:    commit,
		Tag:       tag,
		Branch:    branch,
		LocalPath: localpath,
		Writer:    Writer,
	}
}

// WithRepoURL sets the repo URL for CloneOptions
func WithRepoURL(repoURL string) CloneOption {
	return func(o *CloneOptions) {
		o.RepoURL = repoURL
	}
}

// WithBranch sets the branch for CloneOptions
func WithBranch(branch string) CloneOption {
	return func(o *CloneOptions) {
		o.Branch = branch
	}
}

// WithCommit sets the commit for CloneOptions
func WithCommit(commit string) CloneOption {
	return func(o *CloneOptions) {
		o.Commit = commit
	}
}

// WithTag sets the tag for CloneOptions
func WithTag(tag string) CloneOption {
	return func(o *CloneOptions) {
		o.Tag = tag
	}
}

// WithLocalPath sets the local path for CloneOptions
func WithLocalPath(localPath string) CloneOption {
	return func(o *CloneOptions) {
		o.LocalPath = localPath
	}
}

// WithWriter sets the writer for CloneOptions
func WithWriter(writer io.Writer) CloneOption {
	return func(o *CloneOptions) {
		o.Writer = writer
	}
}

// Validate checks if the CloneOptions are valid
func (cloneOpts *CloneOptions) Validate() error {
	onlyOneAllowed := 0
	if cloneOpts.Branch != "" {
		onlyOneAllowed++
	}
	if cloneOpts.Tag != "" {
		onlyOneAllowed++
	}
	if cloneOpts.Commit != "" {
		onlyOneAllowed++
	}

	if onlyOneAllowed > 1 {
		return errors.New("only one of branch, tag or commit is allowed")
	}

	return nil
}

// Clone clones a git repository
func (cloneOpts *CloneOptions) Clone() (*git.Repository, error) {
	err := cloneOpts.Validate()
	if err != nil {
		return nil, err
	}

	gitCloneOpts := &git.CloneOptions{
		URL:      cloneOpts.RepoURL,
		Progress: nil,
	}

	if cloneOpts.Tag != "" {
		gitCloneOpts.ReferenceName = plumbing.ReferenceName(plumbing.NewTagReferenceName(cloneOpts.Tag))
	}

	if cloneOpts.Branch != "" {
		gitCloneOpts.ReferenceName = plumbing.ReferenceName(plumbing.NewBranchReferenceName(cloneOpts.Branch))
	}

	repo, err := git.PlainClone(cloneOpts.LocalPath, false, gitCloneOpts)
	if err != nil {
		return nil, err
	}
	// if the commit is specified, checkout the commit
	if cloneOpts.Commit != "" {
		// Checkout the specific commit
		w, err := repo.Worktree()
		if err != nil {
			return nil, err
		}

		// checkout the commit
		err = w.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(cloneOpts.Commit),
		})

		// if the commit is not found, try to find the full commit hash
		if err != nil {
			// get all the references
			commits, err := repo.CommitObjects()
			if err != nil {
				return nil, err
			}

			checkoutFinished := errors.New("checkout by commit finished")
			// iterate over the references, find the full commit hash and checkout
			err = commits.ForEach(func(commit *object.Commit) error {
				// if the commit hash starts with the commit, checkout it
				if strings.HasPrefix(commit.Hash.String(), cloneOpts.Commit) {
					err = w.Checkout(&git.CheckoutOptions{
						Hash: commit.Hash,
					})
					if err != nil {
						return err
					}
					return checkoutFinished
				}
				return nil
			})

			if err != nil && err == checkoutFinished {
				return repo, nil
			}

			if err != checkoutFinished {
				if err != nil {
					return nil, err
				} else {
					return nil, fmt.Errorf("commit '%s' not found", cloneOpts.Commit)
				}
			}
			return repo, nil
		}
	}

	return repo, nil
}

// CloneWithOpts will clone from `repoURL` to `localPath` via git by using CloneOptions
func CloneWithOpts(opts ...CloneOption) (*git.Repository, error) {
	cloneOpts := &CloneOptions{}
	for _, opt := range opts {
		opt(cloneOpts)
	}

	err := cloneOpts.Validate()
	if err != nil {
		return nil, err
	}

	return cloneOpts.Clone()
}

// Clone will clone from `repoURL` to `localPath` via git by tag name.
// Deprecated: This function will be removed in a future version. Use CloneWithOpts instead.
func Clone(repoURL string, tagName string, localPath string, writer io.Writer) (*git.Repository, error) {
	repo, err := git.PlainClone(localPath, false, &git.CloneOptions{
		URL:           repoURL,
		Progress:      writer,
		ReferenceName: plumbing.ReferenceName(plumbing.NewTagReferenceName(tagName)),
	})
	return repo, err
}
