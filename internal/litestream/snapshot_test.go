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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLTXOutput(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	t.Run("empty json returns zero stats with snapshot missing flag", func(t *testing.T) {
		t.Parallel()
		stats, err := ParseLTXOutput("[]", now)
		require.NoError(t, err)
		assert.Equal(t, SnapshotStats{SnapshotMissing: 1}, stats)

		stats, err = ParseLTXOutput("", now)
		require.NoError(t, err)
		assert.Equal(t, SnapshotStats{SnapshotMissing: 1}, stats)
	})

	t.Run("valid ltx output with snapshots and WALs", func(t *testing.T) {
		t.Parallel()
		rawJSON := `[
			{"level": 9, "min_txid": "00000001", "max_txid": "00000050", "size": 1000000, "timestamp": "2026-09-04T10:00:00Z"},
			{"level": 0, "min_txid": "00000051", "max_txid": "00000052", "size": 4096, "timestamp": "2026-09-04T12:00:00Z"},
			{"level": 9, "min_txid": "00000001", "max_txid": "00000100", "size": 1500000, "timestamp": "2026-09-05T10:00:00Z"},
			{"level": 0, "min_txid": "00000101", "max_txid": "00000102", "size": 8192, "timestamp": "2026-09-05T10:30:00Z"},
			{"level": 0, "min_txid": "00000103", "max_txid": "00000104", "size": 12288, "timestamp": "2026-09-05T11:00:00Z"}
		]`

		stats, err := ParseLTXOutput(rawJSON, now)
		require.NoError(t, err)

		assert.InDelta(t, float64(7200), stats.LatestSnapshotAgeSeconds, 0.001)
		assert.InDelta(t, float64(2), stats.WALFilesSinceSnapshot, 0.001)
		assert.InDelta(t, float64(2), stats.TotalSnapshotCount, 0.001)
		assert.InDelta(t, float64(0), stats.SnapshotMissing, 0.001) // 正常に存在
		assert.InDelta(t, float64(5), stats.RemoteTotalObjects, 0.001)
		assert.InDelta(t, float64(2524576), stats.RemoteTotalBytes, 0.001)
	})

	t.Run("ltx output with no snapshots", func(t *testing.T) {
		t.Parallel()
		rawJSON := `[
			{"level": 0, "min_txid": "00000001", "max_txid": "00000002", "size": 4096, "timestamp": "2026-09-05T11:00:00Z"},
			{"level": 0, "min_txid": "00000003", "max_txid": "00000004", "size": 8192, "timestamp": "2026-09-05T11:30:00Z"}
		]`

		stats, err := ParseLTXOutput(rawJSON, now)
		require.NoError(t, err)

		assert.InDelta(t, float64(0), stats.LatestSnapshotAgeSeconds, 0.001)
		assert.InDelta(t, float64(0), stats.TotalSnapshotCount, 0.001)
		assert.InDelta(t, float64(1), stats.SnapshotMissing, 0.001) // スナップショット不在
		assert.InDelta(t, float64(2), stats.WALFilesSinceSnapshot, 0.001)
		assert.InDelta(t, float64(2), stats.RemoteTotalObjects, 0.001)
		assert.InDelta(t, float64(12288), stats.RemoteTotalBytes, 0.001)
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		t.Parallel()
		stats, err := ParseLTXOutput("{invalid-json}", now)
		require.Error(t, err)
		assert.InDelta(t, float64(1), stats.SnapshotMissing, 0.001)
	})
}

