package service

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Modificator/readlater-wip/internal/model"
	"github.com/Modificator/readlater-wip/internal/store"
	"golang.org/x/net/html"
)

type ArchiveService struct {
	repo        *store.MemoryRepository
	ruleSvc     *RuleService
	giteaSvc    *GiteaService
	storageRoot string
	queue       chan string
	wg          sync.WaitGroup
	client      *http.Client
	workerStop  chan struct{}
}

type CreateTaskInput struct {
	URL                string `json:"url"`
	SaveWebContent     bool   `json:"save_web_content"`
	DownloadAssets     bool   `json:"download_assets"`
	ExtractGithubLinks bool   `json:"extract_github_links"`
	CloneToGitea       bool   `json:"clone_to_gitea"`
	GiteaTargetAlias   string `json:"gitea_target_alias"`
	RuleID             string `json:"rule_id"`
	ForceBrowserRender bool   `json:"force_browser_render"`
	OverwriteExisting  bool   `json:"overwrite_existing"`
	RepoNameOverride   string `json:"repo_name_override"`
}

func NewArchiveService(repo *store.MemoryRepository, ruleSvc *RuleService, giteaSvc *GiteaService, storageRoot string) *ArchiveService {
	return &ArchiveService{
		repo:        repo,
		ruleSvc:     ruleSvc,
		giteaSvc:    giteaSvc,
		storageRoot: storageRoot,
		queue:       make(chan string, 1024),
		workerStop:  make(chan struct{}),
		client:      &http.Client{Timeout: 25 * time.Second},
	}
}

func (s *ArchiveService) StartWorkers(n int) {
	for i := 0; i < n; i++ {
		s.wg.Add(1)
		go s.workerLoop()
	}
}

func (s *ArchiveService) StopWorkers() {
	close(s.workerStop)
	s.wg.Wait()
}

func (s *ArchiveService) CreateTask(in CreateTaskInput) (*model.ArchiveTask, error) {
	if err := validateURL(in.URL); err != nil {
		return nil, err
	}
	if in.CloneToGitea && in.GiteaTargetAlias == "" {
		return nil, errors.New("gitea_target_alias is required when clone_to_gitea is true")
	}

	if !in.OverwriteExisting {
		existing, err := s.repo.FindTaskByURL(in.URL)
		if err == nil {
			return existing, nil
		}
	}

	now := time.Now().UTC()
	task := &model.ArchiveTask{
		ID:                 newID("task"),
		URL:                in.URL,
		Status:             model.StatusCreated,
		SaveWebContent:     in.SaveWebContent,
		DownloadAssets:     in.DownloadAssets,
		ExtractGithubLinks: in.ExtractGithubLinks,
		CloneToGitea:       in.CloneToGitea,
		GiteaTargetAlias:   in.GiteaTargetAlias,
		RuleID:             in.RuleID,
		ForceBrowserRender: in.ForceBrowserRender,
		OverwriteExisting:  in.OverwriteExisting,
		RepoNameOverride:   in.RepoNameOverride,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	s.repo.SaveTask(task)
	s.queue <- task.ID
	return task, nil
}

func (s *ArchiveService) RetryTask(taskID string) (*model.ArchiveTask, error) {
	task, err := s.repo.GetTask(taskID)
	if err != nil {
		return nil, err
	}
	task.Status = model.StatusRetryPending
	task.RetryCount++
	task.ErrorMessage = ""
	task.UpdatedAt = time.Now().UTC()
	s.repo.SaveTask(task)
	s.queue <- task.ID
	return task, nil
}

func (s *ArchiveService) GetTask(taskID string) (*model.ArchiveTask, error) {
	return s.repo.GetTask(taskID)
}

func (s *ArchiveService) workerLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.workerStop:
			return
		case taskID := <-s.queue:
			s.processTask(taskID)
		}
	}
}

