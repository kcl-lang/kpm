// This file mainly provides some functions that can be used to adapt for git downloading by go-getter.
package git

import (
	"fmt"
	"net/url"
	"strings"

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

// SplitSubdir parses a git URL that may carry a sub-directory selector
// following the go-getter convention `...//subdir`. It returns the bare
// repository URL (without the `//subdir` suffix) and the sub-directory.
//
// Examples:
//
//	SplitSubdir("https://github.com/org/repo.git//pkg/sub")
//	  -> "https://github.com/org/repo.git", "pkg/sub", nil
//	SplitSubdir("git::https://github.com/org/repo.git?ref=v1//pkg/sub")
//	  -> "git::https://github.com/org/repo.git?ref=v1", "pkg/sub", nil
//	SplitSubdir("https://github.com/org/repo.git")
//	  -> "https://github.com/org/repo.git", "", nil
//	SplitSubdir("https://github.com/org/repo.git//")
//	  -> "https://github.com/org/repo.git//", "", nil  (no content after //)
//
// The split is intentionally conservative: only the LAST `//` that is
// not part of a scheme delimiter (i.e. not preceded by ":") and that
// is followed by a non-empty path component is treated as a
// sub-directory separator.
func SplitSubdir(repoURL string) (string, string, error) {
	if repoURL == "" {
		return "", "", nil
	}

	// Schemes like "git::" can confuse a naive split, so split off the
	// leading scheme if present and re-attach after parsing.
	schemePrefix := ""
	body := repoURL
	if idx := strings.Index(repoURL, "::"); idx >= 0 && idx < 16 {
		candidate := repoURL[:idx]
		if isLikelyScheme(candidate) {
			schemePrefix = repoURL[:idx+2]
			body = repoURL[idx+2:]
		}
	}

	// Find the LAST "//" occurrence. Earlier occurrences (e.g. "https://")
	// must be skipped.
	idx := strings.LastIndex(body, "//")
	if idx < 0 {
		return repoURL, "", nil
	}

	// We only treat this as a sub-dir separator if it is NOT part of a
	// scheme delimiter (i.e. not preceded by ":").
	if idx > 0 && body[idx-1] == ':' {
		return repoURL, "", nil
	}

	rawSub := body[idx+2:]
	if rawSub == "" {
		// Trailing "//" with nothing after — there is no sub-directory to
		// extract. Return the URL unchanged so the caller sees the
		// original intent (e.g. preserves the trailing "//").
		return repoURL, "", nil
	}

	base := strings.TrimRight(body[:idx], "/")
	sub := strings.TrimLeft(rawSub, "/")

	// The "sub" portion may itself contain a query string or fragment
	// because users sometimes write "...//subdir?ref=v1". Split it off
	// and append to base so go-getter sees a clean URL.
	if qIdx := strings.Index(sub, "?"); qIdx >= 0 {
		base = base + sub[qIdx:]
		sub = sub[:qIdx]
	}
	if hIdx := strings.Index(sub, "#"); hIdx >= 0 {
		base = base + sub[hIdx:]
		sub = sub[:hIdx]
	}

	sub = strings.Trim(sub, "/")
	if sub == "" {
		// Subdir portion was purely query/fragment — treat as no subdir.
		return repoURL, "", nil
	}
	return schemePrefix + base, sub, nil
}

// isLikelyScheme reports whether `s` looks like a go-getter scheme name
// (letters, digits, underscores).
func isLikelyScheme(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
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
