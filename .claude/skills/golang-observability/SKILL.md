---
name: golang-observability
description: "Go (Golang) のオブザーバビリティ (o11y) プラクティス。log/slogによる構造化ログ、Prometheusによるメトリクス（RED/USEメソッド、ヒストグラム、低カーディナリティ）、OpenTelemetryによる分散トレース、pprof/Pyroscopeによるプロファイリングを適用する際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: [YOUR_NAME]
  version: "1.1.0"
  openclaw:
    emoji: "📡"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Agent
---

> [!IMPORTANT]
> **ガバナンスと変更管理:** 
> 本スキルファイルは組織の標準運用観測指針です。人間の明示的な指示がない限り、AIエージェント自身でこのファイルを書き換えないでください。変更は必ず Git のプルリクエストおよびアーキテクトによるレビューを経て行われます。

**Persona:** あなたは Go のオブザーバビリティエンジニアです。「観測できない本番システムはリスクである」と考え、ログ・メトリクス・トレース・プロファイルを適切に紐付けて問題の早期解決を可能にします。

# Go オブザーバビリティ (Observability) ベストプラクティス (Go-Specific)

## 1. 構造化ログ (Structured Logging with slog)
- **`log/slog` の使用**: 本番環境のログは JSON などの構造化フォーマットで出力する。
- **コンテキスト付きログ**: `slog.InfoContext(ctx, ...)` を用い、Context経由でログにトレースIDなどのメタデータを自動付与できるようにする。
  ```go
  // ✅ 良い例: コンテキストを明示的に渡す
  slog.InfoContext(ctx, "processing payment", slog.String("payment_id", payID))
  ```
- **適切なログレベル**: Debug, Info, Warn, Error を適切に使い分ける。

- **DB・デッドロックの構造化ログ規約 (transaction-profiling-analyzerとの連携)**:
  - データベース操作でエラー、デッドロック、スロークエリ（例: 200ms以上）を検知した際は、以下の属性を slog のキー値として構造化します。
    - `db.query`: 実行されたSQL文（機密情報はプレースホルダーにマスクされていること）。
    - `db.duration_ms`: クエリまたはトランザクションの実行時間（ミリ秒単位）。
    - `db.error_code`: PostgreSQLのエラーコード（例: デッドロックは `40001` 等）。
    - `db.lock_conflict`: 排他制御競合が発生した情報。

---

## 2. メトリクス (Prometheus client_golang)
- **Prometheus SDK の利用**: `github.com/prometheus/client_golang` を用いて、リクエスト数や処理時間を計測する。
- **ヒストグラムの優先使用**:
  - 処理時間の計測には `Summary` ではなく `Histogram` を使用し、PromQL で `histogram_quantile()` を用いてP95/P99などのパーセンタイルを計算可能にする。
- **低カーディナリティの徹底**:
  - ユーザーIDや未加工のURLなど、値の種類が無限に増えるラベルを Prometheus メトリクスに設定しない。メモリ破綻を招く。
- **PromQL コメント規約**:
  - コード内でメトリクス変数を定義する際、その直前のコメントとして「代表的な監視クエリ（PromQL）」を記述する。
  ```go
  // HTTP リクエスト処理遅延ヒストグラム
  // PromQL: histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, path))
  var httpRequestDuration = prometheus.NewHistogramVec(...)
  ```

---

## 3. 分散トレース (OpenTelemetry Go SDK)
- **OpenTelemetry の導入**: `go.opentelemetry.io/otel` を用いて、ボトルネックを可視化するため、Spanを定義する。
  ```go
  ctx, span := otel.Tracer("my-service").Start(ctx, "ProcessPayment")
  defer span.End()
  ```
- **Context 伝搬**: HTTPリクエストやメッセージキューの境界を越えてトレースContextを正しく伝搬させる。

- **トランザクション・クエリの計装ルール (transaction-profiling-analyzerとの連携)**:
  - データベースのクエリおよびトランザクション全体のボトルネックを `transaction-profiling-analyzer` で自動プロファイリング可能にするため、以下の標準的なセマンティック属性をスパンに付与します。
    - `db.system` (例: `"postgresql"`)
    - `db.name` (データベース名)
    - `db.statement` (実行されたSQL文)
    - `db.operation` (例: `"SELECT"`, `"UPDATE"`, `"TX_COMMIT"`)
  - トランザクション全体の生存時間を計測するため、トランザクション開始（`db.Begin`）からコミット/ロールバック終了までのライフサイクルをカバーする親スパン（例: `Database Transaction`）を計装します。

---

## 4. プロファイリング (pprof / Pyroscope)
- **pprof の安全な有効化**:
  - 本番環境での CPU やメモリのホットスポット分析のため、`net/http/pprof` を有効化できるようにする。
  - **セキュリティ保護**: エンドポイント（`/debug/pprof`）は外部に公開せず、認証や内部NWからのアクセス制限で保護する。
    ```go
    // ❌ 危険: デフォルトの http.ListenAndServe を用いると全ポートに公開される
    go func() {
        http.ListenAndServe(":6060", nil) // 誰でもアクセス可能
    }()

    // ✅ 安全: 内部用のプライベートなMuxを作成し、認証を掛けるかlocalhostのみでリスンする
    mux := http.NewServeMux()
    mux.HandleFunc("/debug/pprof/", pprof.Index)
    mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
    mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
    mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
    mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

    server := &http.Server{
        Addr:    "127.0.0.1:6060", // localhost にバインド
        Handler: mux,
    }
    go server.ListenAndServe()
    ```

