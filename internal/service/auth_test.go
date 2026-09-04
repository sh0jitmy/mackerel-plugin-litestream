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

package service

import (
	"context"
	"testing"

	"github.com/shjtmy/go_sh0jitmy_template/ent/enttest"
	_ "github.com/shjtmy/go_sh0jitmy_template/internal/database"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Authenticate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// テストごとに一意のメモリキャッシュ識別名（enttest_auth）を使用し、並行テスト間の干渉を防ぎます
	client := enttest.Open(t, "sqlite3", "file:enttest_auth?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	t.Cleanup(func() {
		_ = client.Close()
	})

	// テスト用パスワードのハッシュ化とユーザーの投入
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	_, err = client.User.Create().
		SetUsername("testuser").
		SetPasswordHash(string(hashedPassword)).
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	svc := NewAuthService(client)

	// ケース1: 正常に認証成功
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		token, err := svc.Authenticate(ctx, "testuser", "correct-password")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if token != "secret-bearer-token" {
			t.Errorf("expected secret-bearer-token, got %s", token)
		}
	})

	// ケース2: 存在しないユーザー名での認証失敗
	t.Run("UserNotFound", func(t *testing.T) {
		t.Parallel()
		token, err := svc.Authenticate(ctx, "nonexistent", "correct-password")
		if err != ErrAuthenticationFailed {
			t.Errorf("expected ErrAuthenticationFailed, got %v", err)
		}
		if token != "" {
			t.Errorf("expected empty token, got %s", token)
		}
	})

	// ケース3: パスワード不一致での認証失敗
	t.Run("PasswordMismatch", func(t *testing.T) {
		t.Parallel()
		token, err := svc.Authenticate(ctx, "testuser", "wrong-password")
		if err != ErrAuthenticationFailed {
			t.Errorf("expected ErrAuthenticationFailed, got %v", err)
		}
		if token != "" {
			t.Errorf("expected empty token, got %s", token)
		}
	})
}

func TestAuthService_GetUserByUsername(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// テストごとに一意のメモリキャッシュ識別名（enttest_me）を使用し、並行テスト間の干渉を防ぎます
	client := enttest.Open(t, "sqlite3", "file:enttest_me?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	t.Cleanup(func() {
		_ = client.Close()
	})

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	u, err := client.User.Create().
		SetUsername("me").
		SetPasswordHash(string(hashedPassword)).
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	svc := NewAuthService(client)

	// ケース1: ユーザー取得成功
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		res, err := svc.GetUserByUsername(ctx, "me")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if res.ID != u.ID || res.Username != u.Username {
			t.Errorf("returned user does not match created user")
		}
	})

	// ケース2: ユーザー取得失敗
	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		res, err := svc.GetUserByUsername(ctx, "someone")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
		if res != nil {
			t.Errorf("expected nil user, got %v", res)
		}
	})
}