func (s *ArchiveService) processTask(taskID string) {
	task, err := s.repo.GetTask(taskID)
	if err != nil {
		return
	}

	setStatus := func(st model.TaskStatus) {
		task.Status = st
		task.UpdatedAt = time.Now().UTC()
		s.repo.SaveTask(task)
	}
	fail := func(msg string) {
		task.Status = model.StatusFailed
		task.ErrorMessage = msg
		task.UpdatedAt = time.Now().UTC()
		s.repo.SaveTask(task)
	}

	setStatus(model.StatusMatchingRule)
	rule, _ := s.ruleSvc.MatchRule(task.URL, task.RuleID)
	if rule != nil {
		task.MatchedRuleID = rule.ID
		s.repo.SaveTask(task)
	}

	setStatus(model.StatusFetching)
	rawHTML, fetchErr := s.fetchHTML(task.URL)
	if fetchErr != nil {
		fail(fetchErr.Error())
		return
	}

	setStatus(model.StatusParsing)
	title, cleanedHTML := extractTitleAndBody(rawHTML)
	if title == "" {
		title = task.URL
	}

	setStatus(model.StatusScanningGithubLinks)
	if task.ExtractGithubLinks {
		task.DetectedLinks = DetectGithubLinks(cleanedHTML, true)
		s.repo.SaveTask(task)
	}

	assetMap := map[string]string{}
	assetStats := model.AssetStats{}
	assetsDir := ""
	if task.DownloadAssets || (rule != nil && rule.DownloadAssets) {
		setStatus(model.StatusDownloadingAssets)
		assetsDir = s.assetsDir(task.ID)
		_ = os.MkdirAll(assetsDir, 0o755)
		assetMap, assetStats = s.downloadImages(cleanedHTML, task.URL, assetsDir)
		cleanedHTML = rewriteImageLinks(cleanedHTML, assetMap)
		task.AssetStats = assetStats
		s.repo.SaveTask(task)
	}

	setStatus(model.StatusSaving)
	record, saveErr := s.saveArtifacts(task, title, rawHTML, cleanedHTML, assetsDir)
	if saveErr != nil {
		fail(saveErr.Error())
		return
	}

	task.ArchiveRecordID = record.ID
	resultStatus := model.StatusSuccess

	if task.CloneToGitea && IsGithubRepoURL(task.URL) {
		setStatus(model.StatusMirroringToGitea)
		m := s.giteaSvc.MirrorRepo(task.GiteaTargetAlias, task.URL, task.RepoNameOverride)
		task.MirrorResult = &m
		if m.Status != "success" {
			resultStatus = model.StatusPartialSuccess
		}
	}

	if task.AssetStats.Failed > 0 && resultStatus == model.StatusSuccess {
		resultStatus = model.StatusPartialSuccess
	}

	task.Status = resultStatus
	task.ErrorMessage = ""
	task.UpdatedAt = time.Now().UTC()
	s.repo.SaveTask(task)
}

func (s *ArchiveService) fetchHTML(rawURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "readlater-wip/0.1")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("fetch failed with status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func extractTitleAndBody(raw string) (string, string) {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return "", raw
	}
	var title string
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = strings.TrimSpace(n.FirstChild.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	body := findElement(doc, "body")
	if body == nil {
		return title, raw
	}
	var b strings.Builder
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&b, c)
	}
	return title, b.String()
}

func (s *ArchiveService) saveArtifacts(task *model.ArchiveTask, title, rawHTML, cleanedHTML, assetsDir string) (*model.ArchiveRecord, error) {
	now := time.Now().UTC()
	baseDir := filepath.Join(s.storageRoot, now.Format("2006"), now.Format("01"), now.Format("02"), task.ID)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}

	contentPath := filepath.Join(baseDir, "content.md")
	metadataPath := filepath.Join(baseDir, "metadata.json")
	rawPath := filepath.Join(baseDir, "raw.html")

	markdown := htmlToMarkdown(cleanedHTML)
	if err := os.WriteFile(contentPath, []byte(markdown), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(rawPath, []byte(rawHTML), 0o644); err != nil {
		return nil, err
	}
	meta := map[string]any{
		"title":           title,
		"source_url":      task.URL,
		"task_id":         task.ID,
		"matched_rule_id": task.MatchedRuleID,
		"status":          task.Status,
		"created_at":      now,
		"asset_stats":     task.AssetStats,
		"detected_links":  task.DetectedLinks,
		"mirror_result":   task.MirrorResult,
	}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metadataPath, metaBytes, 0o644); err != nil {
		return nil, err
	}

	u, _ := url.Parse(task.URL)
	rec := &model.ArchiveRecord{
		ID:           newID("record"),
		TaskID:       task.ID,
		Title:        title,
		SourceURL:    task.URL,
		Domain:       u.Host,
		ContentPath:  contentPath,
		MetadataPath: metadataPath,
		RawHTMLPath:  rawPath,
		AssetsDir:    assetsDir,
		RuleID:       task.MatchedRuleID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.repo.SaveRecord(rec)
	return rec, nil
}

