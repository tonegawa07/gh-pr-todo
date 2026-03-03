# gh pr-todo

A [gh](https://cli.github.com/) extension that lists PRs you're involved in (as a reviewer, commenter, etc.) along with your review status.

## Installation

```bash
gh extension install tonegawa07/gh-pr-todo
```

## Usage

```bash
gh pr-todo
```

Shows all open PRs you're involved in (excluding your own), with your review status. PRs are sorted by status priority so that unreviewed PRs appear first.

Status icons: ⏳ Unreviewed / 💬 Commented / 🔴 Changes requested / ✅ Approved

In supported terminals (iTerm2, Windows Terminal, etc.), each column is a clickable hyperlink.

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
2. Searches for open PRs you're involved in, excluding your own (`involves:@me -author:@me`)
3. Displays all matching PRs with your review status, sorted by priority (unreviewed first)

By default, draft PRs are excluded.

## Development

```bash
go build
gh extension install .
```

## Release

```bash
git tag vX.Y.Z
git push --tags
# GitHub Actions automatically builds cross-platform binaries and creates the release
# Do NOT use `gh release create` — it publishes a release before binaries are ready
```
