package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Modificator/readlater-wip/internal/model"
	"github.com/Modificator/readlater-wip/internal/store"
)

type GiteaService struct {
	repo   *store.MemoryRepository
	client *http.Client
}

func NewGiteaService(repo *store.MemoryRepository) *GiteaService {
	return &GiteaService{
		repo:   repo,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *GiteaService) CreateTarget(target *model.GiteaTarget) error {
	if target.Alias == "" || target.BaseURL == "" || target.Owner == "" {
		return errors.New("alias, base_url and owner are required")
	}
	now := time.Now().UTC()
	target.CreatedAt, target.UpdatedAt = now, now
	target.Enabled = true
	s.repo.SaveGiteaTarget(target)
	return nil
}

func (s *GiteaService) ListTargets() []*model.GiteaTarget {
	return s.repo.ListGiteaTargets()
}

func (s *GiteaService) MirrorRepo(targetAlias, sourceRepoURL, repoNameOverride string) model.MirrorResult {
	target, err := s.repo.GetGiteaTargetByAlias(targetAlias)
	if err != nil {
		return model.MirrorResult{Status: "failed", ErrorMessage: "gitea target alias not found"}
	}
	if !target.Enabled {
		return model.MirrorResult{Status: "failed", ErrorMessage: "gitea target disabled"}
	}

	repoName := repoNameFromURL(sourceRepoURL)
	if repoNameOverride != "" {
		repoName = repoNameOverride
	}
	if repoName == "" {
		return model.MirrorResult{Status: "failed", ErrorMessage: "cannot parse repository name"}
	}

	payload := map[string]any{
		"clone_addr": sourceRepoURL,
		"repo_name":  repoName,
		"uid":        0,
		"mirror":     true,
		"private":    strings.EqualFold(target.DefaultVisibility, "private"),
	}
	body, _ := json.Marshal(payload)

	apiURL := strings.TrimRight(target.BaseURL, "/") + fmt.Sprintf("/api/v1/repos/migrate?owner=%s", target.Owner)
	req, _ := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if target.AccessToken != "" {
		req.Header.Set("Authorization", "token "+target.AccessToken)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return model.MirrorResult{Status: "failed", ErrorMessage: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return model.MirrorResult{Status: "failed", ErrorMessage: "gitea migrate api failed"}
	}

	return model.MirrorResult{
		Status:       "success",
		GiteaRepoURL: strings.TrimRight(target.BaseURL, "/") + "/" + target.Owner + "/" + repoName,
	}
}

func repoNameFromURL(raw string) string {
	parts := strings.Split(strings.Trim(raw, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".git")
	if name == "" {
		return ""
	}
	return name
}
