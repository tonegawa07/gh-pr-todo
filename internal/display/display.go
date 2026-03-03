package display

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/tonegawa07/gh-pr-todo/internal/github"
)

var (
	bold    = "\033[1m"
	dim     = "\033[2m"
	reset   = "\033[0m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	cyan    = "\033[36m"
	magenta = "\033[35m"
)

func init() {
	if !isTerminal() {
		bold, dim, reset = "", "", ""
		red, green, yellow, cyan, magenta = "", "", "", "", ""
	}
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// reviewStateLabel はステータスの色付きラベルを返す
func reviewStateLabel(state string) string {
	switch state {
	case "":
		return yellow + "⏳" + reset
	case "CHANGES_REQUESTED":
		return red + "🔴" + reset
	case "COMMENTED":
		return cyan + "💬" + reset
	case "DISMISSED":
		return magenta + "🚫" + reset
	case "APPROVED":
		return green + "✅" + reset
	default:
		return dim + state + reset
	}
}

// reviewStatePriority はステータスのソート優先度を返す (小さいほど上に表示)
func reviewStatePriority(state string) int {
	switch state {
	case "":
		return 0
	case "CHANGES_REQUESTED":
		return 1
	case "COMMENTED":
		return 2
	case "DISMISSED":
		return 3
	case "APPROVED":
		return 4
	default:
		return 5
	}
}

// ciLabel は CI ステータスの絵文字を返す
func ciLabel(state string) string {
	switch state {
	case "SUCCESS":
		return green + "✅" + reset
	case "FAILURE", "ERROR":
		return red + "❌" + reset
	case "PENDING", "EXPECTED":
		return yellow + "🟡" + reset
	default:
		return dim + "➖" + reset
	}
}

// runeWidth は文字の表示幅を返す (全角=2, 半角=1)
func runeWidth(r rune) int {
	if r >= 0x1100 &&
		(r <= 0x115f || r == 0x2329 || r == 0x232a ||
			(r >= 0x2e80 && r <= 0x303e) ||
			(r >= 0x3040 && r <= 0x33bf) ||
			(r >= 0x3400 && r <= 0x4dbf) ||
			(r >= 0x4e00 && r <= 0xa4cf) ||
			(r >= 0xa960 && r <= 0xa97c) ||
			(r >= 0xac00 && r <= 0xd7a3) ||
			(r >= 0xf900 && r <= 0xfaff) ||
			(r >= 0xfe30 && r <= 0xfe6f) ||
			(r >= 0xff01 && r <= 0xff60) ||
			(r >= 0xffe0 && r <= 0xffe6) ||
			(r >= 0x1f000 && r <= 0x1faff) ||
			(r >= 0x20000 && r <= 0x2fffd) ||
			(r >= 0x30000 && r <= 0x3fffd)) {
		return 2
	}
	return 1
}

// stringWidth は文字列の表示幅を返す
func stringWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

// truncate は文字列を maxWidth に収まるように切り詰める
func truncate(s string, maxWidth int) string {
	w := 0
	for i, r := range s {
		rw := runeWidth(r)
		if w+rw > maxWidth-1 {
			return s[:i] + "…"
		}
		w += rw
	}
	return s
}

// padRight は文字列を表示幅 width になるまでスペースで埋める
func padRight(s string, width int) string {
	sw := stringWidth(s)
	if sw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-sw)
}

// padLeft は文字列を表示幅 width になるまで左側にスペースで埋める
func padLeft(s string, width int) string {
	sw := stringWidth(s)
	if sw >= width {
		return s
	}
	return strings.Repeat(" ", width-sw) + s
}

// hyperlink は OSC 8 ハイパーリンクを生成する (対応ターミナルでクリック可能)
func hyperlink(url, text string) string {
	if !isTerminal() {
		return text
	}
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

// PrintTable はコンパクトな1行テーブルで PR 一覧を表示
func PrintTable(prs []github.PullRequest) {
	if len(prs) == 0 {
		fmt.Printf("%s✅ レビュー依頼されているPRはありません！%s\n", green, reset)
		return
	}

	// ステータス優先度でソート (同じ優先度ならリポジトリ名 → PR番号)
	sort.Slice(prs, func(i, j int) bool {
		pi := reviewStatePriority(prs[i].MyReviewState)
		pj := reviewStatePriority(prs[j].MyReviewState)
		if pi != pj {
			return pi < pj
		}
		if prs[i].Repo != prs[j].Repo {
			return prs[i].Repo < prs[j].Repo
		}
		return prs[i].Number < prs[j].Number
	})

	// 列幅の計算
	maxRepo := len("REPO")
	maxNum := len("#")
	maxTitle := len("TITLE")
	maxAuthor := len("AUTHOR")
	type row struct {
		repo   string
		num    string
		title  string
		url    string
		author string
		label  string // review status emoji
		ci     string // CI status emoji
	}

	rows := make([]row, len(prs))
	for i, pr := range prs {
		numStr := fmt.Sprintf("#%d", pr.Number)
		rows[i] = row{
			repo:   pr.Repo,
			num:    numStr,
			title:  pr.Title,
			url:    pr.URL,
			author: pr.Author,
			label:  reviewStateLabel(pr.MyReviewState),
			ci:     ciLabel(pr.CIState),
		}
		if w := stringWidth(pr.Repo); w > maxRepo {
			maxRepo = w
		}
		if w := stringWidth(numStr); w > maxNum {
			maxNum = w
		}
		if w := stringWidth(pr.Title); w > maxTitle {
			maxTitle = w
		}
		if w := stringWidth(pr.Author); w > maxAuthor {
			maxAuthor = w
		}
	}

	// タイトルの最大幅を制限
	const titleLimit = 50
	if maxTitle > titleLimit {
		maxTitle = titleLimit
	}

	// ヘッダー
	header := fmt.Sprintf("     %s  %s  %s  %s  %s",
		padRight("REPO", maxRepo),
		padLeft("#", maxNum),
		padRight("TITLE", maxTitle),
		padRight("AUTHOR", maxAuthor),
		"CI",
	)
	fmt.Printf("%s%s%s\n", bold, header, reset)
	fmt.Println(strings.Repeat("─", stringWidth(header)))

	// 各行
	for _, r := range rows {
		title := r.title
		if stringWidth(title) > maxTitle {
			title = truncate(title, maxTitle)
		}
		displayTitle := padRight(title, maxTitle)
		linkedTitle := hyperlink(r.url, displayTitle)

		linkedRepo := hyperlink("https://github.com/"+r.repo, padRight(r.repo, maxRepo))
		linkedNum := hyperlink(r.url, padLeft(r.num, maxNum))
		linkedAuthor := hyperlink("https://github.com/"+r.author, padRight(r.author, maxAuthor))

		fmt.Printf(" %s  %s  %s  %s  %s  %s\n",
			r.label,
			linkedRepo,
			linkedNum,
			linkedTitle,
			linkedAuthor,
			r.ci,
		)
	}
}
