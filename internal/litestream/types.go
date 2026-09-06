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
	"time"
)

// Config は mackerel-plugin-litestream の設定を保持する構造体です。
type Config struct {
	// エンドポイント・基本設定
	URL             string
	MetricKeyPrefix string
	Tempfile        string
	Timeout         time.Duration
	DBFilter        string
	ConfigPath      string
	LitestreamBin   string

	// systemd 連携設定
	SystemdService string

	// スナップショット/S3 監視設定
	SnapshotCacheTTL time.Duration
	CacheFilePath    string

	// メトリクスグループの出力トグル（課金コスト抑制用）
	EnableCore          bool // 同期エラー, 検証エラー, ディスクフル (デフォルト: true)
	EnableStorage       bool // DB本体サイズ, WALサイズ (デフォルト: true)
	EnableSnapshot      bool // 最新スナップショット経過秒, 未コンパクションWAL数, S3オブジェクト統計 (デフォルト: true)
	EnableSyncStats     bool // 同期実行回数, 平均同期所要時間 (デフォルト: false)
	EnableCheckpoint    bool // チェックポイント実行回数, チェックポイントエラー (デフォルト: false)
	EnableRetention     bool // L0リテンションファイル数 (eligible, not_compacted, too_recent) (デフォルト: false)
	EnableReplicaOps    bool // S3等の操作回数, 転送バイト数, レイテンシ (デフォルト: false)
	EnableReplicaErrors bool // S3等のエラーコード別エラー数 (デフォルト: false)

	// リストア検証（冗長化時の復旧確認）設定
	EnableRestoreCheck   bool          // リストア検証メトリクスを収集するか (デフォルト: false)
	RestoreCheckInterval time.Duration // リストア検証の間隔/TTL (デフォルト: 30m)
	CheckRestoreMode     bool          // Mackerel チェックプラグイン互換モードで実行するか
	RestoreDryRun        bool          // リストア検証時に dry-run を使用するか (デフォルト: true)
}

// DefaultConfig は推奨されるデフォルト設定を返します。
// コスト最適化のため、必要最小限の重要メトリクス（Core, Storage, Snapshot）のみ有効化されています。
func DefaultConfig() Config {
	return Config{
		URL:                  "http://localhost:9090/metrics",
		MetricKeyPrefix:      "litestream",
		Timeout:              5 * time.Second,
		ConfigPath:           "/etc/litestream.yml",
		LitestreamBin:        "litestream",
		SnapshotCacheTTL:     5 * time.Minute,
		RestoreCheckInterval: 30 * time.Minute,
		RestoreDryRun:        true,
		EnableCore:           true,
		EnableStorage:        true,
		EnableSnapshot:       true,
		EnableSyncStats:      false,
		EnableCheckpoint:     false,
		EnableRetention:      false,
		EnableReplicaOps:     false,
		EnableReplicaErrors:  false,
		EnableRestoreCheck:   false,
		CheckRestoreMode:     false,
	}
}

// ServiceState は systemd サービスの状態を表す列挙型です。
type ServiceState int

const (
	// ServiceStateUnknown は状態が判定できない、または systemd 管理外であることを示します。
	ServiceStateUnknown ServiceState = iota
	// ServiceStateActive はサービスが正常に稼働中であることを示します。
	ServiceStateActive
	// ServiceStateInactiveEnabled は enabled だが停止・失敗している（異常状態）を示します。
	ServiceStateInactiveEnabled
	// ServiceStateInactiveDisabled は disabled で意図的に停止している（監視除外対象）を示します。
	ServiceStateInactiveDisabled
)

// SnapshotStats はスナップショットおよびリモートストレージの集計結果を保持します。
type SnapshotStats struct {
	LatestSnapshotAgeSeconds float64 // 最新スナップショットからの経過秒数
	WALFilesSinceSnapshot    float64 // 最新スナップショット以降の未コンパクションWAL (L0 LTX) ファイル数
	TotalSnapshotCount       float64 // スナップショット (Level 9 LTX) の総数
	SnapshotMissing          float64 // スナップショット不在・消失フラグ (1: 不在/消失, 0: 正常に存在)
	RemoteTotalBytes         float64 // リモートストレージ上の全ファイル合計サイズ (Bytes)
	RemoteTotalObjects       float64 // リモートストレージ上の総ファイル数
}

// RestoreStats はリストア検証（復元確認）の実行結果を保持します。
type RestoreStats struct {
	Success         bool      `json:"success"`
	DurationSeconds float64   `json:"duration_seconds"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	CheckedAt       time.Time `json:"checked_at"`
}

// LTXFile は litestream ltx -json の出力を表す構造体です。
type LTXFile struct {
	Level     int       `json:"level"`
	MinTXID   string    `json:"min_txid"`
	MaxTXID   string    `json:"max_txid"`
	Size      int64     `json:"size"`
	Timestamp time.Time `json:"timestamp"`
}
