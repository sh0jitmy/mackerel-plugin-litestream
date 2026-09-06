# mackerel-plugin-litestream

[![CI](https://github.com/sh0jitmy/mackerel-plugin-litestream/actions/workflows/ci.yml/badge.svg)](https://github.com/sh0jitmy/mackerel-plugin-litestream/actions/workflows/ci.yml)
[![GitHub release](https://img.shields.io/github/v/release/sh0jitmy/mackerel-plugin-litestream)](https://github.com/sh0jitmy/mackerel-plugin-litestream/releases)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

**mackerel-plugin-litestream** は、SQLite のリアルタイムレプリケーションツール [Litestream](https://litestream.io/) の健全性、レプリケーション状態、およびバックアップ信頼性を可視化・監視するための Mackerel メトリックプラグインです。

Litestream は SQLite データベースのデータをクラウドストレージ（Amazon S3, Google Cloud Storage, Azure Blob Storage 等）へ継続的に冗長化する極めて重要なサービスです。本プラグインは、レプリケーション停止、WAL肥大化、データ整合性エラー、スナップショット作成遅延、およびリストア時間を悪化させる未コンパクション WAL の滞留を早期に検知します。

---

## 🚀 主な特徴

1. **メトリクス課金コストの最小化（トグル設計）**:
   - Mackerel のホストメトリック枠（30〜50メトリック）を無駄に消費しないよう、メトリクスグループごとにフラグで ON/OFF を切り替え可能。
   - デフォルト（標準モード）では、**1DBあたり約8〜9メトリック**の必要最小限の重要指標に厳選されています。
2. **systemd / systemctl とのスマートな連携**:
   - `--systemd-service` オプションにより、サービスの稼働状態を自動判定。
   - メンテナンス等でサービスが `disabled`（無効化）されている場合は、アラート誤爆を防ぐためメトリクス出力を自動スキップします。
3. **S3 / クラウドストレージ側のリアルタイムスナップショット・WAL滞留監視**:
   - CloudWatch の S3 メトリクスが持つ「24時間遅延」の課題を解消。
   - 最新スナップショットの経過時間（鮮度）、最新スナップショット以降に溜まった未コンパクション WAL 数、リモート総容量・オブジェクト数をリアルタイムに取得。
4. **S3 API コスト最適化（TTL キャッシュ機構）**:
   - リモートストレージのリスト処理結果を 5分間（変更可能）ローカルキャッシュし、S3 の ListObjects API コストを大幅に削減。
5. **既存プラグインとのベストプラクティス連携**:
   - プロセスの死活監視には `check-systemd`、ログエラーには `check-log` を組み合わせることで、**メトリック枠消費ゼロ**で多層防御を実現（詳細は [監視運用ガイド](docs/monitoring_guide.md) を参照）。

---

## 📦 インストール

### 方法 1: mkr コマンドでインストール (推奨)
Mackerel 公式 CLI ツール `mkr` がインストールされている環境であれば、1 コマンドで最新バイナリを自動インストール可能です。

```bash
sudo mkr plugin install sh0jitmy/mackerel-plugin-litestream
```
※ `/opt/mackerel-agent/plugins/bin/mackerel-plugin-litestream` に自動配置されます。

### 方法 2: GitHub Releases からバイナリをダウンロード
[Releases ページ](https://github.com/sh0jitmy/mackerel-plugin-litestream/releases) からお使いの OS・アーキテクチャに合った zip アーカイブをダウンロード・解凍し、`/usr/local/bin` 等の PATH の通ったディレクトリに配置します。

```bash
# 例: Linux amd64 の場合
curl -fsSL -O https://github.com/sh0jitmy/mackerel-plugin-litestream/releases/latest/download/mackerel-plugin-litestream_linux_amd64.zip
unzip mackerel-plugin-litestream_linux_amd64.zip
sudo mv mackerel-plugin-litestream /usr/local/bin/
sudo chmod +x /usr/local/bin/mackerel-plugin-litestream
```

### 方法 3: Go コマンドでインストール
```bash
go install github.com/shjtmy/mackerel-plugin-litestream@latest
```

---

## ⚙️ クイックスタート

### 1. Litestream 側のメトリクス有効化
Litestream の設定ファイル (`/etc/litestream.yml`) に `addr` を設定し、Prometheus メトリクスエンドポイントを有効化します。

```yaml
# /etc/litestream.yml
addr: ":9090"

dbs:
  - path: /var/lib/myapp.db
    replicas:
      - url: s3://my-litestream-bucket/myapp
```

Litestream を再起動し、メトリクスが取得できることを確認します：
```bash
curl http://localhost:9090/metrics
```

### 2. Mackerel Agent の設定
`/etc/mackerel-agent/mackerel-agent.conf` にプラグイン設定を追加します。

```ini
[plugin.metrics.litestream]
command = [
    "/usr/local/bin/mackerel-plugin-litestream",
    "-url", "http://localhost:9090/metrics",
    "-config", "/etc/litestream.yml",
    "-systemd-service", "litestream.service"
]
```

エージェントをリロードすると、Mackerel コンソールにグラフが自動登録されます：
```bash
sudo systemctl reload mackerel-agent
```

---

## 📊 収集メトリクス一覧

### デフォルトで収集されるメトリクス（標準モード: 1DBあたり約8〜9点）

| グラフキー | メトリック名 | 種別 | 説明 |
| :--- | :--- | :---: | :--- |
| `litestream.status` | `alive` | Gauge | プラグインがメトリクスエンドポイントに接続可能か (1=正常, 0=停止) |
| `litestream.core_errors.#` | `sync_error_count` | Diff | WAL同期エラー発生回数/分 (★重要) |
| | `verify_error_count` | Diff | コンパクション後の整合性検証失敗数/分 (★データ破損検知) |
| | `disk_full` | Gauge | ステージング書き込み時のローカルディスクフル停止状態 (1=停止, 0=正常) |
| `litestream.storage.#` | `db_bytes` | Gauge | SQLiteデータベース本体のファイルサイズ (Bytes) |
| | `wal_bytes` | Gauge | WALファイルの現在サイズ (Bytes, 肥大化検知) |
| `litestream.snapshot_age.#` | `latest_age_seconds`| Gauge | 最新スナップショット (Level 9) からの経過時間 (秒) |
| `litestream.snapshot_wal.#` | `wal_files_since_snapshot` | Gauge | 最新スナップショット以降に溜まっている未コンパクション WAL 数 |
| | `total_snapshot_count`| Gauge | リモートストレージ上の有効なスナップショット総数 |
| | `snapshot_missing` | Gauge | **スナップショット不在・消失フラグ** (1=消失/不在, 0=正常) |
| `litestream.remote_storage.#` | `remote_total_bytes` | Gauge | リモートストレージ上の全ファイル合計サイズ (Bytes) |
| `litestream.remote_objects.#` | `remote_total_objects`| Gauge | リモートストレージ上の総ファイル数 |

※ `#` にはデータベース名（例: `myapp_db`）が入ります。

### オプトインで有効化できる詳細メトリクス（フラグ指定）

| フラグ | 対象メトリクス | 用途 |
| :--- | :--- | :--- |
| `-enable-restore-check` | `restore.#.restore_error`, `restore_duration_seconds` | **冗長化時の復元可能性検証** (1=エラー, 0=正常) およびリストア所要時間 |
| `-enable-sync-stats` | `sync.#.count`, `sync_latency.#.seconds` | 同期処理の実行回数および所要時間（レイテンシ） |
| `-enable-checkpoint` | `checkpoint.#.<mode>_count`, `*_error` | SQLiteチェックポイントのモード別実行回数・エラー |
| `-enable-retention` | `retention.#.eligible`, `not_compacted`, `too_recent` | L0リテンションポリシー対象ファイルの内訳 |
| `-enable-replica-ops` | `replica_ops.*`, `replica_traffic.*` | S3等の操作回数 (PUT/GET/DELETE) および転送バイト数 |
| `-enable-replica-errors`| `replica_errors.<type>_<op>_<code>` | S3等のエラーコード別 (`AccessDenied`, `SlowDown` 等) エラー数 |

---

## 🛠️ コマンドラインオプション

```text
Usage: mackerel-plugin-litestream [options]

接続・基本設定:
  -url string
        Litestream Prometheus metrics endpoint URL (default "http://localhost:9090/metrics")
  -metric-key-prefix string
        Metric key prefix (default "litestream")
  -tempfile string
        Path to temp file for diff calculations
  -timeout duration
        HTTP and command execution timeout (default 5s)
  -db string
        Filter for a specific database path or name (optional)
  -config string
        Path to Litestream config file (default "/etc/litestream.yml")
  -litestream-bin string
        Path to litestream binary (default "litestream")
  -systemd-service string
        Systemd service name to check status (e.g. litestream.service)
  -snapshot-cache-ttl duration
        Cache TTL for remote snapshot/LTX listing (default 5m0s)

リストア検証設定:
  -enable-restore-check
        Enable restore verification metrics (default false)
  -restore-check-interval duration
        Cache interval / TTL for restore verification (default 30m0s)
  -restore-dry-run
        Use dry-run for restore verification to avoid writing files (default true)
  -check-restore
        Run in Mackerel check plugin mode for restore verification (exit code 0=OK, 2=CRITICAL)

メトリクス課金抑制（トグル）:
  -enable-core
        Enable core error metrics (sync errors, verify errors, disk full) (default true)
  -enable-storage
        Enable database and WAL file size metrics (default true)
  -enable-snapshot
        Enable snapshot age, uncompacted WAL count, and S3 stats (default true)
  -enable-sync-stats
        Enable sync operations count and duration metrics (default false)
  -enable-checkpoint
        Enable checkpoint count and error metrics (default false)
  -enable-retention
        Enable L0 retention file status metrics (default false)
  -enable-replica-ops
        Enable replica storage operations and traffic metrics (default false)
  -enable-replica-errors
        Enable replica error code metrics (default false)

その他:
  -print-graph-defs
        Print graph definitions (Mackerel plugin specification)
  -version
        Show version and exit
```

---

## 📖 関連ドキュメント
- [Litestream 監視設計・運用ガイド (docs/monitoring_guide.md)](docs/monitoring_guide.md):
  - 既存プラグイン（`check-systemd`, `check-log`, `mackerel-plugin-df`）との詳細な役割分担
  - 運用推奨アラートルール集（Critical / Warning の具体的な閾値とインシデント対応手順）
- [動作検証・テスト実施報告書 (docs/test_report.md)](docs/test_report.md):
  - Docker 実環境（MinIO + Litestream + SQLite）での統合テスト結果
  - 標準モード vs フル観測モードの生出力ログ比較
  - 障害検知および差分計算の検証結果

---

## 📜 ライセンス
Apache License 2.0
