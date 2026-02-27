# gh pr-todo

A [gh](https://cli.github.com/) extension that lists unmerged PRs you haven't approved yet.

## Installation

```bash
gh extension install <your-username>/gh-pr-todo
```

## Usage

```bash
gh pr-todo
```

This lists all unapproved PRs where you are requested as a reviewer.

### Specify additional repositories

To also monitor repositories where you are not explicitly requested as a reviewer:

```bash
gh pr-todo -r owner/repo1 -r owner/repo2
```

### Config file

Save repositories in a config file so you don't have to pass `-r` every time:

```bash
mkdir -p ~/.config/gh-pr-todo
cat > ~/.config/gh-pr-todo/config.yml << EOF
repos:
  - owner/repo1
  - owner/repo2
  - owner/repo3
EOF
```

Then `gh pr-todo` alone will check both your assigned reviews and the configured repositories.

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
| `-r, --repo OWNER/REPO` | Additional repositories (can be specified multiple times) |
| `--include-mine` | Include PRs you authored |
| `--include-draft` | Include draft PRs |
| `--json` | Output in JSON format |
| `-v, --version` | Show version |

## How it works

1. Uses `gh` authentication as-is (no extra token management needed)
2. Searches for PRs where you are requested as a reviewer (always runs)
3. Fetches open PRs from repositories specified in the config file or via `-r`
4. Checks your review state on each PR and displays those not yet `APPROVED`

By default, your own PRs and draft PRs are excluded.

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
