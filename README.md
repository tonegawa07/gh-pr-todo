# gh pr-todo

自分がまだ Approve していない未マージ PR 一覧を取得する [gh](https://cli.github.com/) extension。

## インストール

```bash
gh extension install <your-username>/gh-pr-todo
```

## 使い方

```bash
gh pr-todo
```

これだけで、自分にレビュー依頼が来ている未 Approve の PR が一覧表示されます。

### 追加リポジトリを指定

レビュー依頼が来ていないリポジトリも監視したい場合：

```bash
gh pr-todo -r owner/repo1 -r owner/repo2
```

### 設定ファイルでリポジトリを登録

毎回 `-r` を打たなくて済むように、設定ファイルにリポジトリを書いておけます：

```bash
mkdir -p ~/.config/gh-pr-todo
cat > ~/.config/gh-pr-todo/config.yml << EOF
repos:
  - owner/repo1
  - owner/repo2
  - owner/repo3
EOF
```

以降は `gh pr-todo` だけで、アサイン分 + 設定ファイルのリポジトリを全て確認します。

### JSON 出力

```bash
# URL 一覧
gh pr-todo --json | jq -r '.[].url'

# 未レビューのみ
gh pr-todo --json | jq '[.[] | select(.my_review_state == "")]'

# リポジトリごとの件数
gh pr-todo --json | jq 'group_by(.repo) | map({repo: .[0].repo, count: length})'
```

## オプション

| フラグ | 説明 |
| --- | --- |
| `-r, --repo OWNER/REPO` | 追加リポジトリ (複数指定可) |
| `--include-mine` | 自分が作成した PR も含める |
| `--include-draft` | Draft PR も含める |
| `--json` | JSON 形式で出力 |
| `-v, --version` | バージョン表示 |

## 動作の仕組み

1. `gh` の認証情報をそのまま使用（トークン管理不要）
2. 自分にレビュー依頼が来ている PR を検索（常に実行）
3. 設定ファイル / `-r` 指定のリポジトリからオープン PR を取得
4. 各 PR の自分のレビュー状態を確認し、`APPROVED` 以外を表示

デフォルトで自分の PR と Draft PR は除外されます。

## 開発

```bash
go build
gh extension install .
```

## リリース

```bash
git tag v0.1.0
git push origin v0.1.0
# GitHub Actions がクロスプラットフォームバイナリを自動ビルド
```