func (s *ArchiveService) assetsDir(taskID string) string {
	now := time.Now().UTC()
	return filepath.Join(s.storageRoot, now.Format("2006"), now.Format("01"), now.Format("02"), taskID, "assets")
}

var imgURLRe = regexp.MustCompile(`(?i)<img[^>]+(?:src|data-src|data-original|data-lazy-src)=["']([^"']+)["'][^>]*>`)

func (s *ArchiveService) downloadImages(contentHTML, pageURL, assetsDir string) (map[string]string, model.AssetStats) {
	matches := imgURLRe.FindAllStringSubmatch(contentHTML, -1)
	out := map[string]string{}
	seenResolved := map[string]struct{}{}
	stats := model.AssetStats{}

	base, _ := url.Parse(pageURL)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		orig := strings.TrimSpace(m[1])
		resolved := resolveURL(base, orig)
		if resolved == "" {
			stats.Failed++
			continue
		}
		if _, ok := seenResolved[resolved]; ok {
			out[orig] = out[resolved]
			continue
		}
		seenResolved[resolved] = struct{}{}

		fileName := normalizeAssetName(resolved)
		localPath := filepath.Join(assetsDir, fileName)
		if err := downloadWithLimit(s.client, resolved, localPath, 5*1024*1024); err != nil {
			stats.Failed++
			continue
		}
		relPath := "assets/" + fileName
		out[orig] = relPath
		out[resolved] = relPath
		stats.Downloaded++
	}
	return out, stats
}

func rewriteImageLinks(htmlContent string, assetMap map[string]string) string {
	out := htmlContent
	for old, newVal := range assetMap {
		out = strings.ReplaceAll(out, old, newVal)
	}
	return out
}

func htmlToMarkdown(input string) string {
	tok := html.NewTokenizer(strings.NewReader(input))
	var b strings.Builder
	for {
		tt := tok.Next()
		switch tt {
		case html.ErrorToken:
			return strings.TrimSpace(b.String()) + "\n"
		case html.TextToken:
			t := strings.TrimSpace(string(tok.Text()))
			if t != "" {
				b.WriteString(t)
				b.WriteString("\n\n")
			}
		case html.StartTagToken, html.EndTagToken:
			tn, _ := tok.TagName()
			s := string(tn)
			if s == "h1" || s == "h2" || s == "h3" || s == "p" || s == "li" || s == "pre" || s == "blockquote" {
				b.WriteString("\n")
			}
		}
	}
}

func normalizeAssetName(assetURL string) string {
	h := sha1.Sum([]byte(assetURL))
	name := hex.EncodeToString(h[:])
	ext := filepath.Ext(assetURL)
	if len(ext) > 8 || ext == "" {
		ext = ".bin"
	}
	return name + ext
}

func downloadWithLimit(client *http.Client, rawURL, localPath string, maxBytes int64) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "readlater-wip/0.1")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("download status %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, io.LimitReader(resp.Body, maxBytes))
	return err
}

func resolveURL(base *url.URL, ref string) string {
	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	if base == nil {
		return u.String()
	}
	return base.ResolveReference(u).String()
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("only http/https url is allowed")
	}
	if u.Hostname() == "" {
		return errors.New("invalid host")
	}
	if isPrivateHost(u.Hostname()) {
		return errors.New("private or loopback host is not allowed")
	}
	return nil
}

func isPrivateHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
	}
	return false
}

func findElement(n *html.Node, tag string) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if got := findElement(c, tag); got != nil {
			return got
		}
	}
	return nil
}

func newID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UTC().UnixNano())
}
