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
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

var invalidMetricCharRegex = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)

// SanitizeDBKey はデータベースパスを Mackerel のメトリックキーとして安全な文字列に変換します。
// 例: "/var/lib/app.db" -> "app_db" (または "app")
func SanitizeDBKey(dbPath string) string {
	if dbPath == "" {
		return "default"
	}
	base := filepath.Base(dbPath)
	sanitized := invalidMetricCharRegex.ReplaceAllString(base, "_")
	sanitized = strings.Trim(sanitized, "_")
	if sanitized == "" {
		return "db"
	}
	return sanitized
}

// SanitizeKey は任意のラベル値を安全なキー文字列に変換します。
func SanitizeKey(val string) string {
	sanitized := invalidMetricCharRegex.ReplaceAllString(val, "_")
	sanitized = strings.Trim(sanitized, "_")
	if sanitized == "" {
		return "unknown"
	}
	return sanitized
}

// ScrapePrometheusMetrics は HTTP エンドポイントから Prometheus メトリクスを取得し、パースした MetricFamily マップを返します。
func ScrapePrometheusMetrics(ctx context.Context, url string, timeout time.Duration) (map[string]*dto.MetricFamily, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape metrics from %s: %w", url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from %s: %d", url, resp.StatusCode)
	}

	return ParseMetricsReader(resp.Body)
}

// ParseMetricsReader は Reader から Prometheus テキスト形式のメトリクスをパースします。
func ParseMetricsReader(r io.Reader) (map[string]*dto.MetricFamily, error) {
	parser := expfmt.NewTextParser(model.LegacyValidation)
	metricFamilies, err := parser.TextToMetricFamilies(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prometheus metrics: %w", err)
	}
	return metricFamilies, nil
}

// GetLabelValue は Metric 内から指定された名前のラベル値を取得します。
func GetLabelValue(m *dto.Metric, name string) string {
	for _, pair := range m.GetLabel() {
		if pair.GetName() == name {
			return pair.GetValue()
		}
	}
	return ""
}

// GetMetricValue は Gauge, Counter, Untyped から float64 の数値を取得します。
func GetMetricValue(m *dto.Metric) float64 {
	if m.GetGauge() != nil {
		return m.GetGauge().GetValue()
	}
	if m.GetCounter() != nil {
		return m.GetCounter().GetValue()
	}
	if m.GetUntyped() != nil {
		return m.GetUntyped().GetValue()
	}
	return 0
}
