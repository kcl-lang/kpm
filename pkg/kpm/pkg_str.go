package kpm

import (
	"encoding/json"
	"errors"
	"github.com/KusionStack/kpm/pkg/go-oneutils/Semver"
	"github.com/valyala/fasthttp"
	"strings"
	"time"
)

type PkgString = string

func GetRequirePkgStruct(ps PkgString) (*RequirePkgStruct, error) {
	b := ps
	index1, index2 := 0, 0
	index3, indexlast := 0, 0
	var ver string
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case ':':
			index1 = i
		case '/':
			indexlast = index3
			index3 = i

		case '@':
			index2 = i
			break
		}
	}
	if index1 == 0 {
		return nil, errors.New("parsing failed")
	}

	if !(b[:index1] == "git" || b[:index1] == "registry") {
		return nil, errors.New("parsing failed")
	}
	if index2 == 0 || strings.HasSuffix(b, "@latest") {
		index2 = len(ps)
		//获取最新版本
		//println(b[indexlast:index2], "555")
		//"https://gitee.com/api/v5/repos"
		var u string
		if strings.HasPrefix(b, "git:github.com") {
			u = "https://api.github.com/repos" + b[indexlast:index2] + "/releases/latest"
		} else if strings.HasPrefix(b, "git:gitee.com") {
			u = "https://gitee.com/api/v5/repos" + b[indexlast:index2] + "/releases/latest"
		} else {
			return nil, errors.New("unsupported repository type, failed to get latest version")
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)
		req.Header.SetMethod("GET")
		req.SetRequestURI(u)
		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)
		if err := fasthttp.Do(req, resp); err != nil {
			return nil, err
		}
		if strings.HasPrefix(b, "git:github.com") {
			ginfo := GithubInfo{}
			err := json.Unmarshal(resp.Body(), &ginfo)
			if err != nil {
				return nil, err
			}
			ver = ginfo.TagName
		} else {
			ginfo := GiteeInfo{}
			err := json.Unmarshal(resp.Body(), &ginfo)
			if err != nil {
				return nil, err
			}
			ver = ginfo.TagName
		}

	} else {
		ver = b[index2+1:]
	}

	//if pkgStruct.Version == "" {
	//	//https://api.github.com/repos/orangebees/go-oneutils/releases/latest
	//}
	//if index1 == 0 || index2 == 0 || index1 >= index2 {
	//	return nil, errors.New("parsing failed")
	//}

	return &RequirePkgStruct{
		Type:    b[:index1],
		Name:    b[index1+1 : index2],
		Version: Semver.VersionString(ver),
	}, nil

}

type GithubInfo struct {
	Url       string `json:"url"`
	AssetsUrl string `json:"assets_url"`
	UploadUrl string `json:"upload_url"`
	HtmlUrl   string `json:"html_url"`
	Id        int    `json:"id"`
	Author    struct {
		Login             string `json:"login"`
		Id                int    `json:"id"`
		NodeId            string `json:"node_id"`
		AvatarUrl         string `json:"avatar_url"`
		GravatarId        string `json:"gravatar_id"`
		Url               string `json:"url"`
		HtmlUrl           string `json:"html_url"`
		FollowersUrl      string `json:"followers_url"`
		FollowingUrl      string `json:"following_url"`
		GistsUrl          string `json:"gists_url"`
		StarredUrl        string `json:"starred_url"`
		SubscriptionsUrl  string `json:"subscriptions_url"`
		OrganizationsUrl  string `json:"organizations_url"`
		ReposUrl          string `json:"repos_url"`
		EventsUrl         string `json:"events_url"`
		ReceivedEventsUrl string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	NodeId          string        `json:"node_id"`
	TagName         string        `json:"tag_name"`
	TargetCommitish string        `json:"target_commitish"`
	Name            string        `json:"name"`
	Draft           bool          `json:"draft"`
	Prerelease      bool          `json:"prerelease"`
	CreatedAt       time.Time     `json:"created_at"`
	PublishedAt     time.Time     `json:"published_at"`
	Assets          []interface{} `json:"assets"`
	TarballUrl      string        `json:"tarball_url"`
	ZipballUrl      string        `json:"zipball_url"`
	Body            string        `json:"body"`
}

// GiteeInfo https://gitee.com/api/v5/repos/dotnetchina/SmartSQL/releases/latest
type GiteeInfo struct {
	Id              int    `json:"id"`
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
	Prerelease      bool   `json:"prerelease"`
	Name            string `json:"name"`
	Body            string `json:"body"`
	Author          struct {
		Id                int    `json:"id"`
		Login             string `json:"login"`
		Name              string `json:"name"`
		AvatarUrl         string `json:"avatar_url"`
		Url               string `json:"url"`
		HtmlUrl           string `json:"html_url"`
		Remark            string `json:"remark"`
		FollowersUrl      string `json:"followers_url"`
		FollowingUrl      string `json:"following_url"`
		GistsUrl          string `json:"gists_url"`
		StarredUrl        string `json:"starred_url"`
		SubscriptionsUrl  string `json:"subscriptions_url"`
		OrganizationsUrl  string `json:"organizations_url"`
		ReposUrl          string `json:"repos_url"`
		EventsUrl         string `json:"events_url"`
		ReceivedEventsUrl string `json:"received_events_url"`
		Type              string `json:"type"`
	} `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	Assets    []struct {
		BrowserDownloadUrl string `json:"browser_download_url"`
		Name               string `json:"name,omitempty"`
	} `json:"assets"`
}
