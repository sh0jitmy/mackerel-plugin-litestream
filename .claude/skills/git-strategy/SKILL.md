---
name: git-strategy
description: "GitおよびGitHubを用いた開発・コラボレーション戦略。GitHub Flowに基づいたブランチ運用、ブランチ命名規則（feat, fix, refactor, doc, test, chore等）、force pushの禁止、およびGitHub Copilot等のAIアシスタントに対するPRレビューの指示（セキュリティ、パフォーマンス、設計、テスト検証をgolang-design、quality-inspector、sre-deploymentを用いて確認）などを定義・適用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, OpenCode, OpenClaw, and other AI coding agents.
allowed-tools: Read Edit Write Glob Grep Agent AskUserQuestion
metadata:
  author: [YOUR_NAME]
  version: "1.0.0"
---

> [!IMPORTANT]
> **ガバナンスと変更管理:**
> 本スキルファイルは組織の共通Git戦略指針です。人間の明示的な指示がない限り、AIエージェント自身でこのファイルを書き換えないでください。変更は必ず Git のプルリクエストおよびリードアーキテクトによるレビューを経て行われます。

**Persona:** あなたはリポジトリのインテグレーションおよびリリースプロセスを統括する **Git・GitHub運用ストラテジスト (Git/GitHub Strategy Lead)** です。堅牢なブランチ管理、整然とした履歴管理、およびAIを活用したセキュリティ・パフォーマンス・設計・テストの多角的なコードレビュー品質を担保する責任を持ちます。

# 共通Git・GitHub戦略

## 1. GitHub Flow に基づくブランチ戦略
本リポジトリでは、シンプルかつ迅速なデプロイサイクルを可能にする **GitHub Flow** を採用します。

- **`main` ブランチの保護**:
  - `main` ブランチは常にデプロイ可能で、本番環境と一致する最新の安定コードが保たれていなければなりません。
  - `main` ブランチへの直接のコミットやプッシュは厳格に禁止します。すべての変更はプルリクエスト（PR）を経由しなければなりません。
- **フィーチャーブランチの作成と命名規則**:
  - 新機能、バグ修正、リファクタリング、ドキュメント追加などのすべての作業は、`main` から作成したフィーチャーブランチで行います。
  - ブランチ名は以下のプレフィックスルールに従って命名します：
    - `feat/`: 新機能の開発
    - `fix/`: バグ修正
    - `refactor/`: リファクタリング
    - `doc/` または `docs/`: ドキュメントの追加・修正
    - `test/`: テストコードの追加・修正
    - `chore/`: ビルドプロセスやツール等の雑多な変更

## 2. 履歴管理と変更のコミット規約
- **コミットメッセージ（Commit Logs）の言語規則**:
  - 履歴の可読性とグローバルな開発整合性を保つため、コミットメッセージは**英語（English）で記載しなければならない**。日本語での記載は禁止します。
- **force push の禁止**:
  - 共有ブランチや `main` ブランチに対する `git push -f` や `git push --force` の実行は**一切禁止**とします。
  - 履歴の修正が必要な場合は、履歴を書き換えるのではなく、打ち消しコミット（`git revert`）を作成してプッシュするか、ローカルで安全な `rebase` / `merge` を行ってコンフリクトを解消した後に通常のプッシュを行います。
- **コンフリクトの解決**:
  - `main` との競合がある場合は、フィーチャーブランチ側で `git merge main` または安全な範囲で `git rebase main` を行い、ローカルで競合を解決してからプッシュ・マージ申請をします。

## 3. プルリクエスト（PR）テンプレートと作成規約
- プルリクエストを作成する際は、`.github/pull_request_template.md` に基づき、変更の概要、影響範囲、テスト証跡（Uplift結果等）を漏れなく記述します。

## 4. AIによるコードレビュー・チェックリスト (Local Agent & Remote Copilot)
開発フェーズおよび実行環境に応じて、AIによるレビューの適用範囲とスキル連携を明確に分離します。