- **mutex / block プロファイルの有効化 (transaction-profiling-analyzerとの連携)**:
  - ロック競合や待機（ブロッキング）状態のボトルネックをプロファイル分析可能にするため、パフォーマンスに著しい影響を与えない適切なサンプリングレートを設定して、mutex/block プロファイルを安全に有効化します。
    ```go
    // 10分の1の確率でサンプリング
    runtime.SetMutexProfileFraction(10)
    // 10,000ナノ秒以上のブロックをサンプリング
    runtime.SetBlockProfileRate(10000)
    ```

---

## 5. ログの共通化および分離戦略 (OpenTelemetry + Fluent Bit)
- **アプリケーションログと監査ログの棲み分け**:
  - **アプリケーションログ**: システムの動作状況やエラー、デバッグ用の情報。標準の `log/slog` を使用し、JSON形式で `stdout`（標準出力）にログを出力します。
  - **監査ログ**: ログイン成功/失敗、ユーザー権限の変更、重要なデータの作成/更新/削除など、「誰が」「いつ」「何を」したかを記録するセキュリティ重視のログ。
  - **実装方針 (以下のいずれかを選択)**:
    1. **ログタイプ属性による分離 (推奨)**: コードをシンプルに保つため、すべてのログは標準出力に出力し、監査ログには `log_type: "audit"` 属性を付与します。
       ```go
       slog.InfoContext(ctx, "user password changed", 
           "log_type", "audit",
           "target_user_id", targetUserID,
           "actor_user_id", actorUserID,
       )
       ```
    2. **専用ロガーの作成**: 出力先のファイルを直接分けたい場合などは、監査ログ専用のロガーインスタンス（`auditLogger`）を生成します。
       ```go
       var auditLogger *slog.Logger
       
       func initAuditLogger(auditLogFile *os.File) {
           auditHandler := slog.NewJSONHandler(auditLogFile, &slog.HandlerOptions{
               Level: slog.LevelInfo,
           })
           auditLogger = slog.New(auditHandler)
       }
       ```
- **OpenTelemetry によるコンテキスト紐付け**:
  - ログと分散トレースを紐付けるため、`go.opentelemetry.io/contrib/bridges/otelslog` ブリッジを利用し、ログレコードに `trace_id` や `span_id` を自動挿入します。
  - ログを出力する際は、コンテキスト情報を伝播させるために必ず `*Context` 関数（例: `slog.InfoContext`）を使用してください。
- **Fluent Bit によるログの収集とルーティング (インフラ・NE連携)**:
  - コンテナ環境では、サイドカーまたは DaemonSet として配置された Fluent Bit が `stdout` からJSONログを回収します。
  - Fluent Bit はログをパースして正規化し、Kubernetesのポッドメタデータや環境名を付与します。
  - **ルーティング**:
    - `log_type == "audit"` のログ（監査ログ）は、長期保存と厳格なアクセス制御が施された専用のセキュリティ保管ストレージへルーティングします。
    - それ以外の通常のアプリケーションログは、Loki や Datadog などの一般的なログビューアへ転送します。
- **テスト環境における OTel 連携検証**:
  - 計装した OTel メトリクスおよびトレース（スパン）が正しく送信されているかを保証するため、E2Eテスト環境（ローカル/CI）において一時的に OTel Collector コンテナ等を立ち上げ、HTTP エンドポイント等を通じて受信データの整合性を検証するテストコードを実装することを推奨する。

---

## 6. PDM向け SLO/SLA および性能戦略 (Product Performance Strategy)
- **指標の定義 (SLI / SLO / SLA)**:
  - **SLI (Service Level Indicator)**: サービスレベルの具体的な測定値（例: 「成功したリクエストの割合」「HTTPのP95応答時間」）。
  - **SLO (Service Level Objective)**: 目標とする品質のターゲット（例: 「月間の可用性が 99.9% 以上」「P95レイテンシが 200ms 以下」）。
  - **SLA (Service Level Agreement)**: 顧客に対して保証する品質の契約値。未達時は補償が発生します（例: 「月間の可用性が 99.0% 未満の場合に返金」）。
  - **安全マージンの確保**: システム障害時のリスクを最小限に抑えるため、**「SLA（外部契約）」と「SLO（内部目標）」の間には必ず十分な安全マージン（セーフティバッファ）を設けます**（例: 99.0% SLA に対し、SLO を 99.9% に設定）。
- **エラー予算 (Error Budget) の活用**:
  - SLOを定義することで、エラー予算（100% - SLO%）が生まれます（例: 可用性SLOが99.9%の場合、エラー予算は0.1%）。
  - **開発速度とのトレードオフ**:
    - エラー予算が十分に**残っている**間は、新機能のリリースや実験的な変更を迅速に進めます。
    - エラー予算を**使い果たした**場合、新機能 of リリースを一時凍結し、信頼性の向上やパフォーマンスチューニング、バグ修正にリソースを全振りします。
- **性能目標を守るためのアーキテクチャ戦略**:
  - **スロットリング/レートリミット**: スパイクアクセスがシステム全体の可用性（SLO）を損なわないよう、レートリミッターを標準装備します。
  - **キャッシュ戦略**: データベースの負荷を下げ、P99レイテンシSLOを維持するために、Redisやインメモリキャッシュを適切に配置します。
  - **非同期・バッチ処理**: 重い処理は同期処理から切り離し、メッセージキュー（RabbitMQ, Kafka等）を用いて非同期化し、ユーザーレスポンス（レイテンシSLO）への影響を最小化します。
  - **性能リグレッション検知**: 開発段階でパフォーマンス低下を検知するため、CI/CDパイプラインにベンチマークテストや自動負荷テストを組み込み、レイテンシが大幅に悪化したコードは本番デプロイ前にブロックします。
