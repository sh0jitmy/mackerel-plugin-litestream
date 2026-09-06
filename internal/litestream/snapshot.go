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

// SnapshotCollector はスナップショット情報を取得・集計する構造体です。
type SnapshotCollector struct {
	Executor   CommandExecutor
	BinaryPath string
	ConfigPath string
	CacheTTL   time.Duration
	CacheDir   string

	// メモリ内キャッシュ（同一プロセス内の連続呼び出し用）
	mu        sync.RWMutex
	memCache  map[string]SnapshotStats
	cacheTime map[string]time.Time
}

// NewSnapshotCollector は新しい SnapshotCollector を返します。
func NewSnapshotCollector(executor CommandExecutor, binaryPath, configPath string, cacheTTL time.Duration) *SnapshotCollector {
	if executor == nil {
		executor = DefaultCommandExecutor
	}
	if binaryPath == "" {
		binaryPath = "litestream"
	}
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	return &SnapshotCollector{
		Executor:   executor,
		BinaryPath: binaryPath,
		ConfigPath: configPath,
		CacheTTL:   cacheTTL,
		CacheDir:   os.TempDir(),
		memCache:   make(map[string]SnapshotStats),
		cacheTime:  make(map[string]time.Time),
	}
}

// cachePayload はファイルキャッシュ用の構造体です。
type cachePayload struct {
	CachedAt time.Time     `json:"cached_at"`
	Stats    SnapshotStats `json:"stats"`
}

// getCacheFilePath は指定された DB パスに対するキャッシュファイルパスを返します。
func (c *SnapshotCollector) getCacheFilePath(dbPath string) string {
	sanitized := SanitizeKey(dbPath)
	return filepath.Join(c.CacheDir, fmt.Sprintf("litestream_snap_cache_%s.json", sanitized))
}

// readDiskCache はディスクからキャッシュを読み込みます。
func (c *SnapshotCollector) readDiskCache(dbPath string, now time.Time) (*SnapshotStats, bool) {
	cacheFile := filepath.Clean(c.getCacheFilePath(dbPath))
	data, err := os.ReadFile(cacheFile) // #nosec G304
	if err != nil {
		return nil, false
	}

	var payload cachePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, false
	}

	if now.Sub(payload.CachedAt) > c.CacheTTL {
		return nil, false
	}

	return &payload.Stats, true
}

// writeDiskCache はディスクにキャッシュを保存します。
func (c *SnapshotCollector) writeDiskCache(dbPath string, stats SnapshotStats, now time.Time) {
	cacheFile := filepath.Clean(c.getCacheFilePath(dbPath))
	payload := cachePayload{
		CachedAt: now,
		Stats:    stats,
	}
	if data, err := json.Marshal(payload); err == nil {
		_ = os.WriteFile(cacheFile, data, 0600) // #nosec G304
	}
}

