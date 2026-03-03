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
	Branch        string   `json:"branch"`
	Assignees     []string `json:"assignees"`
	MyReviewState string   `json:"my_review_state"`
	CIState       string   `json:"ci_state"`
	Approvals     int      `json:"approvals"`
	ReviewCount   int      `json:"review_count"`
}

// Key は重複排除用の一意キー
func (pr PullRequest) Key() string {
	return fmt.Sprintf("%s#%d", pr.Repo, pr.Number)
}

// Client は GitHub API クライアント
type Client struct {
	gql *api.GraphQLClient
}

// NewClient は go-gh の認証情報を使って Client を生成
func NewClient() (*Client, error) {
	gql, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("GraphQL クライアントの初期化に失敗 (gh auth login は実行済みですか?): %w", err)
	}
	return &Client{gql: gql}, nil
}

// GetAuthenticatedUser は認証ユーザーのログイン名を返す
func (c *Client) GetAuthenticatedUser() (string, error) {
	var q struct {
		Viewer struct {
			Login string
		}
	}
	if err := c.gql.Query("Viewer", &q, nil); err != nil {
		return "", fmt.Errorf("ユーザー情報の取得に失敗: %w", err)
	}
	return q.Viewer.Login, nil
}

// ── Search: レビュー依頼された PR ──────────────────────────────────

const searchQuery = `
query ReviewRequested($query: String!, $cursor: String) {
  search(query: $query, type: ISSUE, first: 50, after: $cursor) {
    pageInfo {
      hasNextPage
      endCursor
    }
    nodes {
      ... on PullRequest {
        number
        title
        url
        createdAt
        updatedAt
        isDraft
        headRefName
        author { login }
        assignees(first: 10) {
          nodes { login }
        }
        repository {
          nameWithOwner
        }
        labels(first: 10) {
          nodes { name }
        }
        commits(last: 1) {
          nodes {
            commit {
              statusCheckRollup {
                state
              }
            }
          }
        }
        reviews(first: 100) {
          nodes {
            author { login }
            state
          }
        }
      }
    }
  }
}
`

type searchResponse struct {
	Search struct {
		PageInfo struct {
			HasNextPage bool
			EndCursor   string
		}
		Nodes []prNode
	}
}

type prNode struct {
	Number     int
	Title      string
	URL        string
	CreatedAt  string
	UpdatedAt  string
	IsDraft     bool
	HeadRefName string
	Author      struct{ Login string }
	Assignees struct {
		Nodes []struct{ Login string }
	}
	Repository struct {
		NameWithOwner string
	}
	Labels struct {
		Nodes []struct{ Name string }
	}
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					State string
				}
			}
		}
	}
	Reviews struct {
		Nodes []struct {
			Author struct{ Login string }
			State  string
		}
	}
}

// SearchReviewRequested は自分にレビュー依頼が来ているオープン PR を
// レビュー状態付きで返す (1クエリで PR + レビューを取得)
func (c *Client) SearchReviewRequested(username string) ([]PullRequest, error) {
	q := fmt.Sprintf("is:pr is:open involves:%s -author:%s", username, username)

	var allPRs []PullRequest
	var cursor *string

	for {
		vars := map[string]interface{}{
			"query": q,
		}
		if cursor != nil {
			vars["cursor"] = *cursor
		}

		var resp searchResponse
		if err := c.gql.Do(searchQuery, vars, &resp); err != nil {
			return nil, fmt.Errorf("レビュー依頼 PR の検索に失敗: %w", err)
		}

		for _, node := range resp.Search.Nodes {
			allPRs = append(allPRs, nodeToPR(node, username))
		}

		if !resp.Search.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Search.PageInfo.EndCursor
	}

	return allPRs, nil
}

// ── Search: 自分の PR ────────────────────────────────────────────────

// SearchMyPRs は自分が author のオープン PR をレビュー状況付きで返す
func (c *Client) SearchMyPRs(username string) ([]PullRequest, error) {
	q := fmt.Sprintf("is:pr is:open author:%s", username)

	var allPRs []PullRequest
	var cursor *string

	for {
		vars := map[string]interface{}{
			"query": q,
		}
		if cursor != nil {
			vars["cursor"] = *cursor
		}

		var resp searchResponse
		if err := c.gql.Do(searchQuery, vars, &resp); err != nil {
			return nil, fmt.Errorf("自分の PR の検索に失敗: %w", err)
		}

		for _, node := range resp.Search.Nodes {
			allPRs = append(allPRs, nodeToMyPR(node))
		}

		if !resp.Search.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Search.PageInfo.EndCursor
	}

	return allPRs, nil
}

