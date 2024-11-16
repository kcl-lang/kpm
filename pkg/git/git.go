package git

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/hashicorp/go-getter"
	giturl "github.com/kubescape/go-git-url"
)

// CloneOptions is a struct for specifying options for cloning a git repository
type CloneOptions struct {
	RepoURL   string
	Commit    string
	Tag       string
	Branch    string
	LocalPath string
	Writer    io.Writer
	Bare      bool   // New field to indicate if the clone should be bare
	ProxyURL  string // New field for proxy URL
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

// WithBare sets the bare flag for CloneOptions
func WithBare(isBare bool) CloneOption {
	return func(o *CloneOptions) {
		o.Bare = isBare
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

// WithProxyURL sets the proxy URL for CloneOptions
func WithProxyURL(proxyURL string) CloneOption {
	return func(o *CloneOptions) {
		o.ProxyURL = proxyURL
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

// CheckoutFromBare checks out the specified reference from a bare repository
func (cloneOpts *CloneOptions) CheckoutFromBare() error {
	if !cloneOpts.Bare {
		return errors.New("repository is not bare")
	}

	var reference string

	if cloneOpts.Branch != "" {
		reference = "refs/heads/" + cloneOpts.Branch
	} else if cloneOpts.Tag != "" {
		reference = "refs/tags/" + cloneOpts.Tag
	} else if cloneOpts.Commit != "" {
		reference = cloneOpts.Commit
	} else {
		return errors.New("no reference specified for checkout")
	}

	cmd := exec.Command("git", "-C", cloneOpts.LocalPath, "symbolic-ref", "HEAD", reference)
	if cloneOpts.Commit != "" {
		cmd = exec.Command("git", "-C", cloneOpts.LocalPath, "update-ref", "HEAD", reference)
	}
	cmd.Stdout = cloneOpts.Writer
	cmd.Stderr = cloneOpts.Writer

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update HEAD in bare repository: %w", err)
	}

	return nil
}

// Clone clones a git repository, handling both bare and non-bare options with proxy support
func (cloneOpts *CloneOptions) Clone() (*git.Repository, error) {
	if err := cloneOpts.Validate(); err != nil {
		return nil, err
	}

	if cloneOpts.Bare {
		// Use local git command to clone as bare repository
		cmdArgs := []string{"clone", "--bare", cloneOpts.RepoURL, cloneOpts.LocalPath}
		cmd := exec.Command("git", cmdArgs...)

		// Set proxy environment variables if ProxyURL is set
		if cloneOpts.ProxyURL != "" {
			env := os.Environ()
			env = append(env, "http_proxy="+cloneOpts.ProxyURL, "https_proxy="+cloneOpts.ProxyURL)
			cmd.Env = env
		}

		cmd.Stdout = cloneOpts.Writer
		cmd.Stderr = cloneOpts.Writer

		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository: %s, error: %w", string(output), err)
		}

		repo, err := git.PlainOpen(cloneOpts.LocalPath)
		if err != nil {
			return nil, err
		}

		// If a branch, tag, or commit is specified, check it out
		if cloneOpts.Branch != "" || cloneOpts.Tag != "" || cloneOpts.Commit != "" {
			err = cloneOpts.CheckoutFromBare()
			if err != nil {
				return nil, err
			}
		}

		return repo, nil
	}

	// Default non-bare clone using go-getter
	url, err := cloneOpts.ForceGitUrl()
	if err != nil {
		return nil, err
	}

	client := &getter.Client{
		Src:       url,
		Dst:       cloneOpts.LocalPath,
		Pwd:       cloneOpts.LocalPath,
		Mode:      getter.ClientModeDir,
		Detectors: goGetterNoDetectors,
		Getters:   goGetterGetters,
	}

	// Set HTTP client with proxy if ProxyURL is set
	if cloneOpts.ProxyURL != "" {
		if err := SetProxy(client, cloneOpts.ProxyURL); err != nil {
			return nil, err
		}
	}

	if err := client.Get(); err != nil {
		return nil, err
	}

	repo, err := git.PlainOpen(cloneOpts.LocalPath)
	if err != nil {
		return nil, err
	}

	return repo, nil

}

// CloneWithOpts clones from `repoURL` to `localPath` via git using CloneOptions
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

// Clone clones from `repoURL` to `localPath` via git by tag name.
// Deprecated: This function will be removed in a future version. Use CloneWithOpts instead.
func Clone(repoURL string, tagName string, localPath string, writer io.Writer) (*git.Repository, error) {
	repo, err := git.PlainClone(localPath, false, &git.CloneOptions{
		URL:           repoURL,
		Progress:      writer,
		ReferenceName: plumbing.ReferenceName(plumbing.NewTagReferenceName(tagName)),
	})
	return repo, err
}

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// parseNextPageURL extracts the 'next' page URL from the 'Link' header
func parseNextPageURL(linkHeader string) (string, error) {
	// Regex to extract 'next' page URL from the link header
	r := regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)
	matches := r.FindStringSubmatch(linkHeader)

	if len(matches) < 2 {
		return "", errors.New("next page URL not found")
	}
	return matches[1], nil
}

// GetAllGithubReleases fetches all releases from a GitHub repository
func GetAllGithubReleases(url string) ([]string, error) {
	// Initialize and parse the URL to extract owner and repo names
	gitURL, err := giturl.NewGitURL(url)
	if err != nil {
		return nil, err
	}

	if gitURL.GetHostName() != "github.com" {
		return nil, errors.New("only GitHub repositories are currently supported")
	}

	// Construct initial API URL for the first page
	apiBase := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", gitURL.GetOwnerName(), gitURL.GetRepoName())
	apiURL := fmt.Sprintf("%s?per_page=100&page=1", apiBase)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	var releaseTags []string

	for apiURL != "" {
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch tags, status code: %d", resp.StatusCode)
		}

		// Decode the JSON response into a slice of releases
		var releases []GitHubRelease
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return nil, err
		}

		// Extract tag names from the releases
		for _, release := range releases {
			releaseTags = append(releaseTags, release.TagName)
		}

		// Read the `Link` header to get the next page URL, if available
		linkHeader := resp.Header.Get("Link")
		if linkHeader != "" {
			nextURL, err := parseNextPageURL(linkHeader)
			if err != nil {
				apiURL = ""
			} else {
				apiURL = nextURL
			}
		} else {
			apiURL = ""
		}
	}

	return releaseTags, nil
}

// IsGitBareRepo checks if a directory is a bare git repository
func IsGitBareRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-bare-repository")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// Fetch fetches the latest changes from a remote repository
func Fetch(dir string, args ...string) error {
	cmdArgs := append([]string{"-C", dir, "fetch", "origin"}, args...)
	cmd := exec.Command("git", cmdArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to fetch latest changes: %v, output: %s", err, out.String())
	}
	return nil
}
