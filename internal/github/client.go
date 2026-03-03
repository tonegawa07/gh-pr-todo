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
	CIState       string   `json:"ci_state"`
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
        author { login }
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
	IsDraft    bool
	Author     struct{ Login string }
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

// ── 共通変換 ────────────────────────────────────────────────────────

func nodeToPR(node prNode, username string) PullRequest {
	labels := make([]string, 0, len(node.Labels.Nodes))
	for _, l := range node.Labels.Nodes {
		labels = append(labels, l.Name)
	}

	// 自分の最新レビュー状態を探す
	var myState string
	for _, r := range node.Reviews.Nodes {
		if equalFold(r.Author.Login, username) {
			myState = r.State
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
		Labels:        labels,
		MyReviewState: myState,
		CIState:       ciState,
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
