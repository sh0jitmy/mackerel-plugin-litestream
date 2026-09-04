---
name: golang-e2e-testing
description: "Go (Golang) の単体テスト、結合テスト、およびE2Eテストのプラクティス。テーブル駆動テスト、testify（suite、assert、mock）、並行テスト（t.Parallel()）、goroutineリーク検出（goleak）、結合/E2EテストにおけるDBシード・クリーンアップ、ネットワークモック、およびFlakyテストの防止対策を適用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: [YOUR_NAME]
  version: "1.0.0"
  openclaw:
    emoji: "🧪"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Agent AskUserQuestion
---

> [!IMPORTANT]
> **ガバナンスと変更管理:** 
> 本スキルファイルは組織の標準テスト指針です。人間の明示的な指示がない限り、AIエージェント自身でこのファイルを書き換えないでください。変更は必ず Git のプルリクエストおよびアーキテクトによるレビューを経て行われます。

**Persona:** あなたは品質保証およびテスト設計の専門家です。テストは仕様書であると考え、信頼性が高く、実行速度が早く、かつメンテナンスが容易なテストスイートを設計します。

# Go テスト & E2E テストプラクティス (Go-Specific)

## 1. 単体テスト (Unit Testing)
- **テーブル駆動テスト (Table-Driven Tests)**: 複数のテストケースをスライスで定義してループで実行する。各ケースには必ず `name` フィールドを含め、`t.Run(tc.name, ...)` でサブテスト化する。
- **アサーション**: 標準の `if` 文によるチェックに加え、可読性を高めるために `github.com/stretchr/testify/assert` や `require` を適切に活用する。
- **テストの並行実行**: `t.Parallel()` を呼び出し、テスト実行を高速化する。ただし、共有リソースへの書き込みがある場合は並行実行を避ける。
- **goroutine リーク検出**: `go.uber.org/goleak` を用いて、テスト終了時に終了していない不正な goroutine が残っていないか検査する。
  ```go
  func TestMain(m *testing.M) {
      goleak.VerifyTestMain(m)
  }
  ```

## 2. 結合テストと外部モック (Integration Testing & Mocking)
- **ビルドタグの利用**: 時間のかかる結合テストには `//go:build integration` などのビルドタグを付与し、通常の `go test ./...` から分離する。
- **インターフェースのモック**: 具象型ではなくインターフェースに対してモックを作成する（例: `testify/mock` や `mockgen`）。
- **HTTP モック**: 外部APIを呼び出すテストでは、`net/http/httptest` を用いてモックサーバーを作成し、ネットワークの不確実性を排除する。
  ```go
  server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      w.WriteHeader(http.StatusOK)
      w.Write([]byte(`{"status":"success"}`))
  }))
  defer server.Close()
  ```

## 3. E2E テスト (End-to-End Testing)
- **データベースの隔離とシード**: E2Eテストを実行する際、テスト用データベースが汚染されないように、各テストケースの前後でトランザクションのロールバックを行うか、スキーマをクリーンアップしてシードデータを再挿入する。
- **Playwright-Go の活用**: Web UI が伴うシステムの場合、`github.com/playwright-community/playwright-go` を用いてブラウザ操作を自動化し、実際のユーザーフローをテストする。
- **Flaky テストの防止 (防脆性)**:
  - 実行順序に依存するテストを書かない。
  - テスト内で `time.Sleep` を使用した時間待ちをしない。代わりに、チャンネルのイベント通知の待機や、ポーリングによる状態変化の監視（例: `assert.Eventually`）を行う。
  ```go
  // ✅ 良い例: アサーションによる状態ポーリング待機
  assert.Eventually(t, func() bool {
      return checkDatabaseState() == expectedValue
  }, 5*time.Second, 100*time.Millisecond)
  ```
- **コンテナ（DB / OTel）を用いた結合 E2E テスト**:
  - モックだけでカバーできない SQL の機能（UPSERT/Conflict 制御等）や、実際の OpenTelemetry 連携を確認するため、Docker Compose を用いた PostgreSQL や OTel Collector 等の実コンテナ環境に対する自動検証テストを導入することを推奨する。
  - **オンデマンド CI / ローカル限定実行制御**:
    - コンテナの立ち上げを伴うヘビーな E2E テストは、CI パイプライン全体の実行時間を短縮するため、Go のテストビルドタグ（例: `//go:build integration_e2e`）で制御し、ローカル開発環境での確認時や、マージ前の最終 CI で「必要な時」にのみトリガーできるように設計すること。

## 4. 障害模擬テストおよびフォールトインジェクションの自動化（Go-specific）
開発およびQAは、FMEA/FTAで定義された異常系挙動をコード上で検証するため、以下のプラクティスに基づいて障害注入（Fault Injection）テストを実装します。

- **HTTP 外部API呼び出しの障害注入**:
  - `net/http/httptest` を用いて、特定のレスポンス遅延（Latency Injection）や、HTTP 500/503エラー、不正なJSONレスポンスを返却するモックハンドラーを定義し、アプリケーション側のタイムアウトやリトライ、サーキットブレイカー、およびエラーマスクの挙動を検証します。
  - テストコード例（遅延の注入）:
    ```go
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(2 * time.Second) // タイムアウトを引き起こす遅延を注入
        w.WriteHeader(http.StatusServiceUnavailable)
    }))
    defer server.Close()
    ```
- **データベース接続の障害注入**:
  - `github.com/DATA-DOG/go-sqlmock` を用いて、SQL実行時にエラー（`driver.ErrBadConn` やデッドロック等）を発生させるモックを作成し、トランザクションのロールバックやリトライ、接続プールの再接続挙動が正しく行われることをアサーション検証します。
- **Contextのタイムアウト・キャンセル伝播の検証**:
  - 下流の関数や外部リクエストに `context.Context` が正しく伝播していることを検証するため、テスト内で意図的に短いタイムアウト（例: `context.WithTimeout`）を設定し、メソッドが遅延なく `context.DeadlineExceeded` エラーを返却してリソースを解放するか検証します。
- **エラーマスクと構造化ログの検証**:
  - 障害模擬テストの実行時、クライアントに返却されたエラーメッセージが抽象化（マスク）されていること（例: 「一時的なエラーが発生しました」等）を確認し、内部の具体的なエラーメッセージ（例: 「connection refused」やSQLエラー）が含まれていないことをアサーションします。
  - 同時に、`slog` の出力先バッファやカスタムハンドラーを監視し、詳細なエラーメッセージやコールスタックが内部ログ（Errorレベル）として適切に記録されていることを検証します。

## 5. ビジネスロジックの分離とカバレッジしきい値制約（80%以上）
- **ビジネスロジックの配置**:
  - アプリケーションのビジネスロジック（認証、データ処理、複雑なバリデーション等）は、HTTP層（Gin/context等）やデータベースの自動生成コード（ent/ogen等）と密結合させず、必ず `internal/service/` または `internal/domain/` パッケージとして分離して定義する。
- **テストカバレッジのしきい値（80%以上必須）**:
  - `internal/service/` および `internal/domain/` 配下のすべてのパッケージにおいて、ステートメントカバレッジ（コードカバー率）は **80% 以上を必須**とする。
  - プロジェクトのテストスイート（`make test`）では、自動的にこれらのパッケージのカバレッジが算出され、80% 未満の場合はテスト自体が失敗する（終了コード非ゼロを返す）ガードレールが敷かれている。
  - そのため、ビジネスロジックの変更・追加時は必ず、正常系・異常系・エッジケースを網羅するテストコード（ユニットテスト等）を併せて作成しなければならない。


