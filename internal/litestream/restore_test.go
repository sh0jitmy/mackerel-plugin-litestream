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
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreChecker_Defaults(t *testing.T) {
	t.Parallel()

	rc := NewRestoreChecker(nil, "", "", 0, true)
	assert.NotNil(t, rc.Executor)
	assert.Equal(t, "litestream", rc.BinaryPath)
	assert.Equal(t, 30*time.Minute, rc.Interval)
	assert.True(t, rc.DryRun)
}

func TestRestoreChecker_Check(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()

	t.Run("successful dry-run restore check", func(t *testing.T) {
		t.Parallel()
		var capturedArgs []string
		mockExec := func(ctx context.Context, name string, args ...string) (string, error) {
			capturedArgs = args
			return "restore dry-run completed successfully", nil
		}

		rc := NewRestoreChecker(mockExec, "litestream", "/etc/litestream.yml", 30*time.Minute, true)
		rc.CacheDir = tempDir

		stats, err := rc.Check(ctx, "test.db", now)
		require.NoError(t, err)
		assert.True(t, stats.Success)
		assert.Empty(t, stats.ErrorMessage)
		assert.Contains(t, capturedArgs, "-dry-run")
		assert.Contains(t, capturedArgs, "-config")
		assert.Contains(t, capturedArgs, "test.db")
	})

	t.Run("failed restore check returns error and error message", func(t *testing.T) {
		t.Parallel()
		mockExec := func(ctx context.Context, name string, args ...string) (string, error) {
			return "cannot restore: replica not found in s3 bucket", errors.New("exit status 1")
		}

		rc := NewRestoreChecker(mockExec, "litestream", "", 30*time.Minute, true)
		rc.CacheDir = tempDir

		stats, err := rc.Check(ctx, "missing.db", now)
		require.Error(t, err)
		assert.False(t, stats.Success)
		assert.Contains(t, stats.ErrorMessage, "replica not found")
	})

	t.Run("failed restore check with empty stdout uses error string", func(t *testing.T) {
		t.Parallel()
		mockExec := func(ctx context.Context, name string, args ...string) (string, error) {
			return "", errors.New("command timeout")
		}

		rc := NewRestoreChecker(mockExec, "litestream", "", 30*time.Minute, true)
		rc.CacheDir = tempDir

		stats, err := rc.Check(ctx, "timeout.db", now)
		require.Error(t, err)
		assert.False(t, stats.Success)
		assert.Equal(t, "command timeout", stats.ErrorMessage)
	})

	t.Run("actual restore with dryRun=false cleans up temp files", func(t *testing.T) {
		t.Parallel()
		var capturedOutPath string
		mockExec := func(ctx context.Context, name string, args ...string) (string, error) {
			for i, arg := range args {
				if arg == "-o" && i+1 < len(args) {
					capturedOutPath = args[i+1]
					// 模擬的に一時ファイルを作成
					_ = os.WriteFile(capturedOutPath, []byte("sqlite db data"), 0600)
					_ = os.WriteFile(capturedOutPath+"-wal", []byte("wal data"), 0600)
				}
			}
			return "restored 1000 pages", nil
		}

		rc := NewRestoreChecker(mockExec, "litestream", "", 30*time.Minute, false)
		rc.CacheDir = tempDir

		stats, err := rc.Check(ctx, "actual.db", now)
		require.NoError(t, err)
		assert.True(t, stats.Success)
		assert.NotEmpty(t, capturedOutPath)

		// defer でファイルが削除されていることを確認
		_, statErr := os.Stat(capturedOutPath)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("caching in memory and on disk", func(t *testing.T) {
		t.Parallel()
		callCount := 0
		mockExec := func(ctx context.Context, name string, args ...string) (string, error) {
			callCount++
			return "ok", nil
		}

		rc := NewRestoreChecker(mockExec, "litestream", "", 10*time.Minute, true)
		rc.CacheDir = tempDir

		// 1回目実行
		stats1, err := rc.Check(ctx, "cache.db", now)
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)

		// 2回目実行 (5分後、TTL内) -> メモリキャッシュヒット
		stats2, err := rc.Check(ctx, "cache.db", now.Add(5*time.Minute))
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)
		assert.Equal(t, stats1.Success, stats2.Success)

		// プロセス再起動を模倣した新インスタンス -> ディスクキャッシュヒット
		rc2 := NewRestoreChecker(mockExec, "litestream", "", 10*time.Minute, true)
		rc2.CacheDir = tempDir
		stats3, err := rc2.Check(ctx, "cache.db", now.Add(7*time.Minute))
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)
		assert.Equal(t, stats1.Success, stats3.Success)

		// TTL経過後 (15分後) -> 再実行
		stats4, err := rc.Check(ctx, "cache.db", now.Add(15*time.Minute))
		require.NoError(t, err)
		assert.Equal(t, 2, callCount)
		assert.Equal(t, stats1.Success, stats4.Success)
	})

	t.Run("corrupted disk cache file is ignored", func(t *testing.T) {
		t.Parallel()
		mockExec := func(ctx context.Context, name string, args ...string) (string, error) {
			return "ok", nil
		}

		rc := NewRestoreChecker(mockExec, "litestream", "", 10*time.Minute, true)
		rc.CacheDir = tempDir

		cacheFile := rc.getCacheFilePath("corrupt.db")
		_ = os.WriteFile(cacheFile, []byte("{invalid json"), 0600)

		stats, err := rc.Check(ctx, "corrupt.db", now)
		require.NoError(t, err)
		assert.True(t, stats.Success)
	})
}

func TestFormatCheckOutput(t *testing.T) {
	t.Parallel()

	t.Run("empty stats map returns OK", func(t *testing.T) {
		t.Parallel()
		msg, code := FormatCheckOutput(nil)
		assert.Equal(t, 0, code)
		assert.Contains(t, msg, "LITESTREAM RESTORE OK")
	})

	t.Run("all databases succeeded returns OK with avg duration", func(t *testing.T) {
		t.Parallel()
		statsMap := map[string]RestoreStats{
			"db1": {Success: true, DurationSeconds: 0.10},
			"db2": {Success: true, DurationSeconds: 0.20},
		}
		msg, code := FormatCheckOutput(statsMap)
		assert.Equal(t, 0, code)
		assert.Contains(t, msg, "LITESTREAM RESTORE OK: all 2 databases verified successfully")
	})

	t.Run("one or more databases failed returns CRITICAL with exit code 2", func(t *testing.T) {
		t.Parallel()
		statsMap := map[string]RestoreStats{
			"db1": {Success: true, DurationSeconds: 0.10},
			"db2": {Success: false, ErrorMessage: "corrupted snapshot checksum"},
		}
		msg, code := FormatCheckOutput(statsMap)
		assert.Equal(t, 2, code)
		assert.Contains(t, msg, "LITESTREAM RESTORE CRITICAL")
		assert.Contains(t, msg, "db2 (corrupted snapshot checksum)")
	})
}
