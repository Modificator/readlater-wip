package model

import "time"

type TaskStatus string

const (
	StatusCreated             TaskStatus = "CREATED"
	StatusFetching            TaskStatus = "FETCHING"
	StatusMatchingRule        TaskStatus = "MATCHING_RULE"
	StatusParsing             TaskStatus = "PARSING"
	StatusDownloadingAssets   TaskStatus = "DOWNLOADING_ASSETS"
	StatusScanningGithubLinks TaskStatus = "SCANNING_GITHUB_LINKS"
	StatusMirroringToGitea    TaskStatus = "MIRRORING_TO_GITEA"
	StatusSaving              TaskStatus = "SAVING"
	StatusSuccess             TaskStatus = "SUCCESS"
	StatusPartialSuccess      TaskStatus = "PARTIAL_SUCCESS"
	StatusFailed              TaskStatus = "FAILED"
	StatusRetryPending        TaskStatus = "RETRY_PENDING"
)

type MatchType string

const (
	MatchTypePrefix MatchType = "prefix"
	MatchTypeRegex  MatchType = "regex"
)

type LinkType string

const (
	LinkTypeGithubRepo LinkType = "github_repo"
	LinkTypeGithubPage LinkType = "github_page"
	LinkTypeExternal   LinkType = "external"
)

type ArchiveTask struct {
	ID                 string     `json:"id"`
	URL                string     `json:"url"`
	Status             TaskStatus `json:"status"`
	RetryCount         int        `json:"retry_count"`
	SaveWebContent     bool       `json:"save_web_content"`
	DownloadAssets     bool       `json:"download_assets"`
	ExtractGithubLinks bool       `json:"extract_github_links"`
	CloneToGitea       bool       `json:"clone_to_gitea"`
	GiteaTargetAlias   string     `json:"gitea_target_alias,omitempty"`
	RuleID             string     `json:"rule_id,omitempty"`
	ForceBrowserRender bool       `json:"force_browser_render"`
	OverwriteExisting  bool       `json:"overwrite_existing"`
	RepoNameOverride   string     `json:"repo_name_override,omitempty"`
	ErrorMessage       string     `json:"error_message,omitempty"`
	MatchedRuleID      string     `json:"matched_rule_id,omitempty"`
	ArchiveRecordID    string     `json:"archive_record_id,omitempty"`
	MirrorResult       *MirrorResult
	DetectedLinks      []DetectedLink
	AssetStats         AssetStats
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type AssetStats struct {
	Downloaded int `json:"downloaded"`
	Failed     int `json:"failed"`
}

type ArchiveRecord struct {
	ID           string    `json:"id"`
	TaskID       string    `json:"task_id"`
	Title        string    `json:"title"`
	SourceURL    string    `json:"source_url"`
	Domain       string    `json:"domain"`
	ContentPath  string    `json:"content_path"`
	MetadataPath string    `json:"metadata_path"`
	RawHTMLPath  string    `json:"raw_html_path"`
	AssetsDir    string    `json:"assets_dir"`
	RuleID       string    `json:"rule_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ExtractionRule struct {
	ID                          string    `json:"id"`
	Name                        string    `json:"name"`
	MatchType                   MatchType `json:"match_type"`
	MatchExpression             string    `json:"match_expression"`
	Priority                    int       `json:"priority"`
	TitleSelector               string    `json:"title_selector,omitempty"`
	AuthorSelector              string    `json:"author_selector,omitempty"`
	PublishTimeSelector         string    `json:"publish_time_selector,omitempty"`
	ContentSelector             string    `json:"content_selector,omitempty"`
	RemoveSelectors             []string  `json:"remove_selectors,omitempty"`
	LinkExtractScope            string    `json:"link_extract_scope,omitempty"`
	AssetScopeSelector          string    `json:"asset_scope_selector,omitempty"`
	AssetURLAttributes          []string  `json:"asset_url_attributes,omitempty"`
	ExcludeAssetSelectors       []string  `json:"exclude_asset_selectors,omitempty"`
	DownloadAssets              bool      `json:"download_assets"`
	UseBrowserRender            bool      `json:"use_browser_render"`
	InheritPageHeadersForAssets bool      `json:"inherit_page_headers_for_assets"`
	Enabled                     bool      `json:"enabled"`
	CreatedAt                   time.Time `json:"created_at"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

type GiteaTarget struct {
	ID                string    `json:"id"`
	Alias             string    `json:"alias"`
	BaseURL           string    `json:"base_url"`
	AccessToken       string    `json:"access_token,omitempty"`
	Owner             string    `json:"owner"`
	RepoNamespace     string    `json:"repo_namespace"`
	DefaultVisibility string    `json:"default_visibility"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type DetectedLink struct {
	URL           string    `json:"url"`
	Type          LinkType  `json:"type"`
	IsFromContent bool      `json:"is_from_content"`
	CreatedAt     time.Time `json:"created_at"`
}

type MirrorResult struct {
	Status       string `json:"status"`
	GiteaRepoURL string `json:"gitea_repo_url,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}
