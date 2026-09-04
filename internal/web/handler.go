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

package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// SecretString はログ出力時に自動でマスキングされる文字列型です
type SecretString string

// LogValue は slog がログをシリアライズする際に呼び出され、値を隠蔽します
func (s SecretString) LogValue() slog.Value {
	return slog.StringValue("[REDACTED]")
}

// HashableSecret は衝突確認用にSalt付きハッシュ値を出力する型です
type HashableSecret string

// LogValue は slog がログをシリアライズする際に呼び出され、Salt付きハッシュ値を返します
func (h HashableSecret) LogValue() slog.Value {
	salt := []byte("default_system_salt")
	hasher := sha256.New()
	hasher.Write(salt)
	hasher.Write([]byte(h))
	return slog.StringValue(hex.EncodeToString(hasher.Sum(nil)))
}

// OtelSlogHandler はコンテキスト内の OpenTelemetry スパン情報を slog に自動挿入するデコレーターです。
type OtelSlogHandler struct {
	parent slog.Handler
}

// Enabled は parent.Enabled を委譲します。
func (h *OtelSlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.parent.Enabled(ctx, level)
}

// Handle はコンテキストから trace_id/span_id を抽出し、slog レコードに追加してから parent.Handle を呼び出します。
func (h *OtelSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx == nil {
		return h.parent.Handle(ctx, r)
	}
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", spanContext.TraceID().String()),
			slog.String("span_id", spanContext.SpanID().String()),
		)
	}
	return h.parent.Handle(ctx, r)
}

// WithAttrs は parent.WithAttrs を委譲します。
func (h *OtelSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &OtelSlogHandler{parent: h.parent.WithAttrs(attrs)}
}

// WithGroup は parent.WithGroup を委譲します。
func (h *OtelSlogHandler) WithGroup(name string) slog.Handler {
	return &OtelSlogHandler{parent: h.parent.WithGroup(name)}
}

// NewSecureJSONHandler は機密情報を動的にマスキングし、かつ OTel トレース情報を自動付与する slog.Handler を作成します。
func NewSecureJSONHandler(w io.Writer) slog.Handler {
	jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			keyLower := strings.ToLower(a.Key)
			if keyLower == "password" || keyLower == "token" || keyLower == "secret" || keyLower == "authorization" || keyLower == "api_key" || keyLower == "apikey" {
				a.Value = slog.StringValue("[REDACTED]")
			}
			return a
		},
	})
	return &OtelSlogHandler{parent: jsonHandler}
}
