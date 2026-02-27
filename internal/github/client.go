package github

import (
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
)

// PullRequest はフィルタ対象の PR 情報
type PullRequest struct {
	Repo          string   `json:"repo"`
	Number        int      `json:"number"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	URL           string   `json:"url"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
	Draft         bool     `json:"draft"`
	Labels        []string `json:"labels"`
	MyReviewState string   `json:"my_review_state"`
}

// Key は重複排除用の一意キー
func (pr PullRequest) Key() string {
	return fmt.Sprintf("%s#%d", pr.Repo, pr.Number)
}

// Client は GitHub API クライアント
type Client struct {
	rest *api.RESTClient
}

// NewClient は go-gh の認証情報を使って Client を生成
func NewClient() (*Client, error) {
	rest, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("GitHub API クライアントの初期化に失敗 (gh auth login は実行済みですか?): %w", err)
	}
	return &Client{rest: rest}, nil
}

// GetAuthenticatedUser は認証ユーザーのログイン名を返す
func (c *Client) GetAuthenticatedUser() (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := c.rest.Get("user", &user); err != nil {
		return "", fmt.Errorf("ユーザー情報の取得に失敗: %w", err)
	}
	return user.Login, nil
}

// SearchReviewRequested は自分にレビュー依頼が来ているオープン PR を返す
func (c *Client) SearchReviewRequested(username string) ([]PullRequest, error) {
	query := fmt.Sprintf("is:pr is:open review-requested:%s", username)

	var result searchResult
	if err := c.rest.Get(fmt.Sprintf("search/issues?q=%s&per_page=100&sort=updated", urlEncode(query)), &result); err != nil {
		return nil, fmt.Errorf("レビュー依頼 PR の検索に失敗: %w", err)
	}

	prs := make([]PullRequest, 0, len(result.Items))
	for _, item := range result.Items {
		labels := make([]string, 0, len(item.Labels))
		for _, l := range item.Labels {
			labels = append(labels, l.Name)
		}
		prs = append(prs, PullRequest{
			Repo:      extractRepo(item.RepositoryURL),
			Number:    item.Number,
			Title:     item.Title,
			Author:    item.User.Login,
			URL:       item.HTMLURL,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			Labels:    labels,
		})
	}
	return prs, nil
}

// GetOpenPRs は特定リポジトリのオープン PR を返す
func (c *Client) GetOpenPRs(repo string) ([]PullRequest, error) {
	var all []prResponse

	page := 1
	for {
		var page_ []prResponse
		path := fmt.Sprintf("repos/%s/pulls?state=open&per_page=100&page=%d", repo, page)
		if err := c.rest.Get(path, &page_); err != nil {
			return nil, fmt.Errorf("%s の PR 取得に失敗: %w", repo, err)
		}
		if len(page_) == 0 {
			break
		}
		all = append(all, page_...)
		if len(page_) < 100 {
			break
		}
		page++
	}

	prs := make([]PullRequest, 0, len(all))
	for _, pr := range all {
		labels := make([]string, 0, len(pr.Labels))
		for _, l := range pr.Labels {
			labels = append(labels, l.Name)
		}
		prs = append(prs, PullRequest{
			Repo:      repo,
			Number:    pr.Number,
			Title:     pr.Title,
			Author:    pr.User.Login,
			URL:       pr.HTMLURL,
			CreatedAt: pr.CreatedAt,
			UpdatedAt: pr.UpdatedAt,
			Draft:     pr.Draft,
			Labels:    labels,
		})
	}
	return prs, nil
}

// GetMyReviewState は PR に対する自分の最新レビュー状態を返す
func (c *Client) GetMyReviewState(repo string, prNumber int, username string) (string, error) {
	var reviews []reviewResponse
	path := fmt.Sprintf("repos/%s/pulls/%d/reviews", repo, prNumber)
	if err := c.rest.Get(path, &reviews); err != nil {
		return "", fmt.Errorf("レビュー状態の取得に失敗 (%s#%d): %w", repo, prNumber, err)
	}

	var latest string
	for _, r := range reviews {
		if equalFold(r.User.Login, username) {
			latest = r.State
		}
	}
	return latest, nil
}

// ── 内部型 ──────────────────────────────────────────────────────────

type searchResult struct {
	Items []searchItem `json:"items"`
}

type searchItem struct {
	Number        int         `json:"number"`
	Title         string      `json:"title"`
	User          userInfo    `json:"user"`
	HTMLURL       string      `json:"html_url"`
	RepositoryURL string      `json:"repository_url"`
	CreatedAt     string      `json:"created_at"`
	UpdatedAt     string      `json:"updated_at"`
	Labels        []labelInfo `json:"labels"`
}

type prResponse struct {
	Number    int         `json:"number"`
	Title     string      `json:"title"`
	User      userInfo    `json:"user"`
	HTMLURL   string      `json:"html_url"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
	Draft     bool        `json:"draft"`
	Labels    []labelInfo `json:"labels"`
}

type reviewResponse struct {
	User  userInfo `json:"user"`
	State string   `json:"state"`
}

type userInfo struct {
	Login string `json:"login"`
}

type labelInfo struct {
	Name string `json:"name"`
}

func extractRepo(repositoryURL string) string {
	const prefix = "https://api.github.com/repos/"
	if len(repositoryURL) > len(prefix) {
		return repositoryURL[len(prefix):]
	}
	return repositoryURL
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func urlEncode(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~':
			result = append(result, c)
		case c == ' ':
			result = append(result, '+')
		default:
			result = append(result, '%')
			result = append(result, "0123456789ABCDEF"[c>>4])
			result = append(result, "0123456789ABCDEF"[c&0x0F])
		}
	}
	return string(result)
}

// ToJSON は PR リストを JSON 文字列に変換
func ToJSON(prs []PullRequest) (string, error) {
	if prs == nil {
		prs = []PullRequest{}
	}
	data, err := json.MarshalIndent(prs, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
