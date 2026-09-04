#!/bin/bash
# Copyright 2026 [Copyright Holder]
# Licensed under the Apache License, Version 2.0 (the "License");

set -e

# カバレッジ測定用の対象パッケージリストの生成（自動生成コード ent, ogen を除外）
COVERPKG=$(go list ./... | grep -v -E '/ent|/ogen' | paste -sd, -)

# 全パッケージのテスト実行とカバレッジプロファイルの出力
echo "==> Running tests with coverage profile..."
go test -v -race -coverprofile=coverage.out -coverpkg="$COVERPKG" ./...

# internal/service/ および internal/domain/ 配下の合計ステートメントカバー率を検証
echo "==> Verifying business logic coverage (internal/service, internal/domain)..."
awk '
BEGIN { total = 0; covered = 0; }
/:/ {
    if ($0 ~ /\/internal\/(service|domain)\//) {
        total += $2;
        if ($3 > 0) {
            covered += $2;
        }
    }
}
END {
    if (total == 0) {
        print "ERROR: No statements found in internal/service/ or internal/domain/."
        exit 1
    }
    rate = (covered / total) * 100
    printf "=========================================\n"
    printf "Business Logic Coverage (service/domain) Summary:\n"
    printf "  Covered Statements: %d\n", covered
    printf "  Total Statements:   %d\n", total
    printf "  Coverage Rate:      %.2f%%\n", rate
    printf "=========================================\n"
    if (rate < 80.0) {
        printf "ERROR: Business logic coverage is %.2f%%, which is below the required 80.0%%!\n", rate
        exit 1
    }
    printf "SUCCESS: Business logic coverage is %.2f%% (>= 80.0%%)\n", rate
}
' coverage.out
