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

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	// Import glebarez driver to register it for sqlite.
	entsql "entgo.io/ent/dialect/sql"
	"github.com/shjtmy/go_sh0jitmy_template/ent"
	"github.com/shjtmy/go_sh0jitmy_template/internal/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.uber.org/goleak"
	"golang.org/x/crypto/bcrypt"
)

// TestMain はテスト全体の実行前後にフック処理を行います。
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// setupTestDB はインメモリ SQLite データベースを初期化し、スキーマ生成とシード投入を行います。
func setupTestDB(t *testing.T) *ent.Client {
	t.Helper()

	// 1. 手動で sql.Open を実行
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	require.NoError(t, err)

	// 2. ent.NewClient でクライアントをラップ
	drv := entsql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(drv))

	ctx := context.Background()
	// スキーマ生成
	err = client.Schema.Create(ctx)
	require.NoError(t, err)

	// テスト用のシードデータ投入
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("seed-password"), bcrypt.DefaultCost)
	require.NoError(t, err)

	_, err = client.User.Create().
		SetUsername("seed-user").
		SetPasswordHash(string(hashedPassword)).
		Save(ctx)
	require.NoError(t, err)

	// admin ユーザーも登録 (E2Eテスト時の /users/me 取得検証用)
	adminHashed, err := bcrypt.GenerateFromPassword([]byte("admin-pass"), bcrypt.DefaultCost)
	require.NoError(t, err)

	_, err = client.User.Create().
		SetUsername("admin").
		SetPasswordHash(string(adminHashed)).
		Save(ctx)
	require.NoError(t, err)

	return client
}

// TestE2E_AppAPI は OpenAPI + ent + SQLite + Bearer 認証が統合された E2E テストです。
func TestE2E_AppAPI(t *testing.T) {
	t.Parallel()

	// 0. テスト環境用の OTel および pprof の初期化
	tp, mp, err := initOTel()
	require.NoError(t, err)
	defer func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	}()

	pprofSrv := startSecurePprof()
	defer func() {
		_ = pprofSrv.Shutdown(context.Background())
	}()

	// 1. テストデータベース (SQLite メモリ) の準備
	dbClient := setupTestDB(t)
	defer func() {
		if closeErr := dbClient.Close(); closeErr != nil {
			t.Logf("Failed to close test DB: %v", closeErr)
		}
	}()

	// 2. ログキャプチャ用のハンドラーを初期化し、グローバルロガーに設定
	var logBuf bytes.Buffer
	handler := web.NewSecureJSONHandler(&logBuf)
	slog.SetDefault(slog.New(handler))

	// 3. APIエンジンの構成
	r := web.SetupEngine(dbClient)

	// 4. 認証失敗の検証 (無効な資格情報での /login)
	{
		payload := map[string]interface{}{
			"username": "seed-user",
			"password": "wrong-password",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/v1/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}

	// 5. 認証成功の検証 (正しい資格情報での /login と Bearerトークン返却、OTelトレース紐付けと監査ロギングの検証)
	var bearerToken string
	{
		logBuf.Reset()

		// OTel のスパンを開始
		tracer := otel.Tracer("test-tracer")
		ctx, span := tracer.Start(context.Background(), "test-span")

		payload := map[string]interface{}{
			"username": "seed-user",
			"password": "seed-password",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "/v1/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		span.End()

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		bearerToken = resp["token"]
		assert.NotEmpty(t, bearerToken)

		// ログへの機密情報の漏洩チェックおよび OTel トレース ID、監査タグの検証
		logLines := bytes.Split(logBuf.Bytes(), []byte("\n"))
		var foundAudit = false
		for _, line := range logLines {
			if len(line) == 0 {
				continue
			}
			var loggedData map[string]interface{}
			err = json.Unmarshal(line, &loggedData)
			if err == nil {
				if loggedData["msg"] == "Login attempt received" {
					// パスワードのマスキング検証
					assert.Equal(t, "[REDACTED]", loggedData["password"])
					// 監査タグの検証
					assert.Equal(t, "audit", loggedData["log_type"])
					foundAudit = true

					// OTelトレース情報の埋め込み検証
					assert.NotEmpty(t, loggedData["trace_id"])
					assert.Equal(t, span.SpanContext().TraceID().String(), loggedData["trace_id"])
					assert.Equal(t, span.SpanContext().SpanID().String(), loggedData["span_id"])
				}
			}
		}
		assert.True(t, foundAudit, "Audit log not found in log output")
	}

	// 6. Bearer 認証なしでの /v1/users/me アクセス (401エラー)
	{
		req, _ := http.NewRequest(http.MethodGet, "/v1/users/me", nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}

	// 7. 不正な Bearer トークンでの /v1/users/me アクセス (401エラー)
	{
		req, _ := http.NewRequest(http.MethodGet, "/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-value")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}

	// 8. 正しい Bearer トークンでの /v1/users/me アクセス (200成功)
	{
		req, _ := http.NewRequest(http.MethodGet, "/v1/users/me", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "admin", resp["username"])
	}

	// 9. Prometheus メトリクスエンドポイントの検証
	{
		req, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		metricsOutput := w.Body.String()

		// メトリクス定義が Prometheus 形式で含まれていることの確認 (OTel 計装)
		assert.Contains(t, metricsOutput, "http_requests_total")
		assert.Contains(t, metricsOutput, "http_request_duration_seconds")
	}

	// 10. pprof エンドポイントの検証 (localhostバインド)
	{
		resp, err := http.Get("http://127.0.0.1:6060/debug/pprof/")
		require.NoError(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
}
