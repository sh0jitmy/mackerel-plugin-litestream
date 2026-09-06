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

// Package main is the entry point for mackerel-plugin-litestream.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	mp "github.com/mackerelio/go-mackerel-plugin"
	"github.com/shjtmy/mackerel-plugin-litestream/internal/litestream"
)

func main() {
	cfg := litestream.DefaultConfig()

	optURL := flag.String("url", cfg.URL, "Litestream Prometheus metrics endpoint URL")
	optPrefix := flag.String("metric-key-prefix", cfg.MetricKeyPrefix, "Metric key prefix")
	optTempfile := flag.String("tempfile", "", "Path to temp file for diff calculations")
	optTimeout := flag.Duration("timeout", cfg.Timeout, "HTTP and command execution timeout")
	optDB := flag.String("db", "", "Filter for a specific database path or name (optional)")
	optConfigPath := flag.String("config", cfg.ConfigPath, "Path to Litestream config file")
	optBin := flag.String("litestream-bin", cfg.LitestreamBin, "Path to litestream binary")
	optSystemd := flag.String("systemd-service", "", "Systemd service name to check status (e.g. litestream.service)")
	optCacheTTL := flag.Duration("snapshot-cache-ttl", cfg.SnapshotCacheTTL, "Cache TTL for remote snapshot/LTX listing")

	// メトリクス課金抑制トグル
	optEnableCore := flag.Bool("enable-core", cfg.EnableCore, "Enable core error metrics (sync errors, verify errors, disk full)")
	optEnableStorage := flag.Bool("enable-storage", cfg.EnableStorage, "Enable database and WAL file size metrics")
	optEnableSnapshot := flag.Bool("enable-snapshot", cfg.EnableSnapshot, "Enable snapshot age, uncompacted WAL count, and S3 stats")
	optEnableSyncStats := flag.Bool("enable-sync-stats", cfg.EnableSyncStats, "Enable sync operations count and duration metrics")
	optEnableCheckpoint := flag.Bool("enable-checkpoint", cfg.EnableCheckpoint, "Enable checkpoint count and error metrics")
	optEnableRetention := flag.Bool("enable-retention", cfg.EnableRetention, "Enable L0 retention file status metrics")
	optEnableReplicaOps := flag.Bool("enable-replica-ops", cfg.EnableReplicaOps, "Enable replica storage operations and traffic metrics")
	optEnableReplicaErrors := flag.Bool("enable-replica-errors", cfg.EnableReplicaErrors, "Enable replica error code metrics")

	// リストア検証オプション
	optEnableRestoreCheck := flag.Bool("enable-restore-check", cfg.EnableRestoreCheck, "Enable restore verification metrics")
	optRestoreCheckInterval := flag.Duration("restore-check-interval", cfg.RestoreCheckInterval, "Interval / cache TTL for restore verification")
	optRestoreDryRun := flag.Bool("restore-dry-run", cfg.RestoreDryRun, "Use dry-run for restore verification to avoid disk write")
	optCheckRestore := flag.Bool("check-restore", false, "Run in Mackerel check plugin mode for restore verification")

	optPrintGraphDefs := flag.Bool("print-graph-defs", false, "Print graph definitions")
	optVersion := flag.Bool("version", false, "Show version and exit")

	flag.Parse()

	if *optVersion {
		fmt.Printf("mackerel-plugin-litestream version %s\n", Version)
		os.Exit(0)
	}

	cfg.URL = *optURL
	cfg.MetricKeyPrefix = *optPrefix
	cfg.Tempfile = *optTempfile
	cfg.Timeout = *optTimeout
	cfg.DBFilter = *optDB
	cfg.ConfigPath = *optConfigPath
	cfg.LitestreamBin = *optBin
	cfg.SystemdService = *optSystemd
	cfg.SnapshotCacheTTL = *optCacheTTL

	cfg.EnableCore = *optEnableCore
	cfg.EnableStorage = *optEnableStorage
	cfg.EnableSnapshot = *optEnableSnapshot
	cfg.EnableSyncStats = *optEnableSyncStats
	cfg.EnableCheckpoint = *optEnableCheckpoint
	cfg.EnableRetention = *optEnableRetention
	cfg.EnableReplicaOps = *optEnableReplicaOps
	cfg.EnableReplicaErrors = *optEnableReplicaErrors

	cfg.EnableRestoreCheck = *optEnableRestoreCheck
	cfg.RestoreCheckInterval = *optRestoreCheckInterval
	cfg.RestoreDryRun = *optRestoreDryRun
	cfg.CheckRestoreMode = *optCheckRestore

	// Mackerel チェック監視モード
	if *optCheckRestore {
		checker := litestream.NewRestoreChecker(nil, cfg.LitestreamBin, cfg.ConfigPath, cfg.RestoreCheckInterval, cfg.RestoreDryRun)
		statsMap := make(map[string]litestream.RestoreStats)
		dbTarget := cfg.DBFilter
		targetName := dbTarget
		if targetName == "" {
			targetName = "default"
		}
		rStats, _ := checker.Check(context.Background(), dbTarget, time.Now())
		statsMap[targetName] = rStats

		msg, exitCode := litestream.FormatCheckOutput(statsMap)
		fmt.Println(msg)
		os.Exit(exitCode)
	}

	plugin := litestream.NewLitestreamPlugin(cfg)

	helper := mp.NewMackerelPlugin(plugin)
	helper.Tempfile = *optTempfile

	if *optPrintGraphDefs {
		helper.OutputDefinitions()
		return
	}

	helper.Run()
}
