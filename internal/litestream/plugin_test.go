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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const samplePrometheusOutput = `
# HELP litestream_sync_error_count Sync error count
# TYPE litestream_sync_error_count counter
litestream_sync_error_count{db="/var/lib/myapp.db"} 1
# HELP litestream_compaction_verify_error_count Compaction verify error count
# TYPE litestream_compaction_verify_error_count counter
litestream_compaction_verify_error_count{db="/var/lib/myapp.db"} 0
# HELP litestream_disk_full Disk full
# TYPE litestream_disk_full gauge
litestream_disk_full{db="/var/lib/myapp.db"} 0

# HELP litestream_db_size DB size
# TYPE litestream_db_size gauge
litestream_db_size{db="/var/lib/myapp.db"} 20971520
# HELP litestream_wal_size WAL size
# TYPE litestream_wal_size gauge
litestream_wal_size{db="/var/lib/myapp.db"} 65536

# HELP litestream_sync_count Sync count
# TYPE litestream_sync_count counter
litestream_sync_count{db="/var/lib/myapp.db"} 42
# HELP litestream_sync_seconds Sync duration
# TYPE litestream_sync_seconds counter
litestream_sync_seconds{db="/var/lib/myapp.db"} 1.5

# HELP litestream_checkpoint_count Checkpoint count
# TYPE litestream_checkpoint_count counter
litestream_checkpoint_count{db="/var/lib/myapp.db",mode="PASSIVE"} 10
# HELP litestream_checkpoint_error_count Checkpoint error count
# TYPE litestream_checkpoint_error_count counter
litestream_checkpoint_error_count{db="/var/lib/myapp.db",mode="PASSIVE"} 0

# HELP litestream_l0_retention_files_total L0 retention files
# TYPE litestream_l0_retention_files_total gauge
litestream_l0_retention_files_total{db="/var/lib/myapp.db",status="eligible"} 3

# HELP litestream_replica_operation_total Replica ops
# TYPE litestream_replica_operation_total counter
litestream_replica_operation_total{operation="PUT",replica_type="s3"} 50
# HELP litestream_replica_operation_bytes Replica bytes
# TYPE litestream_replica_operation_bytes counter
litestream_replica_operation_bytes{operation="PUT",replica_type="s3"} 1048576
# HELP litestream_replica_operation_errors_total Replica errors
# TYPE litestream_replica_operation_errors_total counter
litestream_replica_operation_errors_total{code="AccessDenied",operation="PUT",replica_type="s3"} 1
`

func TestLitestreamPlugin_GraphDefinition(t *testing.T) {
	t.Parallel()

	t.Run("default config enables core, storage, and snapshot", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		p := NewLitestreamPlugin(cfg)
		graphs := p.GraphDefinition()

		assert.Contains(t, graphs, "status")
		assert.Contains(t, graphs, "core_errors.#")
		assert.Contains(t, graphs, "storage.#")
		assert.Contains(t, graphs, "snapshot_age.#")
		assert.Contains(t, graphs, "snapshot_wal.#")
		assert.Contains(t, graphs, "remote_storage.#")
		assert.Contains(t, graphs, "remote_objects.#")

		// オプトイン項目は含まれない
		assert.NotContains(t, graphs, "sync.#")
		assert.NotContains(t, graphs, "checkpoint.#")
		assert.NotContains(t, graphs, "retention.#")
		assert.NotContains(t, graphs, "replica_ops")
		assert.NotContains(t, graphs, "replica_errors")
	})

	t.Run("opt-in config enables additional graphs", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.EnableSyncStats = true
		cfg.EnableCheckpoint = true
		cfg.EnableRetention = true
		cfg.EnableReplicaOps = true
		cfg.EnableReplicaErrors = true

		p := NewLitestreamPlugin(cfg)
		graphs := p.GraphDefinition()

		assert.Contains(t, graphs, "sync.#")
		assert.Contains(t, graphs, "checkpoint.#")
		assert.Contains(t, graphs, "retention.#")
		assert.Contains(t, graphs, "replica_ops")
		assert.Contains(t, graphs, "replica_errors")
	})
}