// Collect は指定されたデータベースの LTX/スナップショット情報を取得し、集計結果を返します。
func (c *SnapshotCollector) Collect(ctx context.Context, dbPath string, now time.Time) (SnapshotStats, error) {
	c.mu.RLock()
	cachedStats, exists := c.memCache[dbPath]
	cTime := c.cacheTime[dbPath]
	c.mu.RUnlock()

	if exists && now.Sub(cTime) <= c.CacheTTL {
		return cachedStats, nil
	}

	// ディスクキャッシュの確認（mackerel-agent は毎分プロセスを起動するためディスクキャッシュが有効）
	if diskStats, ok := c.readDiskCache(dbPath, now); ok {
		c.mu.Lock()
		c.memCache[dbPath] = *diskStats
		c.cacheTime[dbPath] = now
		c.mu.Unlock()
		return *diskStats, nil
	}

	// 直前のキャッシュ情報（消失検知用）
	var prevStats *SnapshotStats
	if exists {
		prevStats = &cachedStats
	}

	// litestream ltx コマンド引数の組み立て
	args := []string{"ltx", "-level", "all", "-json"}
	if c.ConfigPath != "" {
		args = append(args, "-config", c.ConfigPath)
	}
	if dbPath != "" {
		args = append(args, dbPath)
	}

	stdout, err := c.Executor(ctx, c.BinaryPath, args...)
	if err != nil {
		// コマンドが失敗した場合、直近のキャッシュがあればフォールバック
		if exists {
			return cachedStats, nil
		}
		return SnapshotStats{SnapshotMissing: 1.0}, fmt.Errorf("failed to run litestream ltx: %w (output: %s)", err, stdout)
	}

	stats, err := ParseLTXOutput(stdout, now)
	if err != nil {
		return SnapshotStats{SnapshotMissing: 1.0}, fmt.Errorf("failed to parse ltx output: %w", err)
	}

	// スナップショット消失検知（前回は存在していたのに今回 0 件になった場合）
	if prevStats != nil && prevStats.TotalSnapshotCount > 0 && stats.TotalSnapshotCount == 0 {
		stats.SnapshotMissing = 1.0
	}

	c.mu.Lock()
	c.memCache[dbPath] = stats
	c.cacheTime[dbPath] = now
	c.mu.Unlock()

	c.writeDiskCache(dbPath, stats, now)

	return stats, nil
}

// ParseLTXOutput は litestream ltx -json の出力を解析して SnapshotStats を算出します。
func ParseLTXOutput(rawJSON string, now time.Time) (SnapshotStats, error) {
	rawJSON = strings.TrimSpace(rawJSON)
	if rawJSON == "" || rawJSON == "[]" {
		return SnapshotStats{SnapshotMissing: 1.0}, nil
	}

	var files []LTXFile
	if err := json.Unmarshal([]byte(rawJSON), &files); err != nil {
		return SnapshotStats{SnapshotMissing: 1.0}, err
	}

	if len(files) == 0 {
		return SnapshotStats{SnapshotMissing: 1.0}, nil
	}

	var (
		totalObjects       float64
		totalBytes         float64
		snapshotCount      float64
		latestSnapshotTime time.Time
		hasSnapshot        bool
		l0Files            []LTXFile
	)

	for _, f := range files {
		totalObjects++
		totalBytes += float64(f.Size)

		switch f.Level {
		case 9:
			snapshotCount++
			if !hasSnapshot || f.Timestamp.After(latestSnapshotTime) {
				latestSnapshotTime = f.Timestamp
				hasSnapshot = true
			}
		case 0:
			l0Files = append(l0Files, f)
		}
	}

	var latestAgeSeconds float64
	var walSinceSnapshot float64
	var snapshotMissing float64

	if hasSnapshot {
		latestAgeSeconds = now.Sub(latestSnapshotTime).Seconds()
		if latestAgeSeconds < 0 {
			latestAgeSeconds = 0
		}

		// 最新スナップショット以降に作成された L0 ファイル数をカウント
		for _, l0 := range l0Files {
			if l0.Timestamp.After(latestSnapshotTime) || l0.Timestamp.Equal(latestSnapshotTime) {
				walSinceSnapshot++
			}
		}
	} else {
		// スナップショットがまだ存在しない場合
		walSinceSnapshot = float64(len(l0Files))
		snapshotMissing = 1.0
	}

	// タイムスタンプ順でソートして整頓（必要に応じて）
	sort.Slice(files, func(i, j int) bool {
		return files[i].Timestamp.Before(files[j].Timestamp)
	})

	return SnapshotStats{
		LatestSnapshotAgeSeconds: latestAgeSeconds,
		WALFilesSinceSnapshot:    walSinceSnapshot,
		TotalSnapshotCount:       snapshotCount,
		SnapshotMissing:          snapshotMissing,
		RemoteTotalBytes:         totalBytes,
		RemoteTotalObjects:       totalObjects,
	}, nil
}
