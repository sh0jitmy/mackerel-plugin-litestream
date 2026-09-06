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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemdChecker_CheckStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name        string
		serviceName string
		mockExec    CommandExecutor
		wantState   ServiceState
	}{
		{
			name:        "empty service name returns Unknown",
			serviceName: "",
			mockExec:    nil,
			wantState:   ServiceStateUnknown,
		},
		{
			name:        "active service returns ServiceStateActive",
			serviceName: "litestream.service",
			mockExec: func(ctx context.Context, name string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "is-active" {
					return "active", nil
				}
				return "", errors.New("unexpected command")
			},
			wantState: ServiceStateActive,
		},
		{
			name:        "inactive but enabled service returns ServiceStateInactiveEnabled",
			serviceName: "litestream.service",
			mockExec: func(ctx context.Context, name string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "is-active" {
					return "inactive", errors.New("inactive")
				}
				if len(args) >= 2 && args[0] == "is-enabled" {
					return "enabled", nil
				}
				return "", errors.New("unexpected command")
			},
			wantState: ServiceStateInactiveEnabled,
		},
		{
			name:        "failed but enabled-runtime service returns ServiceStateInactiveEnabled",
			serviceName: "litestream.service",
			mockExec: func(ctx context.Context, name string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "is-active" {
					return "failed", errors.New("failed")
				}
				if len(args) >= 2 && args[0] == "is-enabled" {
					return "enabled-runtime", nil
				}
				return "", errors.New("unexpected command")
			},
			wantState: ServiceStateInactiveEnabled,
		},
		{
			name:        "inactive and disabled service returns ServiceStateInactiveDisabled",
			serviceName: "litestream.service",
			mockExec: func(ctx context.Context, name string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "is-active" {
					return "inactive", errors.New("inactive")
				}
				if len(args) >= 2 && args[0] == "is-enabled" {
					return "disabled", errors.New("disabled")
				}
				return "", errors.New("unexpected command")
			},
			wantState: ServiceStateInactiveDisabled,
		},
		{
			name:        "masked service returns ServiceStateInactiveDisabled",
			serviceName: "litestream.service",
			mockExec: func(ctx context.Context, name string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "is-active" {
					return "inactive", errors.New("inactive")
				}
				if len(args) >= 2 && args[0] == "is-enabled" {
					return "masked", errors.New("masked")
				}
				return "", errors.New("unexpected command")
			},
			wantState: ServiceStateInactiveDisabled,
		},
		{
			name:        "unknown state returns ServiceStateUnknown",
			serviceName: "litestream.service",
			mockExec: func(ctx context.Context, name string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "is-active" {
					return "unknown", errors.New("unknown")
				}
				if len(args) >= 2 && args[0] == "is-enabled" {
					return "something-else", errors.New("unknown")
				}
				return "", errors.New("unexpected command")
			},
			wantState: ServiceStateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			checker := NewSystemdChecker(tt.mockExec, 2*time.Second)
			state, err := checker.CheckStatus(ctx, tt.serviceName)
			require.NoError(t, err)
			assert.Equal(t, tt.wantState, state)
		})
	}
}

func TestSystemdChecker_DefaultsAndExecutor(t *testing.T) {
	t.Parallel()

	t.Run("defaults when nil or zero arguments provided", func(t *testing.T) {
		t.Parallel()
		checker := NewSystemdChecker(nil, 0)
		assert.NotNil(t, checker.Executor)
		assert.Equal(t, 3*time.Second, checker.Timeout)
	})

	t.Run("DefaultCommandExecutor executes standard command successfully", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		out, err := DefaultCommandExecutor(ctx, "echo", "hello-test")
		require.NoError(t, err)
		assert.Contains(t, out, "hello-test")
	})

	t.Run("DefaultCommandExecutor returns error on invalid command", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		_, err := DefaultCommandExecutor(ctx, "non_existent_command_12345")
		require.Error(t, err)
	})
}
