# Go & SRE/DB/Security 開発用 GitHub テンプレートリポジトリ

このリポジトリは、Go (Golang) によるセキュアで高信頼なWebアプリケーション開発を迅速に開始するための、GitHub テンプレートリポジトリです。
CIでの静的解析、脆弱性診断、自動タグ付け (tagpr)、リリース管理 (GoReleaser) のパイプラインがあらかじめ統合されているほか、AIエージェントの回答品質を向上させるための日本語カスタムスキル（`.claude/skills`）を同梱しています。

## 🚀 特徴

1. **セキュアコーディングのお手本**: `main.go` には `slog` を用いた機密情報（パスワードやAPIキー等）の静的・動的マスキング処理が含まれています。
2. **自動リリースパイプライン (tagpr & GoReleaser)**:
   - `main` ブランチへのPRマージ時にリリース用PRが自動で作成・更新されます。
   - リリースPRをマージすると自動的に `vX.Y.Z` タグが打たれ、GitHub Releases にクロスコンパイルされたバイナリが公開されます。
3. **継続的インテグレーション (CI)**:
   - `golangci-lint` による静的解析。
   - `govulncheck` による依存パッケージの脆弱性診断。
   - 競合検知付きの `go test` による自動検証。
4. **AIエージェント用カスタムスキル**:
   - 開発時に Claude Code や Cursor 等のAIエージェントに読み込ませることで、SRE/DBA/セキュリティの専門知識に基づいた設計・実装・レビューを自動で実施させることができます。

---

## 🛠️ クイックスタート

### 1. このリポジトリから新規リポジトリを作成
GitHubの「Use this template」ボタンから、ご自身のリポジトリを作成します。

### 2. モジュール名の変更
作成したリポジトリの `go.mod` 内のモジュール名を変更します。
```go
module github.com/your-username/your-repo-name
```
また、`main.go` や `.goreleaser.yaml` などに含まれるプロジェクト名も必要に応じて書き換えてください。

### 3. AIカスタムスキルのインストール (任意)
同梱されているカスタムスキルをお使いのPC（グローバル）にインストールして、すべての Claude Code セッションで有効にします：
```bash
make install
```
*(内部的に `~/.claude/skills/` にコピーします)*

---

## ⚙️ 開発コマンド一覧

Makefile に定義されている以下のコマンドを使用して開発を進めます：

| コマンド | 説明 |
| :--- | :--- |
| `make fmt` | ソースコードのフォーマットおよびリンターによる自動修正 |
| `make lint` | `golangci-lint` を使用した静的解析の実行 |
| `make tidy` | 依存関係 (`go.mod` / `go.sum`) の整理 |
| `make vulncheck` | `govulncheck` を使用した脆弱性診断の実行 |
| `make test` | データ競合検知 (`-race`) およびカバレッジ測定付きテストの実行 |
| `make build` | `bin/app` へのコンパイルの実行 |
| `make release-snapshot` | `GoReleaser` によるローカルでのスナップショットビルドテスト |
| `make check` | 同梱スキルのマークダウン文法チェック |
| `make self-eval` | リポジトリが要件を満たしているかの自己評価の実行 (`REQUIREMENTS.md` の更新) |
| `make clean` | ビルド成果物やテストキャッシュのクリーンアップ |

---

## ☁️ さくらのクラウド Terraform CI/CD

本テンプレートには、さくらのクラウド用の Terraform CI/CD ワークフローが含まれています。`terraform/` ディレクトリ配下のファイルに変更があった場合のみトリガーされます。

### 🔑 GitHub Secrets の設定
このワークフローを正常に実行するには、事前にGitHubリポジトリの設定（`Settings -> Secrets and variables -> Actions`）から、以下の GitHub Secrets を必ず登録してください。

| Secret 名 | 説明 |
| :--- | :--- |
| `SAKURA_ACCESS_TOKEN` | さくらのクラウド API アクセストークン |
| `SAKURA_ACCESS_TOKEN_SECRET` | さくらのクラウド API アクセストークンシークレット |
| `AWS_ACCESS_KEY_ID` | S3互換バックエンド (State管理) 用の AWS Access Key ID |
| `AWS_SECRET_ACCESS_KEY` | S3互換バックエンド (State管理) 用の AWS Secret Access Key |

---

## 📋 REQUIREMENTS.md による品質自己評価とカスタマイズ

本テンプレートには、開発時のチェックリストとして `REQUIREMENTS.md` が含まれています。
`make self-eval` コマンドを実行すると、このファイルのチェックボックス（`[ ]` と `[x]`）が集計され、適合率（パーセンテージ）が動的に自動計算されてファイル下部に書き込まれます。

### 🛠️ カスタマイズ方法（独自の要件の追加）

開発するプロジェクトに合わせて、`REQUIREMENTS.md` に独自の機能要件や非機能要件を自由に追加・変更できます。

1. **`REQUIREMENTS.md` を開く**
2. **`## 📋 要件チェックリスト` セクションの下に、項目を追加する**
   - 項目は必ず `- [ ]` (未達成) または `- [x]` (達成) のフォーマットで記述してください。
   - 例：
     ```markdown
     ### 3. プロジェクト固有の機能要件
     - [ ] **R-3.1 ユーザー認証API**: JWTによる認証機能が実装され、E2Eテストがパスすること。
     - [ ] **R-3.2 データベース移行**: マイグレーションスクリプトが作成されていること。
     ```
3. **セルフチェックの実行**
   - 項目を追加した後に、以下のコマンドを実行します：
     ```bash
     make self-eval
     ```
   - これにより、追加したチェックボックスを含めた最新の適合率が自動的に集計され、`## 📈 自己評価結果` セクションが更新されます。

---

## 📄 ライセンスと作成者 (AUTHOR) のカスタマイズ

本リポジトリは **Apache License 2.0** でライセンスされています。複製して使用する際は、以下の項目をご自身の情報にカスタマイズしてご利用ください。

### 1. LICENSE ファイルの更新
リポジトリルートにある [LICENSE](file:///Users/shjtmy/gravity/go_sh0jitmy_template/LICENSE) ファイル内の `[Copyright Holder]` 部分をご自身の名称または組織名に書き換えてください。

### 2. カスタムスキル (author) の一括置換
同梱されている各カスタムスキル (`.claude/skills/*/SKILL.md`) のフロントマターに定義されている `author: [YOUR_NAME]` を、ご自身の名称に変更してください。

以下のワンコマンドを使用して、すべてのスキルファイルに対して一括置換を実行できます：

**macOS (BSD sed) の場合:**
```bash
find .claude/skills -name "SKILL.md" -exec sed -i '' 's/\[YOUR_NAME\]/ご自身の名前/g' {} +
```

**Linux (GNU sed) の場合:**
```bash
find .claude/skills -name "SKILL.md" -exec sed -i 's/\[YOUR_NAME\]/ご自身の名前/g' {} +
```

---

## 🔒 ログ出力時の機密情報保護指針
`golang-implementation` スキルに準拠し、本テンプレートでは以下のマスキング機構が実装されています。

- **`SecretString` 型**: ログに出力しようとすると、自動的に `[REDACTED]` に置き換わります。
- **`HashableSecret` 型**: ソルト付きハッシュ化された値を出力し、ログの検索性を維持しつつ秘匿します。
- **`NewSecureJSONHandler`**: ログのキー名が `password`, `token`, `secret`, `authorization` の属性を検知した場合、動的に値を `[REDACTED]` へ一括マスキングします。
