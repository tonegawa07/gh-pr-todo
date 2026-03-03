package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"

	"github.com/tonegawa07/gh-pr-todo/internal/display"
	"github.com/tonegawa07/gh-pr-todo/internal/github"
)

//go:embed index.html
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "index.html"))

type templateData struct {
	ShowMine    bool
	ShowReviews bool
	MinePRs     []prRow
	ReviewPRs   []prRow
}

type prRow struct {
	Repo        string   `json:"repo"`
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	Author      string   `json:"author"`
	URL         string   `json:"url"`
	Branch      string   `json:"branch,omitempty"`
	Assignees   []string `json:"assignees"`
	StatusEmoji string   `json:"status_emoji"`
	CIEmoji     string   `json:"ci_emoji"`
	Approvals   string   `json:"approvals,omitempty"`
}

func filterDraft(prs []github.PullRequest, includeDraft bool) []github.PullRequest {
	if includeDraft {
		return prs
	}
	var filtered []github.PullRequest
	for _, pr := range prs {
		if !pr.Draft {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}

func fetchReviewPRs(client *github.Client, username string, includeDraft bool) ([]prRow, error) {
	prs, err := client.SearchReviewRequested(username)
	if err != nil {
		return nil, err
	}

	prs = filterDraft(prs, includeDraft)
	display.SortPRs(prs)

	rows := make([]prRow, len(prs))
	for i, pr := range prs {
		rows[i] = prRow{
			Repo:        pr.Repo,
			Number:      pr.Number,
			Title:       pr.Title,
			Author:      pr.Author,
			URL:         pr.URL,
			Assignees:   pr.Assignees,
			StatusEmoji: display.ReviewStateEmoji(pr.MyReviewState),
			CIEmoji:     display.CIEmoji(pr.CIState),
		}
	}
	return rows, nil
}

func fetchMyPRs(client *github.Client, username string) ([]prRow, error) {
	prs, err := client.SearchMyPRs(username)
	if err != nil {
		return nil, err
	}

	display.SortMyPRs(prs)

	rows := make([]prRow, len(prs))
	for i, pr := range prs {
		rows[i] = prRow{
			Repo:        pr.Repo,
			Number:      pr.Number,
			Title:       pr.Title,
			Author:      pr.Author,
			URL:         pr.URL,
			Branch:      pr.Branch,
			Assignees:   pr.Assignees,
			StatusEmoji: display.MyPRStateEmoji(pr.MyReviewState),
			CIEmoji:     display.CIEmoji(pr.CIState),
			Approvals:   fmt.Sprintf("%d/%d", pr.Approvals, pr.ReviewCount),
		}
	}
	return rows, nil
}

// Start starts the HTTP server.
func Start(client *github.Client, username string, includeDraft bool, port int, showMine, showReviews bool) error {
	fetchBoth := func() (mine, reviews []prRow, err error) {
		var wg sync.WaitGroup
		var mineErr, reviewsErr error

		if showMine {
			wg.Add(1)
			go func() {
				defer wg.Done()
				mine, mineErr = fetchMyPRs(client, username)
			}()
		}
		if showReviews {
			wg.Add(1)
			go func() {
				defer wg.Done()
				reviews, reviewsErr = fetchReviewPRs(client, username, includeDraft)
			}()
		}
		wg.Wait()

		if mineErr != nil {
			return nil, nil, mineErr
		}
		if reviewsErr != nil {
			return nil, nil, reviewsErr
		}
		return mine, reviews, nil
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		mine, reviews, err := fetchBoth()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, templateData{
			ShowMine: showMine, ShowReviews: showReviews,
			MinePRs: mine, ReviewPRs: reviews,
		})
	})

	http.HandleFunc("/api/prs", func(w http.ResponseWriter, r *http.Request) {
		mine, reviews, err := fetchBoth()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Mine    []prRow `json:"mine,omitempty"`
			Reviews []prRow `json:"reviews,omitempty"`
		}{Mine: mine, Reviews: reviews})
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Listening on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, nil)
}
