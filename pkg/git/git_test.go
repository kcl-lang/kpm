package git

import (
	"bytes"
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestWithGitOptions(t *testing.T) {
	cloneOpts := &CloneOptions{}
	WithRepoURL("test_url")(cloneOpts)
	assert.Equal(t, cloneOpts.RepoURL, "test_url")
	WithBranch("test_branch")(cloneOpts)
	assert.Equal(t, cloneOpts.Branch, "test_branch")
	WithCommit("test_commit")(cloneOpts)
	assert.Equal(t, cloneOpts.Commit, "test_commit")
	WithTag("test_tag")(cloneOpts)
	assert.Equal(t, cloneOpts.Tag, "test_tag")
	WithLocalPath("test_local_path")(cloneOpts)
	assert.Equal(t, cloneOpts.LocalPath, "test_local_path")
	WithWriter(nil)(cloneOpts)
	assert.Equal(t, cloneOpts.Writer, nil)
}

func TestNewCloneOptions(t *testing.T) {
	cloneOpts := NewCloneOptions("https://github.com/kcl-lang/kcl", "", "v1.0.0", "", "", nil)
	assert.Equal(t, cloneOpts.RepoURL, "https://github.com/kcl-lang/kcl")
	assert.Equal(t, cloneOpts.Tag, "v1.0.0")
	assert.Equal(t, cloneOpts.Commit, "")
	assert.Equal(t, cloneOpts.Branch, "")
	assert.Equal(t, cloneOpts.LocalPath, "")
	assert.Equal(t, cloneOpts.Writer, nil)
}

func TestValidateGitOptions(t *testing.T) {
	cloneOpts := &CloneOptions{}
	WithBranch("test_branch")(cloneOpts)
	err := cloneOpts.Validate()
	assert.Equal(t, err, nil)
	WithCommit("test_commit")(cloneOpts)
	err = cloneOpts.Validate()
	assert.Equal(t, err.Error(), "only one of branch, tag or commit is allowed")
}

func TestCloneWithOptions(t *testing.T) {
	var buf bytes.Buffer

	tmpdir, err := os.MkdirTemp("", "git")
	assert.Equal(t, err, nil)
	defer func() {
		rErr := os.RemoveAll(tmpdir)
		assert.Equal(t, rErr, nil)
	}()

	repo, err := CloneWithOpts(
		WithRepoURL("https://github.com/KusionStack/catalog.git"),
		WithCommit("4e59d5852cd7"),
		WithWriter(&buf),
		WithLocalPath(tmpdir),
	)
	assert.Equal(t, err, nil)

	head, err := repo.Head()
	assert.Equal(t, err, nil)
	assert.Equal(t, head.Hash().String(), "4e59d5852cd76542f9f0ec65e5773ca9f4e02462")
	assert.Equal(t, err, nil)
}
