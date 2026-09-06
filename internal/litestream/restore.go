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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RestoreChecker は litestream restore コマンドを用いたリストア検証を行う構造体です。
type RestoreChecker struct {
	Executor      CommandExecutor
	BinaryPath    string
	ConfigPath    string
	Interval      time.Duration
	DryRun        bool
	CacheDir      string
	mu            sync.RWMutex
	memCache      map[string]RestoreStats
	lastCheckedAt map[string]time.Time
}

// NewRestoreChecker は新しい RestoreChecker を返します。
func NewRestoreChecker(executor CommandExecutor, binaryPath, configPath string, interval time.Duration, dryRun bool) *RestoreChecker {
	if executor == nil {
		executor = DefaultCommandExecutor
	}
	if binaryPath == "" {
		binaryPath = "litestream"
	}
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return &RestoreChecker{
		Executor:      executor,
		BinaryPath:    binaryPath,
		ConfigPath:    configPath,
		Interval:      interval,
		DryRun:        dryRun,
		CacheDir:      os.TempDir(),
		memCache:      make(map[string]RestoreStats),
		lastCheckedAt: make(map[string]time.Time),
	}
}

// restoreCachePayload はディスクキャッシュ用の構造体です。
type restoreCachePayload struct {
	Stats RestoreStats `json:"stats"`
}

// getCacheFilePath は指定された DB パスに対するキャッシュファイルパスを返します。
func (r *RestoreChecker) getCacheFilePath(dbPath string) string {
	sanitized := SanitizeKey(dbPath)
	return filepath.Join(r.CacheDir, fmt.Sprintf("litestream_restore_cache_%s.json", sanitized))
}

// readDiskCache はディスクからキャッシュを読み込みます。
func (r *RestoreChecker) readDiskCache(dbPath string, now time.Time) (*RestoreStats, bool) {
	cacheFile := filepath.Clean(r.getCacheFilePath(dbPath))
	data, err := os.ReadFile(cacheFile) // #nosec G304
	if err != nil {
		return nil, false
	}

	var payload restoreCachePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, false
	}

	if now.Sub(payload.Stats.CheckedAt) > r.Interval {
		return nil, false
	}

	return &payload.Stats, true
}

// writeDiskCache はディスクにキャッシュを保存します。
func (r *RestoreChecker) writeDiskCache(dbPath string, stats RestoreStats) {
	cacheFile := filepath.Clean(r.getCacheFilePath(dbPath))
	payload := restoreCachePayload{
		Stats: stats,
	}
	if data, err := json.Marshal(payload); err == nil {
		_ = os.WriteFile(cacheFile, data, 0600) // #nosec G304
	}
}

// Check は指定されたデータベースのレプリカ復元可能性を検証します。
func (r *RestoreChecker) Check(ctx context.Context, dbPath string, now time.Time) (RestoreStats, error) {
	r.mu.RLock()
	cachedStats, exists := r.memCache[dbPath]
	cTime := r.lastCheckedAt[dbPath]
	r.mu.RUnlock()

	if exists && now.Sub(cTime) <= r.Interval {
		return cachedStats, nil
	}

	// ディスクキャッシュの確認
	if diskStats, ok := r.readDiskCache(dbPath, now); ok {
		r.mu.Lock()
		r.memCache[dbPath] = *diskStats
		r.lastCheckedAt[dbPath] = now
		r.mu.Unlock()
		return *diskStats, nil
	}

	// コマンド引数の組み立て
	args := []string{"restore"}
	if r.ConfigPath != "" {
		args = append(args, "-config", r.ConfigPath)
	}

	var tempOutPath string
	if r.DryRun {
		args = append(args, "-dry-run")
	} else {
		// 実リストア検証の場合は一時ファイルへ復元して即座に削除
		tempOutPath = filepath.Join(r.CacheDir, fmt.Sprintf("litestream_restore_tmp_%d_%s.db", now.UnixNano(), SanitizeKey(dbPath)))
		args = append(args, "-o", tempOutPath)
		defer func() {
			if tempOutPath != "" {
				_ = os.Remove(tempOutPath)
				_ = os.Remove(tempOutPath + "-wal")
				_ = os.Remove(tempOutPath + "-shm")
			}
		}()
	}

	if dbPath != "" {
		args = append(args, dbPath)
	}

	startTime := now
	stdout, err := r.Executor(ctx, r.BinaryPath, args...)
	duration := time.Since(startTime).Seconds()

	stats := RestoreStats{
		DurationSeconds: duration,
		CheckedAt:       now,
	}

	if err != nil {
		stats.Success = false
		stats.ErrorMessage = strings.TrimSpace(stdout)
		if stats.ErrorMessage == "" {
			stats.ErrorMessage = err.Error()
		}
	} else {
		stats.Success = true
	}

	r.mu.Lock()
	r.memCache[dbPath] = stats
	r.lastCheckedAt[dbPath] = now
	r.mu.Unlock()

	r.writeDiskCache(dbPath, stats)

	if !stats.Success {
		return stats, fmt.Errorf("restore check failed: %s", stats.ErrorMessage)
	}
	return stats, nil
}

// FormatCheckOutput は Mackerel チェック監視用のメッセージと終了コード (0: OK, 2: CRITICAL) を返します。
func FormatCheckOutput(statsMap map[string]RestoreStats) (string, int) {
	if len(statsMap) == 0 {
		return "LITESTREAM RESTORE OK: no databases configured for restore check", 0
	}

	// 安定した出力順にするためソート
	keys := make([]string, 0, len(statsMap))
	for k := range statsMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var failedDBs []string
	var successCount int
	var totalDuration float64

	for _, db := range keys {
		stats := statsMap[db]
		if !stats.Success {
			failedDBs = append(failedDBs, fmt.Sprintf("%s (%s)", db, stats.ErrorMessage))
		} else {
			successCount++
			totalDuration += stats.DurationSeconds
		}
	}

	if len(failedDBs) > 0 {
		return fmt.Sprintf("LITESTREAM RESTORE CRITICAL: %s", strings.Join(failedDBs, ", ")), 2
	}

	return fmt.Sprintf("LITESTREAM RESTORE OK: all %d databases verified successfully (avg duration: %.2fs)", successCount, totalDuration/float64(successCount)), 0
}
