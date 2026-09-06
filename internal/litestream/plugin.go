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

package litestream

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

// LitestreamPlugin は Mackerel メトリックプラグインインターフェースを実装します。
type LitestreamPlugin struct {
	Config            Config
	SystemdChecker    *SystemdChecker
	SnapshotCollector *SnapshotCollector
	RestoreChecker    *RestoreChecker
	NowFunc           func() time.Time // テスト用
}

// NewLitestreamPlugin は新しい LitestreamPlugin を生成します。
func NewLitestreamPlugin(cfg Config) *LitestreamPlugin {
	return &LitestreamPlugin{
		Config:            cfg,
		SystemdChecker:    NewSystemdChecker(nil, cfg.Timeout),
		SnapshotCollector: NewSnapshotCollector(nil, cfg.LitestreamBin, cfg.ConfigPath, cfg.SnapshotCacheTTL),
		RestoreChecker:    NewRestoreChecker(nil, cfg.LitestreamBin, cfg.ConfigPath, cfg.RestoreCheckInterval, cfg.RestoreDryRun),
		NowFunc:           time.Now,
	}
}

// MetricKeyPrefix は Mackerel メトリックキーのプレフィックスを返します。
func (p *LitestreamPlugin) MetricKeyPrefix() string {
	if p.Config.MetricKeyPrefix == "" {
		return "litestream"
	}
	return p.Config.MetricKeyPrefix
}

// GraphDefinition は有効化されたメトリクスグループに基づいてグラフ定義を返します。
// 不要なグループを無効化することで、Mackerel のメトリック課金枠の消費を抑制します。
func (p *LitestreamPlugin) GraphDefinition() map[string]mp.Graphs {
	graphs := make(map[string]mp.Graphs)

	// サービス死活状態 (常に定義)
	graphs["status"] = mp.Graphs{
		Label: "Litestream Service Status",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "alive", Label: "Service Alive (1=Up, 0=Down)"},
		},
	}

	// 1. Core (重要エラー & ディスクフル)
	if p.Config.EnableCore {
		graphs["core_errors.#"] = mp.Graphs{
			Label: "Litestream Core Errors",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "sync_error_count", Label: "Sync Errors", Diff: true},
				{Name: "verify_error_count", Label: "Compaction Verification Errors", Diff: true},
				{Name: "disk_full", Label: "Disk Full Staging Wedge (1=Full, 0=OK)"},
			},
		}
	}

	// 2. Storage (DB本体およびWALファイルサイズ)
	if p.Config.EnableStorage {
		graphs["storage.#"] = mp.Graphs{
			Label: "Litestream Storage Size",
			Unit:  "bytes",
			Metrics: []mp.Metrics{
				{Name: "db_bytes", Label: "Database Size"},
				{Name: "wal_bytes", Label: "WAL File Size"},
			},
		}
	}

	// 3. Snapshot (スナップショット経過時間 & 未コンパクションWAL数 & S3統計)
	if p.Config.EnableSnapshot {
		graphs["snapshot_age.#"] = mp.Graphs{
			Label: "Litestream Latest Snapshot Age",
			Unit:  "seconds",
			Metrics: []mp.Metrics{
				{Name: "latest_age_seconds", Label: "Age Since Latest Snapshot"},
			},
		}
		graphs["snapshot_wal.#"] = mp.Graphs{
			Label: "Litestream WAL Files Since Snapshot",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "wal_files_since_snapshot", Label: "Uncompacted WAL (L0) Files"},
				{Name: "total_snapshot_count", Label: "Total Snapshot Count"},
				{Name: "snapshot_missing", Label: "Snapshot Missing Flag (1=Missing, 0=OK)"},
			},
		}
		graphs["remote_storage.#"] = mp.Graphs{
			Label: "Litestream Remote Storage Volume",
			Unit:  "bytes",
			Metrics: []mp.Metrics{
				{Name: "remote_total_bytes", Label: "Total Stored Bytes"},
			},
		}
		graphs["remote_objects.#"] = mp.Graphs{
			Label: "Litestream Remote Storage Objects",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "remote_total_objects", Label: "Total Stored Files/Objects"},
			},
		}
	}

	// 4. Sync Stats (詳細同期パフォーマンス - オプトイン)
	if p.Config.EnableSyncStats {
		graphs["sync.#"] = mp.Graphs{
			Label: "Litestream Replication Sync",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "count", Label: "Sync Operations", Diff: true},
			},
		}
		graphs["sync_latency.#"] = mp.Graphs{
			Label: "Litestream Sync Cumulative Seconds",
			Unit:  "float",
			Metrics: []mp.Metrics{
				{Name: "seconds", Label: "Sync Time Seconds", Diff: true},
			},
		}
	}

	// 5. Checkpoint (チェックポイント処理詳細 - オプトイン)
	if p.Config.EnableCheckpoint {
		graphs["checkpoint.#"] = mp.Graphs{
			Label: "Litestream Checkpoint Count",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "*_count", Label: "Checkpoint Count", Diff: true},
				{Name: "*_error", Label: "Checkpoint Errors", Diff: true},
			},
		}
	}

	// 6. Retention (L0保持状態詳細 - オプトイン)
	if p.Config.EnableRetention {
		graphs["retention.#"] = mp.Graphs{
			Label: "Litestream L0 Retention Files",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "eligible", Label: "Eligible for Deletion"},
				{Name: "not_compacted", Label: "Not Compacted"},
				{Name: "too_recent", Label: "Too Recent"},
			},
		}
	}

	// 7. Replica Ops (リモートストレージ操作詳細 - オプトイン)
	if p.Config.EnableReplicaOps {
		graphs["replica_ops"] = mp.Graphs{
			Label: "Litestream Replica Operations",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "*", Label: "Operations", Diff: true},
			},
		}
		graphs["replica_traffic"] = mp.Graphs{
			Label: "Litestream Replica Traffic",
			Unit:  "bytes",
			Metrics: []mp.Metrics{
				{Name: "*", Label: "Transferred Bytes", Diff: true},
			},
		}
	}

	// 8. Replica Errors (リモートストレージエラー詳細 - オプトイン)
	if p.Config.EnableReplicaErrors {
		graphs["replica_errors"] = mp.Graphs{
			Label: "Litestream Replica Errors",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "*", Label: "Error Count", Diff: true},
			},
		}
	}

	// 9. Restore Check (リストア検証 - オプトイン)
	if p.Config.EnableRestoreCheck {
		graphs["restore.#"] = mp.Graphs{
			Label: "Litestream Replica Restore Check",
			Unit:  "float",
			Metrics: []mp.Metrics{
				{Name: "restore_error", Label: "Restore Error (1=Failed, 0=OK)"},
				{Name: "restore_duration_seconds", Label: "Restore Duration Seconds"},
			},
		}
	}

	return graphs
}

