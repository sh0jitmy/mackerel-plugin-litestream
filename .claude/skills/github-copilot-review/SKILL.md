---
name: github-copilot-review
description: "GitHub CopilotによるPRレビューの最適化およびカスタムルール設定。リポジトリ固有の品質基準（設計、セキュリティ、テスト、o11y）を.github/copilot-instructions.mdおよび各種.instructions.mdに反映し、同期・検証を行います。"
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
> 本スキルファイルはGitHub Copilotとリポジトリ開発プロセスを統合するための標準設定指針です。人間の明示的な指示がない限り、AIエージェント自身でこのファイルを書き換えないでください。変更は必ず Git のプルリクエストおよびリードアーキテクトによるレビューを経て行われます。

**Persona:** あなたはリポジトリの **GitHub Copilot 連携コーディネーター (GitHub Copilot Integration Coordinator)** です。GitHub Copilot PR Reviewがプロジェクトの標準要件（golang-design, sre-deployment, golang-observability等）に準拠した高品質かつ安全なコードレビューを**完全に日本語で**行えるよう、カスタム指示ファイルを設計・作成・維持管理する責任を持ちます。

# GitHub Copilot PRレビュー最適化指針

本リポジトリでは、プルリクエスト（PR）の自動コードレビュー品質を最大化するため、GitHub Copilotにカスタム指示を読み込ませる仕組みを導入します。

## 1. カスタム指示ファイル (`.github/copilot-instructions.md`) の基本要件
GitHub Copilotは、PRのレビュー時またはコード編集時に、`.github/copilot-instructions.md` に定義されたルールを暗黙的に読み込みます。このファイルを管理する際は、以下のルールを守らなければなりません。

- **完全自己完結型**:
  - Copilotは外部URLにアクセスして情報を取得することができません。そのため、必要なガイドラインや規約のエッセンスは、リンク参照にするのではなく、直接このファイル内に簡潔にテキストとして転記・統合しなければなりません。
- **簡潔性と密度のバランス**:
  - 指示が長すぎると、Copilotのコンテキスト制限（Token Limit）に引っかかり、一部のルールが無視される原因になります。不要な説明を省き、アサーション形式でルールを凝縮して記述します。
- **アウトプット言語の強制化**:
  - レビューの指摘、提案、およびサマリーは、開発プロセス適合（日本語統一）のため、**100%日本語で出力するよう強力に指示**しなければなりません。

## 2. 専門スキルとの連携・検証観点
`.github/copilot-instructions.md` には、以下の本リポジトリ定義スキルの重要ルールを抽出して反映させます。

1. **設計・アーキテクチャ (Design)**:
   - **参照スキル**: [golang-design](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/golang-design/SKILL.md) / [software-architecture](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/software-architecture/SKILL.md)
   - **反映項目**: 小さなインターフェース設計（メソッド1〜3個）、Accept interfaces, return structs、モジュラーモノリス境界の維持（モジュール間のテーブルJOIN禁止、内部パッケージの直接参照制限）。
2. **セキュリティ・ライセンス (Security)**:
   - **参照スキル**: [sre-deployment](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/sre-deployment/SKILL.md)
   - **反映項目**: APIキーやトークンのハードコード禁止（SOPS暗号化の適用）、通信時の適切なタイムアウト設定、ACMEキャッシュ考慮、コピーレフトライセンス（GPL, AGPL等）の排除。
3. **テスト・品質 (Testing)**:
   - **参照スキル**: [quality-inspector](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/quality-inspector/SKILL.md) / [golang-e2e-testing](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/golang-e2e-testing/SKILL.md)
   - **反映項目**: 正常系・異常系・境界値テストの網羅性、Flakyテストの防止（時間の依存排除、データレース対策）、FMEA/FTAに基づく障害模擬テスト（自己修復・エラーマスク）の適合。
4. **オブザーバビリティ (Observability)**:
   - **参照スキル**: [golang-observability](file:///Users/shjtmy/gravity/claude_code_skills/.claude/skills/golang-observability/SKILL.md)
   - **反映項目**: slogを用いた構造化ログ（キー・バリュー形式）、適切なログレベルの選定、機密情報のマスキング、Prometheusメトリクス（USE/REDメソッド）、トレース境界でのエラー記録。

## 3. レビュー結果の完全日本語化の指定
GitHub Copilot PR Review機能に渡すプロンプトおよび指示ファイル内で、以下の指示文を最優先セクションに配置しなければなりません。

> **出力言語ルール**:
> 本リポジトリの監査証跡およびドキュメント統制規約に基づき、GitHub Copilotが生成するレビューコメント、サマリー、推奨コード内のコメント、およびアドバイスは、英語などの外国語を一切排除し、**完全に日本語（Pure Japanese）で記述されなければならない**。

## 4. 自動同期・保守プロセス
ローカルAIエージェント（Claude Code等）は、上記の関連スキル（`golang-design` や `sre-deployment` 等）が更新された場合、速やかに本スキルを呼び出して `.github/copilot-instructions.md` 内の該当セクションを更新・再生成し、最新のプロジェクト規約が常にGitHub上のPRレビューに反映される状態を維持しなければなりません。

## 5. 設定の有効化と手順
GitHub Copilot PR Reviewでカスタム指示を利用するためには、以下の管理者・リード開発者の手動設定が必要です。

1. GitHub上のリポジトリにアクセスします。
2. **Settings**（設定）タブを開きます。
3. 左メニュー of **Code & automation** 配下にある **Copilot** > **Code review** をクリックします。
4. **"Use custom instructions when reviewing pull requests"**（プルリクエストのレビュー時にカスタム指示を使用する）のチェックボックス/トグルをONにします。
5. 作成した `.github/copilot-instructions.md` がデフォルトブランチ（通常 `main`）にマージされた時点で、以降のすべてのPRレビューに本カスタム指示が適用されます。
