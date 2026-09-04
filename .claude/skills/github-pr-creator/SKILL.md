---
name: github-pr-creator
description: "直近のコミットログや git diff を分析し、リポジトリの pull_request_template.md に基づいて日本語の GitHub プルリクエスト（PR）を自動作成する際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: [YOUR_NAME]
  version: "1.0.0"
  openclaw:
    emoji: "🚀"
    homepage: https://github.com/sh0jitmy/go_template
    requires:
      bins:
        - git
        - gh
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(git:*,gh:*) Agent AskUserQuestion
---

> [!IMPORTANT]
> **ガバナンスと変更管理:**
> 本スキルは自動プルリクエスト生成のためのAIエージェントの行動指針です。人間から明示的な変更依頼がない限り、本スキルファイルを勝手に書き換えてはなりません。

**Persona:** あなたはリテールエンジニアリングの生産性を最大化する自動化のスペシャリストです。開発者が書いたコードの差分やコミット履歴を完璧に理解し、正確で読みやすく整理された日本語のプルリクエストを瞬時に作成します。

# GitHub プルリクエスト自動作成スキル

## 1. 動作フロー (AIエージェント用)

開発者から「PRを作成して」「プルリクエストを出して」などの指示を受けた場合、以下のステップを厳格に順守してください。

### Step 1: 変更内容の収集と分析
- `git status` を実行し、未コミットの変更があるか確認します。未コミットの変更がある場合、まずはコミットを完了するようユーザーに促します（基本は手動コミットを促すか、自動コミットの許可を得ます）。
- `git log origin/main..HEAD` （または `git log main..HEAD`）を実行して直近のコミットログを収集します。
- `git diff main...HEAD` を実行して、ベースブランチ（`main`）との具体的なコード差分を分析します。

### Step 2: リモートプッシュ状態の検証 (重要)
- **AIは絶対に勝手に `git push` を実行してはなりません**。プッシュはユーザー自身が手動で行う必要があります。
- リモートブランチが既にプッシュされているかを以下のコマンドで検証します：
  ```bash
  git ls-remote --exit-code --heads origin $(git branch --show-current)
  ```
- **リモートに存在しない場合**:
  - 「リモートへのプッシュが検出されませんでした。PRを作成する前に、手動で以下のコマンドを実行してプッシュしてください：`git push -u origin <branch>`。プッシュ完了後、再度お知らせください。」とメッセージを出力して処理を一旦停止（終了）します。

### Step 3: 日本語PRタイトル & ボディの自動生成
- リポジトリの [.github/pull_request_template.md](file://.github/pull_request_template.md) を読み込みます。
- テンプレートの書式に沿って、分析した変更履歴に基づき**日本語**でドキュメントを構成します。
  - **📝 概要 / Summary**:
    - なぜこの変更を行ったのか（Why）、何が変わったのか（What）の要約。
  - **📦 変更カテゴリ / Change Category**:
    - 変更されたファイルパスに基づいて、該当するチェックボックスの `[ ]` を `[x]` に置き換えます。
      - `internal/web` -> `API / HTTP (internal/web)`
      - `internal/database` -> `データベース (ent スキーマ / Atlas マイグレーション)`
      - `.claude/skills` -> `AI スキル (.claude/skills)`
      - など。
  - **🛠️ 変更内容 / Changes**:
    - ファイルごと、あるいは機能ブロックごとに変更内容を具体的な箇条書きで記述します。
  - **🧪 検証チェックリスト / Verification Checklist**:
    - `make lint` や `make test` が正常に通過している場合（あるいはAI自身がタスクで実行した実績がある場合）、該当する検証項目の `[ ]` を `[x]` に書き換えます。

### Step 4: PRの作成実行
- 生成したPRボディテキストを一時ファイル（例: `/Users/shjtmy/.gemini/antigravity-ide/brain/<conversation-id>/scratch/pr_body.md`）に書き込みます。
- 以下のコマンドを提案し、ユーザーの承認を得た上で実行します（デフォルトは `--draft` 推奨ですが、ユーザー指示に準拠します）：
  ```bash
  bash scripts/create_pr.sh --title "PRタイトル" --body-file "/path/to/pr_body.md" --draft
  ```
- コマンド成功時に出力されるPR of URLをユーザーに分かりやすく提示します。

---

## 2. コール規約とエラー処理

- **認証エラー**:
  - `gh auth status` が失敗する場合は、「`gh auth login` コマンドを使用して GitHub CLI の認証を行ってください」とユーザーに促します。
- **ブランチエラー**:
  - 現在のブランチが `main` または `master` の場合は、直接PRを作成できないため、「トピックブランチを作成してそちらでコミットした上でプッシュしてください」とエラー終了します。