func TestSnapshotCollector_DefaultsAndErrors(t *testing.T) {
	t.Parallel()

	// NewSnapshotCollector のデフォルト引数検証
	sc := NewSnapshotCollector(nil, "", "", 0)
	assert.NotNil(t, sc.Executor)
	assert.Equal(t, "litestream", sc.BinaryPath)
	assert.Equal(t, 5*time.Minute, sc.CacheTTL)

	ctx := context.Background()
	now := time.Now()
	tempDir := t.TempDir()

	t.Run("command failure with no cache returns error and missing flag", func(t *testing.T) {
		t.Parallel()
		mockFail := func(ctx context.Context, name string, args ...string) (string, error) {
			return "command error", assert.AnError
		}
		collector := NewSnapshotCollector(mockFail, "litestream", "", 5*time.Minute)
		collector.CacheDir = tempDir

		stats, err := collector.Collect(ctx, "nonexistent.db", now)
		require.Error(t, err)
		assert.InDelta(t, float64(1), stats.SnapshotMissing, 0.001)
	})

	t.Run("command failure with existing cache falls back to cache", func(t *testing.T) {
		t.Parallel()
		callCount := 0
		mockExec := func(ctx context.Context, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 {
				return `[{"level": 9, "min_txid": "1", "max_txid": "2", "size": 1000, "timestamp": "2026-09-05T10:00:00Z"}]`, nil
			}
			return "network error", assert.AnError
		}
		collector := NewSnapshotCollector(mockExec, "litestream", "", 10*time.Second)
		collector.CacheDir = tempDir

		// 1回目 (成功)
		stats1, err := collector.Collect(ctx, "fallback.db", now)
		require.NoError(t, err)
		assert.InDelta(t, float64(1), stats1.TotalSnapshotCount, 0.001)

		// 2回目 (TTL切れ後だがコマンド失敗 -> キャッシュにフォールバック)
		stats2, err := collector.Collect(ctx, "fallback.db", now.Add(20*time.Second))
		require.NoError(t, err)
		assert.InDelta(t, stats1.TotalSnapshotCount, stats2.TotalSnapshotCount, 0.001)
	})

	t.Run("snapshot disappearance detection (existed before, now 0)", func(t *testing.T) {
		t.Parallel()
		callCount := 0
		mockExec := func(ctx context.Context, name string, args ...string) (string, error) {
			callCount++
			if callCount == 1 {
				// 初回: スナップショットあり
				return `[{"level": 9, "min_txid": "1", "max_txid": "2", "size": 1000, "timestamp": "2026-09-05T10:00:00Z"}]`, nil
			}
			// 2回目: スナップショット消失 (L0のみ残る)
			return `[{"level": 0, "min_txid": "3", "max_txid": "4", "size": 500, "timestamp": "2026-09-05T11:00:00Z"}]`, nil
		}
		collector := NewSnapshotCollector(mockExec, "litestream", "/etc/litestream.yml", 10*time.Second)
		collector.CacheDir = tempDir

		stats1, err := collector.Collect(ctx, "disappear.db", now)
		require.NoError(t, err)
		assert.InDelta(t, float64(0), stats1.SnapshotMissing, 0.001)

		// TTL経過後に取得 -> スナップショット消失を検知
		stats2, err := collector.Collect(ctx, "disappear.db", now.Add(20*time.Second))
		require.NoError(t, err)
		assert.InDelta(t, float64(0), stats2.TotalSnapshotCount, 0.001)
		assert.InDelta(t, float64(1), stats2.SnapshotMissing, 0.001)
	})

	t.Run("corrupted disk cache file is ignored", func(t *testing.T) {
		t.Parallel()
		mockExec := func(ctx context.Context, name string, args ...string) (string, error) {
			return `[{"level": 9, "min_txid": "1", "max_txid": "2", "size": 1000, "timestamp": "2026-09-05T10:00:00Z"}]`, nil
		}
		collector := NewSnapshotCollector(mockExec, "litestream", "", 5*time.Minute)
		collector.CacheDir = tempDir

		// 壊れたJSONキャッシュファイルを配置
		cacheFile := collector.getCacheFilePath("corrupt.db")
		_ = os.WriteFile(cacheFile, []byte("{corrupted json"), 0600)

		// 正常にパースして新しいデータを取得できること
		stats, err := collector.Collect(ctx, "corrupt.db", now)
		require.NoError(t, err)
		assert.InDelta(t, float64(1), stats.TotalSnapshotCount, 0.001)
	})
}

func TestSnapshotCollector_Caching(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()

	callCount := 0
	mockExec := func(ctx context.Context, name string, args ...string) (string, error) {
		callCount++
		return `[
			{"level": 9, "min_txid": "1", "max_txid": "10", "size": 5000, "timestamp": "2026-09-05T11:00:00Z"}
		]`, nil
	}

	collector := NewSnapshotCollector(mockExec, "litestream", "", 5*time.Minute)
	collector.CacheDir = tempDir

	dbPath := filepath.Join(tempDir, "test.db")
	_ = os.WriteFile(dbPath, []byte(""), 0600)

	// 1回目の呼び出し -> mockExec が呼ばれる
	stats1, err := collector.Collect(ctx, dbPath, now)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.InDelta(t, float64(3600), stats1.LatestSnapshotAgeSeconds, 0.001)

	// 2回目の呼び出し (TTL内: 2分後) -> キャッシュから返され、mockExec は呼ばれない
	stats2, err := collector.Collect(ctx, dbPath, now.Add(2*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 1, callCount) // 増えていない
	assert.Equal(t, stats1, stats2)

	// 新しいコレクタ（プロセス再起動を模倣）を作成し、ディスクキャッシュが機能するか検証
	collector2 := NewSnapshotCollector(mockExec, "litestream", "", 5*time.Minute)
	collector2.CacheDir = tempDir
	stats3, err := collector2.Collect(ctx, dbPath, now.Add(3*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 1, callCount) // ディスクキャッシュがヒットし、外部コマンドは実行されない
	assert.Equal(t, stats1, stats3)

	// TTL経過後 (6分後) -> mockExec が再度呼ばれる
	stats4, err := collector.Collect(ctx, dbPath, now.Add(6*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 2, callCount) // 再実行された
	assert.InDelta(t, float64(1), stats4.TotalSnapshotCount, 0.001)
}
