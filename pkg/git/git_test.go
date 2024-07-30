package git

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestWithGitOptions(t *testing.T) {
	cloneOpts := &CloneOptions{}
	WithBare(true)(cloneOpts)
	assert.Equal(t, cloneOpts.Bare, true)
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

	// Test non-bare repository cloning
	t.Run("NonBareClone", func(t *testing.T) {
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
	})

	// Test bare repository cloning
	t.Run("BareClone", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", "git_bare")
		assert.Equal(t, err, nil)
		defer func() {
			rErr := os.RemoveAll(tmpdir)
			assert.Equal(t, rErr, nil)
		}()

		_, err = CloneWithOpts(
			WithRepoURL("https://github.com/KusionStack/catalog.git"),
			WithCommit("4e59d5852cd7"),
			WithWriter(&buf),
			WithLocalPath(tmpdir),
			WithBare(true), // Set the Bare flag to true
		)
		assert.Equal(t, err, nil)

		// Verify the directory is a bare repository
		_, err = os.Stat(filepath.Join(tmpdir, "HEAD"))
		assert.Equal(t, os.IsNotExist(err), false)

		_, err = os.Stat(filepath.Join(tmpdir, "config"))
		assert.Equal(t, os.IsNotExist(err), false)

		_, err = os.Stat(filepath.Join(tmpdir, "objects"))
		assert.Equal(t, os.IsNotExist(err), false)

		_, err = os.Stat(filepath.Join(tmpdir, "refs"))
		assert.Equal(t, os.IsNotExist(err), false)
	})
}
