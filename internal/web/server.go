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
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shjtmy/go_sh0jitmy_template/ent"
	"github.com/shjtmy/go_sh0jitmy_template/internal/service"
	"github.com/shjtmy/go_sh0jitmy_template/ogen"
)

// Server は OpenAPI の ogen.ServerInterface を実装する構造体です。
// コンパイル時にインターフェース準拠を保証します。
var _ ogen.ServerInterface = (*Server)(nil)

type Server struct {
	authService *service.AuthService
}

// NewServer は API サーバーハンドラーの新しいインスタンスを返します。
func NewServer(db *ent.Client) *Server {
	return &Server{
		authService: service.NewAuthService(db),
	}
}

// SetupEngine は Gin エンジンを構成し、ミドルウェアおよびハンドラーを登録します。
func SetupEngine(db *ent.Client) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// HSTSミドルウェアを全体に適用
	r.Use(HSTSSetMiddleware())

	// OTelメトリクス収集ミドルウェアを全体に適用
	r.Use(OTelMetricsMiddleware())

	s := NewServer(db)

	// メトリクススクレイプ用エンドポイント（Prometheus形式でエクスポート）
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 手動マッピング（OpenAPI自動生成された ogen.ServerInterface 実装のバインド）
	r.POST("/v1/login", func(c *gin.Context) {
		s.Login(c)
	})

	authorized := r.Group("/v1")
	authorized.Use(BearerAuthMiddleware("secret-bearer-token"))
	authorized.GET("/users/me", func(c *gin.Context) {
		s.GetMe(c)
	})

	return r
}

// Login は POST /login エンドポイントの実装です。
func (s *Server) Login(c *gin.Context) {
	var req struct {
		Username string       `json:"username" binding:"required"`
		Password SecretString `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// 安全なロギングおよび監査ログの検証 (log_type: audit)
	slog.InfoContext(ctx, "Login attempt received",
		slog.String("log_type", "audit"),
		slog.String("username", req.Username),
		slog.Any("password", req.Password),
	)

	token, err := s.authService.Authenticate(ctx, req.Username, string(req.Password))
	if err != nil {
		slog.WarnContext(ctx, "Authentication failed", "username", req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	slog.InfoContext(ctx, "Successfully authenticated user",
		slog.String("log_type", "audit"),
		slog.String("username", req.Username),
	)
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// GetMe は GET /users/me エンドポイントの実装です。
func (s *Server) GetMe(c *gin.Context) {
	username, exists := c.Get("authenticated_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx := c.Request.Context()
	u, err := s.authService.GetUserByUsername(ctx, username.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       u.ID,
		"username": u.Username,
	})
}
