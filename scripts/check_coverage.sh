#!/bin/bash
# Copyright 2026 [Copyright Holder]
# Licensed under the Apache License, Version 2.0 (the "License");

set -e

# カバレッジ測定用の対象パッケージリストの生成
COVERPKG=$(go list ./... | paste -sd, -)

# 全パッケージのテスト実行とカバレッジプロファイルの出力
echo "==> Running tests with coverage profile..."
go test -v -race -coverprofile=coverage.out -coverpkg="$COVERPKG" ./...

# internal/litestream 配下の合計ステートメントカバー率を検証
echo "==> Verifying core logic coverage (internal/litestream)..."
awk '
BEGIN { total = 0; covered = 0; }
/:/ {
    if ($0 ~ /\/internal\/litestream\//) {
        total += $2;
        if ($3 > 0) {
            covered += $2;
        }
    }
}
END {
    if (total == 0) {
        print "ERROR: No statements found in internal/litestream/."
        exit 1
    }
    rate = (covered / total) * 100
    printf "=========================================\n"
    printf "Litestream Plugin Coverage Summary:\n"
    printf "  Covered Statements: %d\n", covered
    printf "  Total Statements:   %d\n", total
    printf "  Coverage Rate:      %.2f%%\n", rate
    printf "=========================================\n"
    if (rate < 95.0) {
        printf "ERROR: Core logic coverage is %.2f%%, which is below the required 95.0%%!\n", rate
        exit 1
    }
    printf "SUCCESS: Core logic coverage is %.2f%% (>= 95.0%%)\n", rate
}
' coverage.out