// FetchMetrics は Prometheus エンドポイントおよびスナップショット情報からメトリクスを収集します。
func (p *LitestreamPlugin) FetchMetrics() (map[string]float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.Config.Timeout)
	defer cancel()

	// 1. systemd サービスのステータスチェック（設定されている場合）
	if p.Config.SystemdService != "" && p.SystemdChecker != nil {
		state, err := p.SystemdChecker.CheckStatus(ctx, p.Config.SystemdService)
		if err != nil {
			slog.Warn("failed to check systemd service status", slog.String("service", p.Config.SystemdService), slog.Any("error", err))
		}
		switch state {
		case ServiceStateInactiveDisabled:
			// サービスが意図的に無効化されている場合は、誤アラート防止のためメトリクスを送信せず正常終了
			slog.Info("systemd service is disabled, skipping metric collection", slog.String("service", p.Config.SystemdService))
			return map[string]float64{}, nil
		case ServiceStateInactiveEnabled:
			// サービスが enabled なのに停止している場合は異常状態として alive: 0 を送信
			slog.Warn("systemd service is enabled but inactive/failed", slog.String("service", p.Config.SystemdService))
			return map[string]float64{"alive": 0}, nil
		}
	}

	metrics := make(map[string]float64)

	// 2. Prometheus メトリクスのスクレイピング
	metricFamilies, err := ScrapePrometheusMetrics(ctx, p.Config.URL, p.Config.Timeout)
	if err != nil {
		slog.Warn("failed to scrape prometheus metrics", slog.String("url", p.Config.URL), slog.Any("error", err))
		metrics["alive"] = 0
		return metrics, nil
	}

	metrics["alive"] = 1

	now := time.Now()
	if p.NowFunc != nil {
		now = p.NowFunc()
	}

	// 検出された DB パスのリスト
	dbPaths := make(map[string]string)

	// 3. Prometheus メトリクスのパースと抽出
	for mName, mf := range metricFamilies {
		for _, m := range mf.GetMetric() {
			dbLabel := GetLabelValue(m, "db")
			if p.Config.DBFilter != "" && dbLabel != p.Config.DBFilter {
				continue
			}
			sanitizedDB := SanitizeDBKey(dbLabel)
			if dbLabel != "" {
				dbPaths[dbLabel] = sanitizedDB
			}

			val := GetMetricValue(m)

			switch mName {
			case "litestream_sync_error_count":
				if p.Config.EnableCore {
					metrics[fmt.Sprintf("core_errors.%s.sync_error_count", sanitizedDB)] = val
				}
			case "litestream_compaction_verify_error_count":
				if p.Config.EnableCore {
					metrics[fmt.Sprintf("core_errors.%s.verify_error_count", sanitizedDB)] = val
				}
			case "litestream_disk_full":
				if p.Config.EnableCore {
					metrics[fmt.Sprintf("core_errors.%s.disk_full", sanitizedDB)] = val
				}

			case "litestream_db_size":
				if p.Config.EnableStorage {
					metrics[fmt.Sprintf("storage.%s.db_bytes", sanitizedDB)] = val
				}
			case "litestream_wal_size":
				if p.Config.EnableStorage {
					metrics[fmt.Sprintf("storage.%s.wal_bytes", sanitizedDB)] = val
				}

			case "litestream_sync_count":
				if p.Config.EnableSyncStats {
					metrics[fmt.Sprintf("sync.%s.count", sanitizedDB)] = val
				}
			case "litestream_sync_seconds":
				if p.Config.EnableSyncStats {
					metrics[fmt.Sprintf("sync_latency.%s.seconds", sanitizedDB)] = val
				}

			case "litestream_checkpoint_count":
				if p.Config.EnableCheckpoint {
					mode := SanitizeKey(GetLabelValue(m, "mode"))
					metrics[fmt.Sprintf("checkpoint.%s.%s_count", sanitizedDB, mode)] = val
				}
			case "litestream_checkpoint_error_count":
				if p.Config.EnableCheckpoint {
					mode := SanitizeKey(GetLabelValue(m, "mode"))
					metrics[fmt.Sprintf("checkpoint.%s.%s_error", sanitizedDB, mode)] = val
				}

			case "litestream_l0_retention_files_total":
				if p.Config.EnableRetention {
					status := SanitizeKey(GetLabelValue(m, "status"))
					metrics[fmt.Sprintf("retention.%s.%s", sanitizedDB, status)] = val
				}

			case "litestream_replica_operation_total":
				if p.Config.EnableReplicaOps {
					repType := SanitizeKey(GetLabelValue(m, "replica_type"))
					op := SanitizeKey(GetLabelValue(m, "operation"))
					metrics[fmt.Sprintf("replica_ops.%s_%s", repType, op)] = val
				}
			case "litestream_replica_operation_bytes":
				if p.Config.EnableReplicaOps {
					repType := SanitizeKey(GetLabelValue(m, "replica_type"))
					op := SanitizeKey(GetLabelValue(m, "operation"))
					metrics[fmt.Sprintf("replica_traffic.%s_%s", repType, op)] = val
				}

			case "litestream_replica_operation_errors_total":
				if p.Config.EnableReplicaErrors {
					repType := SanitizeKey(GetLabelValue(m, "replica_type"))
					op := SanitizeKey(GetLabelValue(m, "operation"))
					code := SanitizeKey(GetLabelValue(m, "code"))
					metrics[fmt.Sprintf("replica_errors.%s_%s_%s", repType, op, code)] = val
				}
			}
		}
	}

	// 対象データベース一覧の確定
	targets := make(map[string]string)
	if p.Config.DBFilter != "" {
		targets[p.Config.DBFilter] = SanitizeDBKey(p.Config.DBFilter)
	} else if len(dbPaths) > 0 {
		targets = dbPaths
	} else {
		targets[""] = "default"
	}

	// 4. スナップショット / S3 統計の収集
	if p.Config.EnableSnapshot && p.SnapshotCollector != nil {
		for rawDB, sName := range targets {
			stats, err := p.SnapshotCollector.Collect(ctx, rawDB, now)
			if err != nil {
				slog.Debug("failed to collect snapshot stats", slog.String("db", rawDB), slog.Any("error", err))
				metrics[fmt.Sprintf("snapshot_wal.%s.snapshot_missing", sName)] = 1
				continue
			}

			metrics[fmt.Sprintf("snapshot_age.%s.latest_age_seconds", sName)] = stats.LatestSnapshotAgeSeconds
			metrics[fmt.Sprintf("snapshot_wal.%s.wal_files_since_snapshot", sName)] = stats.WALFilesSinceSnapshot
			metrics[fmt.Sprintf("snapshot_wal.%s.total_snapshot_count", sName)] = stats.TotalSnapshotCount
			metrics[fmt.Sprintf("snapshot_wal.%s.snapshot_missing", sName)] = stats.SnapshotMissing
			metrics[fmt.Sprintf("remote_storage.%s.remote_total_bytes", sName)] = stats.RemoteTotalBytes
			metrics[fmt.Sprintf("remote_objects.%s.remote_total_objects", sName)] = stats.RemoteTotalObjects
		}
	}

	// 5. リストア検証（冗長化からの復元確認）の実行
	if p.Config.EnableRestoreCheck && p.RestoreChecker != nil {
		for rawDB, sName := range targets {
			rStats, err := p.RestoreChecker.Check(ctx, rawDB, now)
			var errVal float64
			if err != nil || !rStats.Success {
				errVal = 1
			}
			metrics[fmt.Sprintf("restore.%s.restore_error", sName)] = errVal
			metrics[fmt.Sprintf("restore.%s.restore_duration_seconds", sName)] = rStats.DurationSeconds
		}
	}

	return metrics, nil
}
