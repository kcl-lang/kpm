package git

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"

	"gotest.tools/v3/assert"
)

// TestCheckoutFromBare tests the CheckoutFromBare method with branch, tag, and commit scenarios.
func TestCheckoutFromBare(t *testing.T) {
	setupTestRepo := func(t *testing.T) (*git.Repository, string) {
		// Create a temporary directory for the repository
		dir := t.TempDir()
		repo, err := git.PlainInit(dir, true) // Create a bare repository
		if err != nil {
			t.Fatal(err)
		}

		// Create a worktree to make commits
		worktree, err := repo.Worktree()
		if err != nil {
			t.Fatal(err)
		}

		// Commit the initial changes
		_, err = worktree.Commit("Initial commit", &git.CommitOptions{})
		if err != nil {
			t.Fatal(err)
		}

		// Create a branch and a tag
		err = repo.CreateBranch(&config.Branch{Name: "test-branch", Remote: "origin"})
		if err != nil {
			t.Fatal(err)
		}

		_, err = repo.CreateTag("v1.0.0", plumbing.NewHash("initial-commit-hash"), nil)
		if err != nil {
			t.Fatal(err)
		}

		return repo, dir
	}

	repo, dir := setupTestRepo(t)

	// Test branch checkout
	cloneOptsBranch := &CloneOptions{
		LocalPath: dir,
		Branch:    "test-branch",
		Bare:      true,
	}
	err := cloneOptsBranch.CheckoutFromBare()
	assert.NilError(t, err)
	headRef, err := repo.Head()
	assert.NilError(t, err)
	assert.Equal(t, "refs/heads/test-branch", headRef.Name().String())

	// Test tag checkout
	cloneOptsTag := &CloneOptions{
		LocalPath: dir,
		Tag:       "v1.0.0",
		Bare:      true,
	}
	err = cloneOptsTag.CheckoutFromBare()
	assert.NilError(t, err)
	headRef, err = repo.Head()
	assert.NilError(t, err)
	assert.Equal(t, "refs/tags/v1.0.0", headRef.Name().String())

	// Test commit checkout
	headRef, err = repo.Head()
	assert.NilError(t, err)
	commitHash := headRef.Hash().String()
	cloneOptsCommit := &CloneOptions{
		LocalPath: dir,
		Commit:    commitHash,
		Bare:      true,
	}
	err = cloneOptsCommit.CheckoutFromBare()
	assert.NilError(t, err)
	headRef, err = repo.Head()
	assert.NilError(t, err)
	assert.Equal(t, commitHash, headRef.Hash().String())

	// Test default checkout (fallback to master branch)
	cloneOptsDefault := &CloneOptions{
		LocalPath: dir,
		Bare:      true,
	}
	err = cloneOptsDefault.CheckoutFromBare()
	assert.NilError(t, err)
	headRef, err = repo.Head()
	assert.NilError(t, err)
	assert.Equal(t, plumbing.Master.String(), headRef.Name().String())
}

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

	tmpdir, err := os.MkdirTemp("", "git")
	tmpdir = filepath.Join(tmpdir, "git")
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
