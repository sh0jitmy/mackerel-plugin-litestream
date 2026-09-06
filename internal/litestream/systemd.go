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
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// CommandExecutor は外部コマンドを実行する関数型です。テスト時のモックに使用します。
type CommandExecutor func(ctx context.Context, name string, args ...string) (string, error)

// DefaultCommandExecutor は標準の os/exec を用いてコマンドを実行します。
func DefaultCommandExecutor(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	if err != nil && output == "" {
		output = strings.TrimSpace(stderr.String())
	}
	return output, err
}

// SystemdChecker は systemd のサービス状態を確認する構造体です。
type SystemdChecker struct {
	Executor CommandExecutor
	Timeout  time.Duration
}

// NewSystemdChecker は新しい SystemdChecker を返します。
func NewSystemdChecker(executor CommandExecutor, timeout time.Duration) *SystemdChecker {
	if executor == nil {
		executor = DefaultCommandExecutor
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &SystemdChecker{
		Executor: executor,
		Timeout:  timeout,
	}
}

// CheckStatus は指定された systemd サービスの状態を判定します。
// 1. `systemctl is-active <service>` を実行
//   - "active" の場合 -> ServiceStateActive
//
// 2. "active" でない場合、`systemctl is-enabled <service>` を実行
//   - "enabled" の場合 -> ServiceStateInactiveEnabled (異常: 自動起動対象なのに停止中)
//   - "disabled" または "masked" の場合 -> ServiceStateInactiveDisabled (意図的停止: 監視除外)
//   - その他不明な場合 -> ServiceStateUnknown
func (c *SystemdChecker) CheckStatus(ctx context.Context, serviceName string) (ServiceState, error) {
	if serviceName == "" {
		return ServiceStateUnknown, nil
	}

	checkCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	// 1. systemctl is-active の判定
	activeOut, _ := c.Executor(checkCtx, "systemctl", "is-active", serviceName)
	activeOut = strings.TrimSpace(activeOut)
	if activeOut == "active" {
		return ServiceStateActive, nil
	}

	// 2. active ではない場合、is-enabled の判定
	enabledOut, _ := c.Executor(checkCtx, "systemctl", "is-enabled", serviceName)
	enabledOut = strings.TrimSpace(enabledOut)

	switch enabledOut {
	case "enabled", "enabled-runtime", "static", "indirect":
		// enabled なのに active でない場合は異常（障害またはクラッシュ）
		return ServiceStateInactiveEnabled, nil
	case "disabled", "masked", "not-found":
		// 意図的に disabled や masked にされている場合は監視対象外（スキップ）
		return ServiceStateInactiveDisabled, nil
	default:
		// 判定不能な場合は unknown
		return ServiceStateUnknown, nil
	}
}
