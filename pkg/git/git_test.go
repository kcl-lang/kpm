package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
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
	cloneOpts := NewCloneOptions("https://github.com/kcl-lang/kcl", "", "v1.0.0", "", "", "", nil)
	assert.Equal(t, cloneOpts.RepoURL, "https://github.com/kcl-lang/kcl")
	assert.Equal(t, cloneOpts.Tag, "v1.0.0")
	assert.Equal(t, cloneOpts.Commit, "")
	assert.Equal(t, cloneOpts.SubPackage, "")
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

	// Test cloning a remote repo as a bare repo
	t.Run("RemoteBareClone", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", "git_bare")
		assert.Equal(t, err, nil)
		defer func() {
			rErr := os.RemoveAll(tmpdir)
			assert.Equal(t, rErr, nil)
		}()

		_, err = CloneWithOpts(
			WithRepoURL("https://github.com/KusionStack/catalog.git"),
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

	// Test cloning a remote repo as a normal repo and checking out a commit
	t.Run("RemoteNonBareCloneWithCommit", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", "git_non_bare")
		assert.Equal(t, err, nil)
		defer func() {
			rErr := os.RemoveAll(tmpdir)
			assert.Equal(t, rErr, nil)
		}()

		repo, err := CloneWithOpts(
			WithRepoURL("https://github.com/KusionStack/catalog.git"),
			WithCommit("4e59d5852cd76542f9f0ec65e5773ca9f4e02462"),
			WithWriter(&buf),
			WithLocalPath(tmpdir),
			WithBare(false), // Ensure the Bare flag is false
		)
		assert.Equal(t, err, nil)

		head, err := repo.Head()
		assert.Equal(t, err, nil)
		assert.Equal(t, head.Hash().String(), "4e59d5852cd76542f9f0ec65e5773ca9f4e02462")
	})

	// Test cloning a bare repo as a bare repo
	t.Run("LocalBareCloneAsBare", func(t *testing.T) {
		// Setup a local bare repository
		bareRepoPath, err := os.MkdirTemp("", "local_bare_repo")
		assert.Equal(t, err, nil)
		defer func() {
			rErr := os.RemoveAll(bareRepoPath)
			assert.Equal(t, rErr, nil)
		}()
		cmd := exec.Command("git", "clone", "--bare", "https://github.com/KusionStack/catalog.git", bareRepoPath)
		err = cmd.Run()
		assert.Equal(t, err, nil)

		// Clone the local bare repository as a bare repository
		tmpdir, err := os.MkdirTemp("", "clone_bare_repo")
		assert.Equal(t, err, nil)
		defer func() {
			rErr := os.RemoveAll(tmpdir)
			assert.Equal(t, rErr, nil)
		}()

		_, err = CloneWithOpts(
			WithRepoURL(bareRepoPath),
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

	// Test cloning a bare repo as a normal repo and checking out a commit
	t.Run("LocalBareCloneAsNonBareWithCommit", func(t *testing.T) {
		// Setup a local bare repository
		bareRepoPath, err := os.MkdirTemp("", "local_bare_repo")
		assert.Equal(t, err, nil)
		defer func() {
			rErr := os.RemoveAll(bareRepoPath)
			assert.Equal(t, rErr, nil)
		}()
		cmd := exec.Command("git", "clone", "--bare", "https://github.com/KusionStack/catalog.git", bareRepoPath)
		err = cmd.Run()
		assert.Equal(t, err, nil)

		// Construct the file URL for the local bare repository
		bareRepoURL := fmt.Sprintf("file://%s", bareRepoPath)

		// Clone the local bare repository as a normal repository and checkout a commit
		tmpdir, err := os.MkdirTemp("", "clone_non_bare_repo")
		assert.Equal(t, err, nil)
		defer func() {
			rErr := os.RemoveAll(tmpdir)
			assert.Equal(t, rErr, nil)
		}()

		repo, err := CloneWithOpts(
			WithRepoURL(bareRepoURL),
			WithCommit("4e59d5852cd76542f9f0ec65e5773ca9f4e02462"),
			WithWriter(&buf),
			WithLocalPath(tmpdir),
			WithBare(false), // Ensure the Bare flag is false
		)
		assert.Equal(t, err, nil)

		head, err := repo.Head()
		assert.Equal(t, err, nil)
		assert.Equal(t, head.Hash().String(), "4e59d5852cd76542f9f0ec65e5773ca9f4e02462")
	})
}

func TestCheckoutFromBare(t *testing.T) {
	var buf bytes.Buffer
	tmpdir, err := os.MkdirTemp("", "git-bare-checkout")
	assert.NilError(t, err)
	defer func() {
		rErr := os.RemoveAll(tmpdir)
		assert.NilError(t, rErr)
	}()

	// First, clone a bare repository
	repoURL := "https://github.com/KusionStack/catalog.git"
	commitSHA := "4e59d5852cd76542f9f0ec65e5773ca9f4e02462"

	repo, err := CloneWithOpts(
		WithRepoURL(repoURL),
		WithWriter(&buf),
		WithLocalPath(tmpdir),
		WithBare(true),
	)
	assert.NilError(t, err, "Failed to clone bare repository: %s", buf.String())

	// Verify that the repository is bare
	config, err := repo.Config()
	assert.NilError(t, err)
	assert.Equal(t, config.Core.IsBare, true, "Expected repository to be bare")

	// Now, attempt to update HEAD to a specific commit
	checkoutOpts := &CloneOptions{
		RepoURL:   repoURL,
		LocalPath: tmpdir,
		Commit:    commitSHA,
		Writer:    &buf,
		Bare:      true,
	}

	err = checkoutOpts.CheckoutFromBare()
	assert.NilError(t, err, "Failed to update HEAD in bare repository: %s", buf.String())

	// Verify that HEAD points to the specified commit
	head, err := repo.Head()
	assert.NilError(t, err)
	assert.Equal(t, head.Hash().String(), commitSHA, "Expected HEAD to point to the specified commit")

	// For a bare repository, we can't check for working directory files
	// Instead, we can verify that the commit exists in the repository
	_, err = repo.CommitObject(plumbing.NewHash(commitSHA))
	assert.NilError(t, err, "Expected commit to exist in the repository")
}
