// This file mainly provides some functions that can be used to adapt for git downloading by go-getter.
package git

import (
	"fmt"

	"github.com/hashicorp/go-getter"
	"kcl-lang.io/kpm/pkg/constants"
)

const GIT_PROTOCOL = "git::"

func ForceProtocol(url, protocol string) string {
	return protocol + url
}

// ForceGitUrl will add the subpackage and branch, tag or commit to the git URL and force it to the git protocol
// `<URL>` will return `Git::<URL>//<SubPackage>?ref=<branch|tag|commit>`
func (cloneOpts *CloneOptions) ForceGitUrl() (string, error) {
	if err := cloneOpts.Validate(); err != nil {
		return "", nil
	}

	newRepoUrl := ""
	if cloneOpts.SubPackage != "" {
		newRepoUrl = cloneOpts.RepoURL + "//" + cloneOpts.SubPackage
	}

	var attributes = []string{cloneOpts.Branch, cloneOpts.Commit, cloneOpts.Tag}
	for _, attr := range attributes {
		if attr != "" {
			return ForceProtocol(
				newRepoUrl+fmt.Sprintf(constants.GIT_PROTOCOL_URL_PATTERN, attr),
				GIT_PROTOCOL,
			), nil
		}
	}

	return ForceProtocol(newRepoUrl, GIT_PROTOCOL), nil
}
