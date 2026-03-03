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
	Count int
	PRs   []prRow
}

type prRow struct {
	Repo        string `json:"repo"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	URL         string `json:"url"`
	StatusEmoji string `json:"status_emoji"`
	CIEmoji     string `json:"ci_emoji"`
}

func fetchPRs(client *github.Client, username string, includeDraft bool) ([]prRow, error) {
	prs, err := client.SearchReviewRequested(username)
	if err != nil {
		return nil, err
	}

	if !includeDraft {
		var filtered []github.PullRequest
		for _, pr := range prs {
			if !pr.Draft {
				filtered = append(filtered, pr)
			}
		}
		prs = filtered
	}

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

// Start starts the HTTP server.
func Start(client *github.Client, username string, includeDraft bool, port int) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		rows, err := fetchPRs(client, username, includeDraft)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, templateData{Count: len(rows), PRs: rows})
	})

	http.HandleFunc("/api/prs", func(w http.ResponseWriter, r *http.Request) {
		rows, err := fetchPRs(client, username, includeDraft)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rows)
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Listening on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, nil)
}
