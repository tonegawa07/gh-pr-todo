package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tonegawa07/gh-pr-todo/internal/display"
	"github.com/tonegawa07/gh-pr-todo/internal/github"
	"github.com/tonegawa07/gh-pr-todo/internal/server"
)

var (
	bold  = "\033[1m"
	reset = "\033[0m"
)

func init() {
	fi, err := os.Stdout.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		bold, reset = "", ""
	}
}

const version = "0.4.0"

func main() {
	var (
		includeDraft bool
		jsonOutput   bool
		showVersion  bool
		serve        bool
		port         int
		showMine     bool
		showReviews  bool
	)

	flag.BoolVar(&includeDraft, "include-draft", false, "Draft PR も含める")
	flag.BoolVar(&jsonOutput, "json", false, "JSON 形式で出力")
	flag.BoolVar(&showVersion, "v", false, "バージョン表示")
	flag.BoolVar(&showVersion, "version", false, "バージョン表示")
	flag.BoolVar(&serve, "serve", false, "Web ダッシュボードを起動")
	flag.IntVar(&port, "p", 8080, "サーバーのポート番号")
	flag.IntVar(&port, "port", 8080, "サーバーのポート番号")
	flag.BoolVar(&showMine, "mine", false, "自分の PR のみ表示")
	flag.BoolVar(&showReviews, "reviews", false, "レビュー依頼された PR のみ表示")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `自分の PR とレビュー依頼されている PR 一覧を表示します。

USAGE
  gh pr-todo [flags]

FLAGS
      --mine               自分の PR のみ表示
      --reviews            レビュー依頼された PR のみ表示
      --include-draft      Draft PR も含める
      --json               JSON 形式で出力
      --serve              Web ダッシュボードを起動
  -p, --port               サーバーのポート番号 (デフォルト: 8080)
  -v, --version            バージョン表示
`)
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("gh-pr-todo %s\n", version)
		return
	}

	if err := run(includeDraft, jsonOutput, serve, port, showMine, showReviews); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}
}

func filterDraft(prs []github.PullRequest) []github.PullRequest {
	var filtered []github.PullRequest
	for _, pr := range prs {
		if !pr.Draft {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}

func run(includeDraft, jsonOutput, serve bool, port int, showMine, showReviews bool) error {
	// 両方 false → 両方表示
	if !showMine && !showReviews {
		showMine = true
		showReviews = true
	}

	// ── クライアント初期化 ──
	client, err := github.NewClient()
	if err != nil {
		return err
	}

	// ── ユーザー名取得 ──
	username, err := client.GetAuthenticatedUser()
	if err != nil {
		return err
	}

	// ── Web ダッシュボード ──
	if serve {
		return server.Start(client, username, includeDraft, port, showMine, showReviews)
	}

	// ── 自分の PR 取得 ──
	var minePRs []github.PullRequest
	if showMine {
		minePRs, err = client.SearchMyPRs(username)
		if err != nil {
			return err
		}
		if !includeDraft {
			minePRs = filterDraft(minePRs)
		}
	}

	// ── レビュー依頼 PR 取得 ──
	var reviewPRs []github.PullRequest
	if showReviews {
		reviewPRs, err = client.SearchReviewRequested(username)
		if err != nil {
			return err
		}
		if !includeDraft {
			reviewPRs = filterDraft(reviewPRs)
		}
	}

	// ── 出力 ──
	if jsonOutput {
		data, err := github.SectionsToJSON(minePRs, reviewPRs)
		if err != nil {
			return err
		}
		fmt.Println(data)
	} else {
		if showMine {
			fmt.Printf("%s── My PRs ──%s\n", bold, reset)
			display.PrintMyPRsTable(minePRs)
		}
		if showReviews {
			if showMine {
				fmt.Println()
			}
			fmt.Printf("%s── Review PRs ──%s\n", bold, reset)
			display.PrintTable(reviewPRs)
		}
	}

	return nil
}
