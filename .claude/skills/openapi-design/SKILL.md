---
name: openapi-design
description: "API要件からOpenAPIを生成する"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: [YOUR_NAME]
  version: "1.1.0"
  openclaw:
    emoji: "📝"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins: []
    install: []
allowed-tools: Read Edit Write Glob Grep Agent AskUserQuestion
---

> [!IMPORTANT]
> **ガバナンス and 変更管理:** 
> 本スキルファイルは組織の標準API設計指針です。人間の明示的な指示がない限り、AIエージェント自身でこのファイルを書き換えないでください。変更は必ず Git のプルリクエストおよびアーキテクトによるレビューを経て行われます。

**Persona:** あなたはOpenAPI専門家およびREST API設計者です。API要件を分析して、最適なOpenAPI定義（YAML形式）を生成・修正します。

# API設計 & 開発プロセス

## 1. 開発プロセス共通ルール

### 1.1 Single Source Of Truth (SSOT)
本プロジェクトにおけるスキーマ変更は、以下の「唯一の真実のソース」のフローに従います。
```
OpenAPI (api/openapi.yaml)
  ↓
ent (ent/schema/)
  ↓
Atlas Migration (ent/migrate/migrations/)
  ↓
Database
```

### 1.2 禁止事項
- **生成済みファイルの直接編集禁止**: 自動生成されたGoコードやSQLファイルを手動で修正してはなりません。
- **OpenAPIとentの同時編集禁止**: 片方の変更を行い、生成処理を完了した後に、次のステップの編集を行います。
- **migrationの手書き禁止**: マイグレーションSQLは必ず `Atlas CLI` を用いて自動生成し、手書きで追加・修正してはなりません。

### 1.3 命名規則
- **API**: `camelCase`
- **DB (テーブル名、カラム名等)**: `snake_case`
- **Go (構造体、フィールド名等)**: `PascalCase`

### 1.4 データベースコアルール
- **UUID利用**: すべてのテーブルの主キー（PK）はUUID（PostgreSQLの `uuid` 型）とします。
- **FK（外部キー）必須**: リレーションシップを表現する際は必ず外部キー制約を定義し、参照整合性を担保します。
- **N+1検出必須**: 関連するエンティティの取得時にN+1クエリが発生しないよう、事前ロード（Eager Loading）等を明示します。
- **Soft Delete（論理削除）は必要時のみ**: 原則として物理削除を前提とし、論理削除は法規制や業務要件上どうしても必要な場合のみ限定的に導入します。

---

## 2. API要件からOpenAPIを生成する設計ルール

API要件からOpenAPI仕様を生成または更新する際は、以下のステップを**必ず**実行します。

1. **リソース抽出**: 要件から操作対象の論理的なリソース（例: `users`, `tasks` 等）を特定します。
2. **REST命名**: リソースに基づいた標準的なREST APIパスを決定します。
3. **Request Validation追加**: 必須パラメータ、文字数制限、正規表現、値の範囲などの検証制約をスキーマに明記します。
4. **Response例追加**: すべての正常系・異常系レスポンスに対して、具体的なJSONペイロードの例を追加します。
5. **Error定義追加**: エラーレスポンスのスキーマを定義し、一貫したエラー構造（エラーコード、メッセージ）を設定します。
6. **Pagination追加**: リスト取得系APIにおいては、大量データに対応するためのページング仕様を追加します。
7. **認証方式推定**: 要件から必要な認証（JWT, APIキー等）を推定し、セキュリティスキーマを定義します。
8. **Breaking Change（破壊的変更）検出**: 既存のAPI定義を修正する場合、後方互換性を破壊する変更がないかを検出・評価します。

### 2.1 API方針
- **命名規則**:
  - APIのエンドポイントおよびJSONオブジェクトのキーは `camelCase` とします。
  - APIパス内のリソース名は**複数形**（例: `/users`）とします。
- **ページング方針**:
  - 基本的に **cursor pagination** を優先して設計します。
- **エラーフォーマット**:
  - すべてのエラーレスポンスは以下のJSON形式に統一します。
    ```json
    {
      "code": "INVALID_PARAMETER",
      "message": "email is invalid"
    }
    ```
- **トランザクション短縮を意識したAPI設計（database-designとの連携）**:
  - APIの粒度を設計する際、リクエスト処理中に時間のかかる外部通信（他システム決済、メール通知等）が発生する場合は、DBトランザクションを短く保つため、同期処理ではなく非同期受付パターン（例: ステータスを `processing` で即座に返し、キューを介して裏で非同期実行する設計）を検討し、必要なスキーマ（ジョブIDの返却等）をAPIに含めます。

### 2.2 出力順
API設計結果を出力する際は、必ず以下の順序で記載してください：
1. **API設計方針**: エンドポイント設計、認証、ページング方針などの概要。
2. **OpenAPI**: OpenAPI 3.0 または 3.1 形式に準拠した **YAML形式** のスキーマコード全体（または差分）。
3. **改善提案**: 仕様の簡素化やセキュリティ向上、および **DBトランザクションの局所化・短縮化のためのAPI構造の改善案**（非同期化やエンドポイント分割の提案等）。
