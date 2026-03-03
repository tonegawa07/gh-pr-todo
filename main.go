package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tonegawa07/gh-pr-todo/internal/display"
	"github.com/tonegawa07/gh-pr-todo/internal/github"
)

const version = "0.2.0"

func main() {
	var (
		includeDraft bool
		jsonOutput   bool
		showVersion  bool
	)

	flag.BoolVar(&includeDraft, "include-draft", false, "Draft PR も含める")
	flag.BoolVar(&jsonOutput, "json", false, "JSON 形式で出力")
	flag.BoolVar(&showVersion, "v", false, "バージョン表示")
	flag.BoolVar(&showVersion, "version", false, "バージョン表示")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `レビュー依頼されている PR 一覧と対応状況を表示します。

USAGE
  gh pr-todo [flags]

FLAGS
      --include-draft      Draft PR も含める
      --json               JSON 形式で出力
  -v, --version            バージョン表示
`)
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("gh-pr-todo %s\n", version)
		return
	}

	if err := run(includeDraft, jsonOutput); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}
}

func run(includeDraft, jsonOutput bool) error {
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

	// ── レビュー依頼 PR 取得 ──
	prs, err := client.SearchReviewRequested(username)
	if err != nil {
		return err
	}

	// ── Draft フィルタ ──
	if !includeDraft {
		var filtered []github.PullRequest
		for _, pr := range prs {
			if !pr.Draft {
				filtered = append(filtered, pr)
			}
		}
		prs = filtered
	}

	// ── 出力 ──
	if jsonOutput {
		data, err := github.ToJSON(prs)
		if err != nil {
			return err
		}
		fmt.Println(data)
	} else {
		display.PrintTable(prs)
	}

	return nil
}
