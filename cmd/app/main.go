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

// Package main is the entry point of the Go template application.
// It initializes dependencies and runs the HTTP/HTTPS API server.
//
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.7.1 --config=../../api/oapi-codegen.cfg.yaml -o ../../ogen/api.gen.go ../../api/openapi.yaml
//go:generate go run -mod=mod entgo.io/ent/cmd/ent@v0.14.6 generate ../../ent/schema
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shjtmy/go_sh0jitmy_template/internal/database"
	"github.com/shjtmy/go_sh0jitmy_template/internal/web"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/crypto/acme/autocert"
)

// Version is the application version, automatically managed by tagpr.
const Version = "0.0.0"

// initOTel は OpenTelemetry SDK (TracerProvider と MeterProvider) を初期化します。
func initOTel() (*trace.TracerProvider, *metric.MeterProvider, error) {
	// 1. Prometheus Exporter の初期化 (MeterProvider用)
	exporter, err := otelprom.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(meterProvider)

	// 2. Simple TracerProvider (E2Eテストやローカル用) の初期化
	tp := trace.NewTracerProvider(trace.WithSampler(trace.AlwaysSample()))
	otel.SetTracerProvider(tp)

	return tp, meterProvider, nil
}

// startSecurePprof は localhost:6060 にバインドされた安全な pprof サーバーを起動します。
func startSecurePprof() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := &http.Server{
		Addr:              "127.0.0.1:6060",
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}

	go func() {
		slog.Info("Starting secure pprof server on localhost...", "address", "127.0.0.1:6060")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("pprof server failed to run", "error", err)
		}
	}()

	return srv
}

func main() {
	handler := web.NewSecureJSONHandler(os.Stdout)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	app := &cli.App{
		Name:    "go-template-app",
		Usage:   "A secure and production-ready Go project template",
		Version: Version,
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Start the secure HTTP/HTTPS API server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Value:   "8080",
						Usage:   "HTTP Port to listen on (ignored if --tls-domain is set)",
					},
					&cli.StringFlag{
						Name:  "tls-domain",
						Usage: "Enable Let's Encrypt / autocert for this domain",
					},
					&cli.StringFlag{
						Name:  "tls-cache",
						Value: "certs",
						Usage: "Let's Encrypt certificates cache directory",
					},
					&cli.StringFlag{
						Name:  "tls-cert",
						Usage: "Path to local SSL certificate file (for local HTTPS test)",
					},
					&cli.StringFlag{
						Name:  "tls-key",
						Usage: "Path to local SSL key file (for local HTTPS test)",
					},
				},
				Action: func(c *cli.Context) error {
					return runServer(c.Context, c)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("Application terminated with error", "error", err)
		os.Exit(1)
	}
}

func runServer(ctx context.Context, c *cli.Context) error {
	// 0. OpenTelemetry と pprof サーバーの初期化
	tp, mp, err := initOTel()
	if err != nil {
		return fmt.Errorf("failed to init OTel: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if shutdownErr := tp.Shutdown(shutdownCtx); shutdownErr != nil {
			slog.Warn("Failed to shutdown OTel TracerProvider", "error", shutdownErr)
		}
		if shutdownErr := mp.Shutdown(shutdownCtx); shutdownErr != nil {
			slog.Warn("Failed to shutdown OTel MeterProvider", "error", shutdownErr)
		}
	}()

	pprofSrv := startSecurePprof()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if shutdownErr := pprofSrv.Shutdown(shutdownCtx); shutdownErr != nil {
			slog.Warn("Failed to shutdown pprof server", "error", shutdownErr)
		}
	}()

	dbDriver := "sqlite"
	dbDSN := "ent.db"
	if os.Getenv("DATABASE_DRIVER") != "" {
		dbDriver = os.Getenv("DATABASE_DRIVER")
	}
	if os.Getenv("DATABASE_URL") != "" {
		dbDSN = os.Getenv("DATABASE_URL")
		if strings.HasPrefix(dbDSN, "postgres://") || strings.HasPrefix(dbDSN, "postgresql://") {
			dbDriver = "postgres"
		}
	}

	// 1. データベースクライアントの初期化 (DI)
	dbClient, err := database.NewClient(ctx, dbDriver, dbDSN)
	if err != nil {
		return fmt.Errorf("database init failed: %w", err)
	}
	defer func() {
		if closeErr := dbClient.Close(); closeErr != nil {
			slog.Warn("Failed to close DB client", "error", closeErr)
		}
	}()

	// 2. シードデータの投入
	if err := database.SeedAdminUser(ctx, dbClient); err != nil {
		return fmt.Errorf("database seed failed: %w", err)
	}

	// 3. APIエンジンのセットアップ (DI)
	r := web.SetupEngine(dbClient)

	tlsDomain := c.String("tls-domain")
	tlsCert := c.String("tls-cert")
	tlsKey := c.String("tls-key")

	var srv *http.Server

	if tlsDomain != "" {
		cacheDir := c.String("tls-cache")
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(tlsDomain),
			Cache:      autocert.DirCache(cacheDir),
		}

		// HTTPSリダイレクタ
		go func() {
			slog.Info("Starting HTTP-to-HTTPS redirector...", "port", "80")
			redirectEngine := gin.New()
			redirectEngine.Use(web.HTTPSRedirectMiddleware())
			redirectSrv := &http.Server{
				Addr:              ":80",
				Handler:           redirectEngine,
				ReadHeaderTimeout: 5 * time.Second,
			}
			if err := redirectSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("Failed to run HTTP redirector", "error", err)
			}
		}()

		srv = &http.Server{
			Addr:         ":443",
			Handler:      r,
			TLSConfig:    m.TLSConfig(),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		go func() {
			slog.Info("Starting secure HTTPS server with autocert...", "domain", tlsDomain)
			if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				slog.Error("HTTPS server failed", "error", err)
			}
		}()

	} else if tlsCert != "" && tlsKey != "" {
		port := c.String("port")
		srv = &http.Server{
			Addr:         ":" + port,
			Handler:      r,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		go func() {
			slog.Info("Starting local HTTPS server...", "port", port)
			if err := srv.ListenAndServeTLS(tlsCert, tlsKey); err != nil && err != http.ErrServerClosed {
				slog.Error("Local HTTPS server failed", "error", err)
			}
		}()

	} else {
		port := c.String("port")
		srv = &http.Server{
			Addr:         ":" + port,
			Handler:      r,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		go func() {
			slog.Info("Starting HTTP server...", "port", port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("HTTP server failed", "error", err)
			}
		}()
	}

	<-ctx.Done()
	slog.Info("Shutting down API server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
