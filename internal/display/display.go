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

// Log はステータスメッセージを stderr に出力
func Log(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s%s\n", dim, msg, reset)
}

// Logf はフォーマット付きで stderr に出力
func Logf(format string, args ...any) {
	Log(fmt.Sprintf(format, args...))
}

func reviewStateLabel(state string) string {
	switch state {
	case "":
		return yellow + "未レビュー" + reset
	case "CHANGES_REQUESTED":
		return red + "変更リクエスト済" + reset
	case "COMMENTED":
		return cyan + "コメント済" + reset
	case "DISMISSED":
		return magenta + "却下済" + reset
	default:
		return dim + state + reset
	}
}

// PrintTable はリポジトリごとにグループ化して PR 一覧を表示
func PrintTable(prs []github.PullRequest) {
	if len(prs) == 0 {
		fmt.Printf("\n%s✅ 未ApproveのPRはありません！%s\n", green, reset)
		return
	}

	byRepo := make(map[string][]github.PullRequest)
	var repoOrder []string
	for _, pr := range prs {
		if _, exists := byRepo[pr.Repo]; !exists {
			repoOrder = append(repoOrder, pr.Repo)
		}
		byRepo[pr.Repo] = append(byRepo[pr.Repo], pr)
	}
	sort.Strings(repoOrder)

	fmt.Printf("\n%s📋 未ApproveのPR一覧 (%d 件)%s\n\n", bold, len(prs), reset)

	for _, repo := range repoOrder {
		repoPRs := byRepo[repo]
		sort.Slice(repoPRs, func(i, j int) bool {
			return repoPRs[i].UpdatedAt > repoPRs[j].UpdatedAt
		})

		fmt.Printf("%s%s%s%s (%d 件)\n", bold, cyan, repo, reset, len(repoPRs))
		fmt.Println(strings.Repeat("─", 70))

		for _, pr := range repoPRs {
			stateLabel := reviewStateLabel(pr.MyReviewState)
			labels := ""
			if len(pr.Labels) > 0 {
				labels = fmt.Sprintf(" %s[%s]%s", dim, strings.Join(pr.Labels, ", "), reset)
			}

			fmt.Printf("  %s#%d%s %s\n", bold, pr.Number, reset, pr.Title)
			fmt.Printf("    by %s%s%s  │  %s%s\n", green, pr.Author, reset, stateLabel, labels)
			fmt.Printf("    %s%s%s\n", dim, pr.URL, reset)
			fmt.Println()
		}
	}
}
