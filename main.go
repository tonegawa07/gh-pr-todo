package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tonegawa07/gh-pr-todo/internal/config"
	"github.com/tonegawa07/gh-pr-todo/internal/display"
	"github.com/tonegawa07/gh-pr-todo/internal/github"
)

const version = "0.1.0"

type repoList []string

func (r *repoList) String() string { return strings.Join(*r, ", ") }
func (r *repoList) Set(val string) error {
	*r = append(*r, val)
	return nil
}

func main() {
	var (
		repos        repoList
		includeMine  bool
		includeDraft bool
		jsonOutput   bool
		showVersion  bool
	)

	flag.Var(&repos, "r", "追加リポジトリ OWNER/REPO (複数指定可)")
	flag.Var(&repos, "repo", "追加リポジトリ OWNER/REPO (複数指定可)")
	flag.BoolVar(&includeMine, "include-mine", false, "自分が作成した PR も含める")
	flag.BoolVar(&includeDraft, "include-draft", false, "Draft PR も含める")
	flag.BoolVar(&jsonOutput, "json", false, "JSON 形式で出力")
	flag.BoolVar(&showVersion, "v", false, "バージョン表示")
	flag.BoolVar(&showVersion, "version", false, "バージョン表示")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `自分がまだ Approve していない未マージ PR 一覧を取得します。

USAGE
  gh pr-todo [flags]

FLAGS
  -r, --repo OWNER/REPO    追加リポジトリ (複数指定可)
      --include-mine       自分が作成した PR も含める
      --include-draft      Draft PR も含める
      --json               JSON 形式で出力
  -v, --version            バージョン表示

CONFIG
  ~/.config/gh-pr-todo/config.yml にリポジトリを設定:

    repos:
      - owner/repo1
      - owner/repo2

EXAMPLES
  gh pr-todo                              # アサイン分 + config のリポジトリ
  gh pr-todo -r owner/extra-repo          # 一時的にリポジトリ追加
  gh pr-todo --json | jq -r '.[].url'     # URL一覧
`)
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("gh-pr-todo %s\n", version)
		return
	}

	if err := run(repos, includeMine, includeDraft, jsonOutput); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}
}

func run(extraRepos repoList, includeMine, includeDraft, jsonOutput bool) error {
	// ── 設定読み込み ──
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("設定ファイルの読み込みに失敗: %w", err)
	}

	// config + CLI引数のリポジトリをマージ
	repos := append(cfg.Repos, extraRepos...)

	// ── クライアント初期化 ──
	client, err := github.NewClient()
	if err != nil {
		return err
	}

	// ── ユーザー名取得 ──
	display.Log("👤 ユーザー名を取得中...")
	username, err := client.GetAuthenticatedUser()
	if err != nil {
		return err
	}
	display.Logf("👤 ユーザー: %s", username)

	// ── PR 収集 (常にアサイン分 + 指定リポジトリ) ──
	collected, err := collectPRs(client, username, repos)
	if err != nil {
		return err
	}

	if len(collected) == 0 {
		if jsonOutput {
			fmt.Println("[]")
		} else {
			fmt.Printf("\n✅ 対象のオープンPRが見つかりませんでした\n")
		}
		return nil
	}

	// ── フィルタ ──
	display.Logf("🔎 %d 件のPRのレビュー状態を確認中...", len(collected))
	pending, err := filterPending(client, collected, username, includeMine, includeDraft)
	if err != nil {
		return err
	}
	display.ProgressClear()

	// ── 出力 ──
	if jsonOutput {
		data, err := github.ToJSON(pending)
		if err != nil {
			return err
		}
		fmt.Println(data)
	} else {
		display.PrintTable(pending)
	}

	return nil
}

func collectPRs(client *github.Client, username string, repos []string) ([]github.PullRequest, error) {
	seen := make(map[string]struct{})
	var result []github.PullRequest

	addUnique := func(prs []github.PullRequest) {
		for _, pr := range prs {
			key := pr.Key()
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				result = append(result, pr)
			}
		}
	}

	// 1) 常にアサイン分を取得
	display.Log("🔍 レビュー依頼されたPRを検索中...")
	assigned, err := client.SearchReviewRequested(username)
	if err != nil {
		return nil, err
	}
	addUnique(assigned)
	display.Logf("   → %d 件", len(assigned))

	// 2) 指定リポジトリのオープン PR
	for _, repo := range repos {
		display.Logf("📦 %s のオープンPRを取得中...", repo)
		prs, err := client.GetOpenPRs(repo)
		if err != nil {
			return nil, err
		}
		before := len(result)
		addUnique(prs)
		added := len(result) - before
		display.Logf("   → %d 件 (新規 %d 件)", len(prs), added)
	}

	return result, nil
}

func filterPending(client *github.Client, prs []github.PullRequest, username string, includeMine, includeDraft bool) ([]github.PullRequest, error) {
	var pending []github.PullRequest
	total := len(prs)

	for i, pr := range prs {
		display.Progress(i+1, total)

		if !includeMine && strings.EqualFold(pr.Author, username) {
			continue
		}
		if !includeDraft && pr.Draft {
			continue
		}

		state, err := client.GetMyReviewState(pr.Repo, pr.Number, username)
		if err != nil {
			return nil, err
		}
		pr.MyReviewState = state

		if state != "APPROVED" {
			pending = append(pending, pr)
		}
	}

	return pending, nil
}