func TestLitestreamPlugin_FetchMetrics(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(samplePrometheusOutput))
	}))
	t.Cleanup(func() {
		server.Close()
	})

	mockSnapshotExec := func(ctx context.Context, name string, args ...string) (string, error) {
		return `[
			{"level": 9, "min_txid": "1", "max_txid": "10", "size": 2000000, "timestamp": "2026-09-05T10:00:00Z"},
			{"level": 0, "min_txid": "11", "max_txid": "12", "size": 4096, "timestamp": "2026-09-05T11:00:00Z"}
		]`, nil
	}

	t.Run("fetch metrics with default config", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.URL = server.URL

		p := NewLitestreamPlugin(cfg)
		p.NowFunc = func() time.Time { return now }
		p.SnapshotCollector.Executor = mockSnapshotExec
		p.SnapshotCollector.CacheDir = t.TempDir()

		metrics, err := p.FetchMetrics()
		require.NoError(t, err)

		// alive
		assert.InDelta(t, float64(1), metrics["alive"], 0.001)

		// Core
		assert.InDelta(t, float64(1), metrics["core_errors.myapp_db.sync_error_count"], 0.001)
		assert.InDelta(t, float64(0), metrics["core_errors.myapp_db.verify_error_count"], 0.001)
		assert.InDelta(t, float64(0), metrics["core_errors.myapp_db.disk_full"], 0.001)

		// Storage
		assert.InDelta(t, float64(20971520), metrics["storage.myapp_db.db_bytes"], 0.001)
		assert.InDelta(t, float64(65536), metrics["storage.myapp_db.wal_bytes"], 0.001)

		// Snapshot
		assert.InDelta(t, float64(7200), metrics["snapshot_age.myapp_db.latest_age_seconds"], 0.001)
		assert.InDelta(t, float64(1), metrics["snapshot_wal.myapp_db.wal_files_since_snapshot"], 0.001)
		assert.InDelta(t, float64(1), metrics["snapshot_wal.myapp_db.total_snapshot_count"], 0.001)
		assert.InDelta(t, float64(2004096), metrics["remote_storage.myapp_db.remote_total_bytes"], 0.001)
		assert.InDelta(t, float64(2), metrics["remote_objects.myapp_db.remote_total_objects"], 0.001)

		// オプトインメトリクスは含まれない
		assert.NotContains(t, metrics, "sync.myapp_db.count")
		assert.NotContains(t, metrics, "checkpoint.myapp_db.PASSIVE_count")
		assert.NotContains(t, metrics, "retention.myapp_db.eligible")
		assert.NotContains(t, metrics, "replica_ops.s3_PUT")
	})

	t.Run("fetch metrics with all enabled", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.URL = server.URL
		cfg.EnableSyncStats = true
		cfg.EnableCheckpoint = true
		cfg.EnableRetention = true
		cfg.EnableReplicaOps = true
		cfg.EnableReplicaErrors = true

		p := NewLitestreamPlugin(cfg)
		p.NowFunc = func() time.Time { return now }
		p.SnapshotCollector.Executor = mockSnapshotExec
		p.SnapshotCollector.CacheDir = t.TempDir()

		metrics, err := p.FetchMetrics()
		require.NoError(t, err)

		assert.InDelta(t, float64(42), metrics["sync.myapp_db.count"], 0.001)
		assert.InDelta(t, float64(1.5), metrics["sync_latency.myapp_db.seconds"], 0.001)
		assert.InDelta(t, float64(10), metrics["checkpoint.myapp_db.PASSIVE_count"], 0.001)
		assert.InDelta(t, float64(3), metrics["retention.myapp_db.eligible"], 0.001)
		assert.InDelta(t, float64(50), metrics["replica_ops.s3_PUT"], 0.001)
		assert.InDelta(t, float64(1048576), metrics["replica_traffic.s3_PUT"], 0.001)
		assert.InDelta(t, float64(1), metrics["replica_errors.s3_PUT_AccessDenied"], 0.001)
	})

	t.Run("systemd disabled skips metrics collection", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.URL = server.URL
		cfg.SystemdService = "litestream.service"

		p := NewLitestreamPlugin(cfg)
		p.SystemdChecker.Executor = func(ctx context.Context, name string, args ...string) (string, error) {
			if len(args) >= 2 && args[0] == "is-active" {
				return "inactive", nil
			}
			if len(args) >= 2 && args[0] == "is-enabled" {
				return "disabled", nil
			}
			return "", nil
		}

		metrics, err := p.FetchMetrics()
		require.NoError(t, err)
		assert.Empty(t, metrics) // 空マップで正常終了
	})

	t.Run("systemd enabled but inactive returns alive: 0", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.URL = server.URL
		cfg.SystemdService = "litestream.service"

		p := NewLitestreamPlugin(cfg)
		p.SystemdChecker.Executor = func(ctx context.Context, name string, args ...string) (string, error) {
			if len(args) >= 2 && args[0] == "is-active" {
				return "inactive", nil
			}
			if len(args) >= 2 && args[0] == "is-enabled" {
				return "enabled", nil
			}
			return "", nil
		}

		metrics, err := p.FetchMetrics()
		require.NoError(t, err)
		assert.InDelta(t, float64(0), metrics["alive"], 0.001)
		assert.Len(t, metrics, 1)
	})

	t.Run("metrics endpoint failure returns alive: 0", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.URL = "http://127.0.0.1:59999/non-existent"
		cfg.Timeout = 100 * time.Millisecond

		p := NewLitestreamPlugin(cfg)
		metrics, err := p.FetchMetrics()
		require.NoError(t, err)
		assert.InDelta(t, float64(0), metrics["alive"], 0.001)
	})

	t.Run("restore check enabled outputs restore metrics", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.URL = server.URL
		cfg.EnableRestoreCheck = true

		p := NewLitestreamPlugin(cfg)
		p.NowFunc = func() time.Time { return now }
		p.SnapshotCollector.Executor = mockSnapshotExec
		p.SnapshotCollector.CacheDir = t.TempDir()
		p.RestoreChecker.Executor = func(ctx context.Context, name string, args ...string) (string, error) {
			return "restore ok", nil
		}
		p.RestoreChecker.CacheDir = t.TempDir()

		metrics, err := p.FetchMetrics()
		require.NoError(t, err)

		assert.InDelta(t, float64(0), metrics["restore.myapp_db.restore_error"], 0.001)
		assert.Contains(t, metrics, "restore.myapp_db.restore_duration_seconds")
	})

	t.Run("restore check failure outputs restore_error: 1", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.URL = server.URL
		cfg.EnableRestoreCheck = true

		p := NewLitestreamPlugin(cfg)
		p.NowFunc = func() time.Time { return now }
		p.SnapshotCollector.Executor = mockSnapshotExec
		p.SnapshotCollector.CacheDir = t.TempDir()
		p.RestoreChecker.Executor = func(ctx context.Context, name string, args ...string) (string, error) {
			return "restore failed checksum mismatch", assert.AnError
		}
		p.RestoreChecker.CacheDir = t.TempDir()

		metrics, err := p.FetchMetrics()
		require.NoError(t, err)

		assert.InDelta(t, float64(1), metrics["restore.myapp_db.restore_error"], 0.001)
	})

	t.Run("snapshot collector error outputs snapshot_missing: 1", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.URL = server.URL

		p := NewLitestreamPlugin(cfg)
		p.NowFunc = func() time.Time { return now }
		p.SnapshotCollector.Executor = func(ctx context.Context, name string, args ...string) (string, error) {
			return "", assert.AnError
		}
		p.SnapshotCollector.CacheDir = t.TempDir()

		metrics, err := p.FetchMetrics()
		require.NoError(t, err)
		assert.InDelta(t, float64(1), metrics["snapshot_wal.myapp_db.snapshot_missing"], 0.001)
	})

	t.Run("db filter skips non matching db", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.URL = server.URL
		cfg.DBFilter = "/var/lib/other.db"

		p := NewLitestreamPlugin(cfg)
		p.NowFunc = func() time.Time { return now }
		p.SnapshotCollector.Executor = mockSnapshotExec
		p.SnapshotCollector.CacheDir = t.TempDir()

		metrics, err := p.FetchMetrics()
		require.NoError(t, err)
		// myapp_db はフィルタで除外されているためストレージメトリックは含まれない
		assert.NotContains(t, metrics, "storage.myapp_db.db_bytes")
	})
}

func TestLitestreamPlugin_MetricKeyPrefix(t *testing.T) {
	t.Parallel()

	t.Run("default metric key prefix", func(t *testing.T) {
		t.Parallel()
		p := NewLitestreamPlugin(DefaultConfig())
		assert.Equal(t, "litestream", p.MetricKeyPrefix())
	})

	t.Run("empty metric key prefix falls back to litestream", func(t *testing.T) {
		t.Parallel()
		p := &LitestreamPlugin{Config: Config{MetricKeyPrefix: ""}}
		assert.Equal(t, "litestream", p.MetricKeyPrefix())
	})

	t.Run("custom metric key prefix", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.MetricKeyPrefix = "custom_prefix"
		p := NewLitestreamPlugin(cfg)
		assert.Equal(t, "custom_prefix", p.MetricKeyPrefix())
	})
}
