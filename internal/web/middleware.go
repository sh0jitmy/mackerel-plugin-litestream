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

// Package web implements the Gin Web HTTP layer, custom middleware,
// and maps request routes to OpenAPI handlers.
package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("github.com/shjtmy/go_sh0jitmy_template/internal/web")

	// HTTP リクエスト総数カウンター
	// PromQL: sum(rate(http_requests_total[5m])) by (method, status, path)
	requestCounter, _ = meter.Int64Counter("http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)

	// HTTP リクエスト処理遅延ヒストグラム
	// PromQL: histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, path))
	requestDuration, _ = meter.Float64Histogram("http_request_duration_seconds",
		metric.WithDescription("Duration of HTTP requests in seconds"),
		metric.WithUnit("s"),
	)
)

// BearerAuthMiddleware は Bearer トークンを検証する認証ミドルウェアです。
func BearerAuthMiddleware(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be Bearer token"})
			return
		}

		token := parts[1]
		if token != expectedToken {
			slog.Warn("Bearer token validation failed", "provided_token", token)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// デモ用にユーザー名を "admin" と仮定してコンテキストに格納
		c.Set("authenticated_user", "admin")
		c.Next()
	}
}

// HTTPSRedirectMiddleware は HTTP リクエストを HTTPS にリダイレクトするミドルウェアです。
func HTTPSRedirectMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isHTTPS := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
		if !isHTTPS {
			target := "https://" + c.Request.Host + c.Request.RequestURI
			c.Redirect(http.StatusMovedPermanently, target)
			c.Abort()
			return
		}
		c.Next()
	}
}

// HSTSSetMiddleware は HSTS（Strict-Transport-Security）ヘッダーを付与するミドルウェアです。
func HSTSSetMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}

// OTelMetricsMiddleware は OpenTelemetry Metrics API を用いて HTTP リクエストを計装する Gin ミドルウェアです。
func OTelMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		// カーディナリティが爆発しないように、Ginの登録パス(例: /users/:id)を使用
		path := c.FullPath()
		if path == "/metrics" {
			return
		}
		if path == "" {
			path = "unknown"
		}
		method := c.Request.Method
		status := fmt.Sprintf("%d", c.Writer.Status())

		opt := metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("status", status),
			attribute.String("path", path),
		)

		ctx := c.Request.Context()
		requestCounter.Add(ctx, 1, opt)
		requestDuration.Record(ctx, duration, opt)
	}
}
