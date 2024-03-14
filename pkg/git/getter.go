// This file mainly provides some functions that can be used to adapt for git downloading by go-getter.
package git

import (
	"fmt"

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
