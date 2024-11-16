// This file mainly provides some functions that can be used to adapt for git downloading by go-getter.
package git

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-getter"
	"kcl-lang.io/kpm/pkg/constants"
)

var goGetterGetters = map[string]getter.Getter{
	"git": new(getter.GitGetter),
}

var goGetterNoDetectors = []getter.Detector{}

const GIT_PROTOCOL = "git::"

func ForceProtocol(url, protocol string) string {
	return protocol + url
}

// ForceGitUrl will add the branch, tag or commit to the git URL and force it to the git protocol
// `<URL>` will return `Git::<URL>?ref=<branch|tag|commit>`
func (cloneOpts *CloneOptions) ForceGitUrl() (string, error) {
	if err := cloneOpts.Validate(); err != nil {
		return "", nil
	}

	repoUrl, err := url.Parse(cloneOpts.RepoURL)
	if err != nil {
		return "", err
	}

	// If the Git URL is a file path, which is a local bare repo,
	// we need to force the protocol to "file://"
	if repoUrl.Scheme == "" {
		repoUrl.Scheme = "file"
	}

	cloneOpts.RepoURL = repoUrl.String()

	var attributes = []string{cloneOpts.Branch, cloneOpts.Commit, cloneOpts.Tag}
	for _, attr := range attributes {
		if attr != "" {
			return ForceProtocol(
				cloneOpts.RepoURL+fmt.Sprintf(constants.GIT_PROTOCOL_URL_PATTERN, attr),
				GIT_PROTOCOL,
			), nil
		}
	}

	return ForceProtocol(cloneOpts.RepoURL, GIT_PROTOCOL), nil
}

// SetProxy sets the HTTP client with proxy settings for go-getter
func SetProxy(client *getter.Client, proxyURL string) error {
	parsedProxyURL, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("invalid proxy URL: %w", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(parsedProxyURL),
	}
	httpClient := &http.Client{
		Transport: transport,
	}

	// Create a custom HttpGetter with the custom HTTP client
	customHttpGetter := &getter.HttpGetter{
		Client: httpClient,
	}

	// Assign the custom getter to the client for "http" and "https" protocols
	client.Getters["http"] = customHttpGetter
	client.Getters["https"] = customHttpGetter

	return nil
}
