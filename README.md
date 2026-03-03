# gh pr-todo

A [gh](https://cli.github.com/) extension that lists PRs where you are requested as a reviewer, along with your review status.

## Installation

```bash
gh extension install tonegawa07/gh-pr-todo
```

## Usage

```bash
gh pr-todo
```

Shows all PRs where you are requested as a reviewer, with your review status (unreviewed, commented, changes requested, approved, etc.). PRs are sorted by status priority so that unreviewed PRs appear first.

### JSON output

```bash
# List URLs
gh pr-todo --json | jq -r '.[].url'

# Only unreviewed
gh pr-todo --json | jq '[.[] | select(.my_review_state == "")]'

# Count by repository
gh pr-todo --json | jq 'group_by(.repo) | map({repo: .[0].repo, count: length})'
```

## Options

| Flag | Description |
| --- | --- |
| `--include-draft` | Include draft PRs |
| `--json` | Output in JSON format |
| `-v, --version` | Show version |

## How it works

1. Uses `gh` authentication as-is (no extra token management needed)
2. Searches for PRs where you are requested as a reviewer
3. Displays all matching PRs with your review status, sorted by priority (unreviewed first)

By default, draft PRs are excluded.

## Development

```bash
go build
gh extension install .
```

## Release

```bash
git tag v0.1.0
git push origin v0.1.0
# GitHub Actions automatically builds cross-platform binaries
```
