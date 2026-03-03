package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

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
	Repo        string `json:"repo"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	URL         string `json:"url"`
	StatusEmoji string `json:"status_emoji"`
	CIEmoji     string `json:"ci_emoji"`
	Approvals   string `json:"approvals,omitempty"`
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
			StatusEmoji: display.ReviewStateEmoji(pr.MyReviewState),
			CIEmoji:     display.CIEmoji(pr.CIState),
		}
	}
	return rows, nil
}

func fetchMyPRs(client *github.Client, username string, includeDraft bool) ([]prRow, error) {
	prs, err := client.SearchMyPRs(username)
	if err != nil {
		return nil, err
	}

	prs = filterDraft(prs, includeDraft)
	display.SortMyPRs(prs)

	rows := make([]prRow, len(prs))
	for i, pr := range prs {
		rows[i] = prRow{
			Repo:        pr.Repo,
			Number:      pr.Number,
			Title:       pr.Title,
			Author:      pr.Author,
			URL:         pr.URL,
			StatusEmoji: display.MyPRStateEmoji(pr.MyReviewState),
			CIEmoji:     display.CIEmoji(pr.CIState),
			Approvals:   fmt.Sprintf("%d/%d", pr.Approvals, pr.ReviewCount),
		}
	}
	return rows, nil
}

// Start starts the HTTP server.
func Start(client *github.Client, username string, includeDraft bool, port int, showMine, showReviews bool) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := templateData{ShowMine: showMine, ShowReviews: showReviews}
		if showMine {
			rows, err := fetchMyPRs(client, username, includeDraft)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			data.MinePRs = rows
		}
		if showReviews {
			rows, err := fetchReviewPRs(client, username, includeDraft)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			data.ReviewPRs = rows
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, data)
	})

	http.HandleFunc("/api/prs", func(w http.ResponseWriter, r *http.Request) {
		result := struct {
			Mine    []prRow `json:"mine,omitempty"`
			Reviews []prRow `json:"reviews,omitempty"`
		}{}
		if showMine {
			rows, err := fetchMyPRs(client, username, includeDraft)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			result.Mine = rows
		}
		if showReviews {
			rows, err := fetchReviewPRs(client, username, includeDraft)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			result.Reviews = rows
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Listening on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, nil)
}
