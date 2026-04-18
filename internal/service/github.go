package service

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Modificator/readlater-wip/internal/model"
)

var hrefRe = regexp.MustCompile(`https?://[^\s"'<>]+`)

func DetectGithubLinks(input string, fromContent bool) []model.DetectedLink {
	matches := hrefRe.FindAllString(input, -1)
	seen := map[string]struct{}{}
	out := make([]model.DetectedLink, 0, len(matches))
	for _, raw := range matches {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		if _, ok := seen[u.String()]; ok {
			continue
		}
		seen[u.String()] = struct{}{}
		out = append(out, model.DetectedLink{
			URL:           u.String(),
			Type:          detectLinkType(u),
			IsFromContent: fromContent,
			CreatedAt:     time.Now().UTC(),
		})
	}
	return out
}

func detectLinkType(u *url.URL) model.LinkType {
	if strings.EqualFold(u.Host, "github.com") {
		if IsGithubRepoURL(u.String()) {
			return model.LinkTypeGithubRepo
		}
		return model.LinkTypeGithubPage
	}
	return model.LinkTypeExternal
}

func IsGithubRepoURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if !strings.EqualFold(u.Host, "github.com") {
		return false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return false
	}
	if parts[0] == "" || parts[1] == "" {
		return false
	}
	if len(parts) == 2 {
		return true
	}
	switch parts[2] {
	case "issues", "pull", "releases", "blob", "tree", "commit", "actions", "wiki", "discussions":
		return false
	default:
		return true
	}
}
