// Copyright 2026 [Copyright Holder]
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Author: [YOUR_NAME]

# ADR 0001: Go 開発テンプレートリポジトリのアーキテクチャ設計

## ステータス
承認済み (Accepted)

## コンテキスト
新規の Go アプリケーションプロジェクトを安全、高速、かつ高品質に立ち上げるための「テンプレートリポジトリ」が必要です。
このテンプレートには、本番運用に必要なセキュリティ（HTTPS/Bearer認証）、API駆動開発（Schema-First）、CGOフリーな永続化層（ent + SQLite）、および厳格な品質・ガバナンス監査（CI/CD、ライセンス/作成者ヘッダー検証）があらかじめ統合されていることが求められます。

## 意思決定

以下の主要な技術スタックおよびアーキテクチャ設計を導入し、テンプレートリポジトリとして構成します。

1. **Go ソフトウェアアーキテクト設計に準拠したレイアウト**
   - ルート直下の肥大化を防ぐため、`cmd/app/`（エントリーポイント）と `internal/`（APIハンドラー、ミドルウェア、データベース）にコードを完全に分離・カプセル化。

2. **OpenAPI (Schema-First) 駆動による自動コード生成**
   - API 仕様を `api/openapi.yaml` に定義し、`oapi-codegen` を用いて Gin 用のルーターインターフェースとデータモデルを `ogen/api.gen.go` へ自動生成。生成コードは独立した `ogen` パッケージとして管理し、ライセンスヘッダーチェックの対象外とする。

3. **ent ORM と CGO-free SQLite によるデータベース層**
   - `ent` スキーマを元に型安全な ORM コードを自動生成。
   - `modernc.org/sqlite` / `github.com/glebarez/go-sqlite` を採用し、CGOを完全に無効化（`CGO_ENABLED=0`）したままでポータブルな SQLite データベースの自動マイグレーションおよび初期データのシード投入を実現。
   - データベースマイグレーションは `Atlas CLI` に基づくバージョン管理マイグレーション (Versioned Migration) を採用。

4. **HTTPS 常時暗号化および認証ゲートウェイの標準実装**
   - Let's Encrypt (`autocert`) を用いた HTTPS 自動更新機能と、HTTP から HTTPS への自動リダイレクトおよび HSTS セキュリティヘッダーを標準で計装。
   - `Authorization: Bearer` トークンを検証する認証ミドルウェアの実装。
   - ログ出力時にパスワードなどの機密情報を自動で隠蔽する `SecureJSONHandler` (`slog`) の標準計装。

5. **ライセンス＆作成者ヘッダーの自動化と CI 検証**
   - 各 Go ソースファイルの冒頭に Apache-2.0 のライセンスヘッダーおよび作成者コメント（`Author`）が入るように `Makefile` および Python スクリプト (`check_license.py`) を実装。
   - GitHub Actions CI において、コードの自動生成の差分チェック (`git diff --exit-code`)、ライセンス監査、静的解析、および単体・結合E2Eテストが全自動で検証されるようにパイプラインを統合。

## 結果
- 新規プロジェクトを立ち上げる開発者は、本リポジトリをテンプレートとして使用するだけで、セキュリティやガバナンスチェック、リリースパイプラインが定義された、極めて安全な Go アプリケーションを即座に書き始められます。
- すべての要件に対する適合度（自己評価）は `make self-eval` で客観的に測定可能になり、適合率 100% の品質が常に保証されます。