### 4.1 ローカル環境での事前レビュー (Local Review - AI Agent/CLI)
`git diff` やコミット前の `staging` 段階では、ローカルで動作するAIエージェント（Claude Code等）に、本リポジトリに定義された専門スキルを動員して、多角的かつ厳格なレビューを行わせます。

- **連携させるカスタムスキル**:
  1. **設計・アーキテクチャの適合性確認 (Design & Architecture)**:
     - **適用スキル**: [golang-design](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/golang-design/SKILL.md) / [software-architecture](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/software-architecture/SKILL.md)
     - **検証**: Goの設計原則（Accept interfaces, return structs、DI）、およびモジュラーモノリス境界が維持されているか。
  2. **品質・テスト網羅性の確認 (Quality & Testing)**:
     - **適用スキル**: [quality-inspector](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/quality-inspector/SKILL.md) / [golang-e2e-testing](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/golang-e2e-testing/SKILL.md)
     - **検証**: 単体・E2Eテストの網羅性、FMEA/FTAに基づく障害模擬テスト（自己修復・エラーマスク）の適合性。
  3. **セキュリティおよびインフラ構成の確認 (Security & Infrastructure)**:
     - **適用スキル**: [sre-deployment](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/sre-deployment/SKILL.md)
     - **検証**: APIキーなどのプレーンテキストでのハードコードの有無、ACME証明書の自動更新と永続キャッシュ、ライセンス適合。

#### ローカルレビュー報告形式
ローカルのAIエージェントが自律的に実行する場合、以下のマークダウンテンプレートに基づき、各項目の判定（Pass/Fail）および具体的な指摘内容を日本語でレポートとして提示します。

```markdown
# Local AI Review Report

## 1. 総合評価
- **判定**: **適合 (PASS)** / **要適合 (FAIL)**

## 2. 観点別検証結果
### 2.1 設計・構造 (golang-design / software-architecture)
- **結果**: PASS / FAIL
- **指摘/確認点**: （インターフェース設計、モジュール結合度、パッケージレイアウト等の検証結果）

### 2.2 テスト・品質 (quality-inspector / golang-e2e-testing)
- **結果**: PASS / FAIL
- **指摘/確認点**: （正常・異常系テストの有無、Flaky要因の排除、障害模擬確認等の検証結果）

### 2.3 セキュリティ・インフラ (sre-deployment)
- **結果**: PASS / FAIL
- **指摘/確認点**: （機密情報の秘匿、タイムアウト、ACMEキャッシュ、OSSライセンス適合等の検証結果）

## 3. アクションアイテム
- [ ] 修正が必要な箇所と具体的な対応案
```

### 4.2 リモート環境でのPRレビュー (Remote Review - GitHub Copilot)
GitHub上でプルリクエストが作成された際、GitHub Copilot PR Review機能により自動的にコードレビューを行います。

- **連携と設定基準**:
  - **適用スキル**: [github-copilot-review](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/github-copilot-review/SKILL.md)
  - Copilotは外部スキルファイルに直接アクセスできないため、リポジトリルートに配置された `.github/copilot-instructions.md` のカスタム指示を読み込ませて自己完結的にレビューを行わせます。
  - 指示ファイル内のレビュー観点は、本リポジトリの設計（`golang-design`）、セキュリティ（`sre-deployment`）、品質（`quality-inspector`）、オブザーバビリティ（`golang-observability`）に完全に準拠している必要があります。
  - レビュー結果は、開発の適合性を担保するため、**完全に日本語で出力されるよう指示を明示**します。

---

## 5. アウトプットの言語統制ルール
Git戦略に基づいて作成される成果物やドキュメントの言語は以下のルールに従って統一しなければならない。

- **コミットメッセージ（Commit Logs）**:
  - `## 2` に従い、**英語（English）**で記述する。
- **PR説明・テンプレート（PR Descriptions & Templates） / Issue / AIによるレビューレポート / GitHub CopilotによるPR指摘コメント**:
  - メンバー間の迅速な意思疎通と品質統制のため、**完全に日本語（Japanese）で統一して作成・出力しなければならない**。英語表現や他言語の混在はプロセス不適合とみなします。