// ── 共通変換 ────────────────────────────────────────────────────────

func nodeAssignees(node prNode) []string {
	assignees := make([]string, 0, len(node.Assignees.Nodes))
	for _, a := range node.Assignees.Nodes {
		assignees = append(assignees, a.Login)
	}
	return assignees
}

func nodeToPR(node prNode, username string) PullRequest {
	labels := make([]string, 0, len(node.Labels.Nodes))
	for _, l := range node.Labels.Nodes {
		labels = append(labels, l.Name)
	}

	// 自分の最新レビュー状態を探す & 承認数カウント
	var myState string
	latestByUser := make(map[string]string)
	for _, r := range node.Reviews.Nodes {
		if equalFold(r.Author.Login, username) {
			myState = r.State
		}
		if r.Author.Login != "" && !equalFold(r.Author.Login, node.Author.Login) {
			latestByUser[r.Author.Login] = r.State
		}
	}
	approvals := 0
	for _, state := range latestByUser {
		if state == "APPROVED" {
			approvals++
		}
	}

	var ciState string
	if len(node.Commits.Nodes) > 0 {
		if rollup := node.Commits.Nodes[0].Commit.StatusCheckRollup; rollup != nil {
			ciState = rollup.State
		}
	}

	return PullRequest{
		Repo:          node.Repository.NameWithOwner,
		Number:        node.Number,
		Title:         node.Title,
		Author:        node.Author.Login,
		URL:           node.URL,
		CreatedAt:     node.CreatedAt,
		UpdatedAt:     node.UpdatedAt,
		Draft:         node.IsDraft,
		Branch:        node.HeadRefName,
		Labels:        labels,
		Assignees:     nodeAssignees(node),
		MyReviewState: myState,
		CIState:       ciState,
		Approvals:     approvals,
	}
}

// nodeToMyPR は自分の PR ノードを PullRequest に変換する
// 全レビュアーのレビュー状態を集約して Approvals / ReviewCount / MyReviewState を設定
func nodeToMyPR(node prNode) PullRequest {
	labels := make([]string, 0, len(node.Labels.Nodes))
	for _, l := range node.Labels.Nodes {
		labels = append(labels, l.Name)
	}

	var ciState string
	if len(node.Commits.Nodes) > 0 {
		if rollup := node.Commits.Nodes[0].Commit.StatusCheckRollup; rollup != nil {
			ciState = rollup.State
		}
	}

	// ユーザーごとに最新レビューのみ採用
	latestByUser := make(map[string]string)
	for _, r := range node.Reviews.Nodes {
		login := r.Author.Login
		if login == "" {
			continue
		}
		// author 自身のレビューは除外
		if equalFold(login, node.Author.Login) {
			continue
		}
		latestByUser[login] = r.State
	}

	approvals := 0
	hasChangesRequested := false
	for _, state := range latestByUser {
		if state == "APPROVED" {
			approvals++
		}
		if state == "CHANGES_REQUESTED" {
			hasChangesRequested = true
		}
	}

	reviewCount := len(latestByUser)

	// 集約ステータス: CHANGES_REQUESTED > 未レビュー > APPROVED
	var aggregatedState string
	switch {
	case hasChangesRequested:
		aggregatedState = "CHANGES_REQUESTED"
	case reviewCount == 0:
		aggregatedState = "" // 未レビュー
	case approvals == reviewCount:
		aggregatedState = "APPROVED"
	default:
		aggregatedState = "" // レビュー途中
	}

	return PullRequest{
		Repo:          node.Repository.NameWithOwner,
		Number:        node.Number,
		Title:         node.Title,
		Author:        node.Author.Login,
		URL:           node.URL,
		CreatedAt:     node.CreatedAt,
		UpdatedAt:     node.UpdatedAt,
		Draft:         node.IsDraft,
		Branch:        node.HeadRefName,
		Labels:        labels,
		Assignees:     nodeAssignees(node),
		MyReviewState: aggregatedState,
		CIState:       ciState,
		Approvals:     approvals,
		ReviewCount:   reviewCount,
	}
}

// ── ヘルパー ────────────────────────────────────────────────────────

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

// SectionsToJSON は mine/reviews セクションを JSON 文字列に変換
func SectionsToJSON(mine, reviews []PullRequest) (string, error) {
	if mine == nil {
		mine = []PullRequest{}
	}
	if reviews == nil {
		reviews = []PullRequest{}
	}
	out := struct {
		Mine    []PullRequest `json:"mine"`
		Reviews []PullRequest `json:"reviews"`
	}{
		Mine:    mine,
		Reviews: reviews,
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
