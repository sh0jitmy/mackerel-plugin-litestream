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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeDBKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"", "default"},
		{"/var/lib/myapp.db", "myapp_db"},
		{"db.sqlite3", "db_sqlite3"},
		{"/data/db/production-user_v2.db", "production-user_v2_db"},
		{"///", "db"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, SanitizeDBKey(tt.input))
		})
	}
}

func TestSanitizeKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"", "unknown"},
		{"PASSIVE", "PASSIVE"},
		{"AccessDenied", "AccessDenied"},
		{"too recent", "too_recent"},
		{"s3://my-bucket/prefix", "s3___my-bucket_prefix"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, SanitizeKey(tt.input))
		})
	}
}

func TestParseMetricsReader(t *testing.T) {
	t.Parallel()

	samplePrometheus := `
# HELP litestream_db_size Database file size in bytes
# TYPE litestream_db_size gauge
litestream_db_size{db="/var/lib/myapp.db"} 10485760
# HELP litestream_wal_size WAL file size in bytes
# TYPE litestream_wal_size gauge
litestream_wal_size{db="/var/lib/myapp.db"} 32768
# HELP litestream_sync_error_count Total sync errors
# TYPE litestream_sync_error_count counter
litestream_sync_error_count{db="/var/lib/myapp.db"} 2
# HELP litestream_disk_full Disk full indicator
# TYPE litestream_disk_full gauge
litestream_disk_full{db="/var/lib/myapp.db"} 0
# HELP litestream_replica_operation_total Total replica operations
# TYPE litestream_replica_operation_total counter
litestream_replica_operation_total{operation="PUT",replica_type="s3"} 150
`

	mf, err := ParseMetricsReader(strings.NewReader(samplePrometheus))
	require.NoError(t, err)
	require.NotNil(t, mf)

	assert.Contains(t, mf, "litestream_db_size")
	assert.Contains(t, mf, "litestream_wal_size")
	assert.Contains(t, mf, "litestream_sync_error_count")
	assert.Contains(t, mf, "litestream_disk_full")
	assert.Contains(t, mf, "litestream_replica_operation_total")

	dbSizeMetric := mf["litestream_db_size"].GetMetric()[0]
	assert.InDelta(t, float64(10485760), GetMetricValue(dbSizeMetric), 0.001)
	assert.Equal(t, "/var/lib/myapp.db", GetLabelValue(dbSizeMetric, "db"))
	assert.Empty(t, GetLabelValue(dbSizeMetric, "non_existent"))

	repMetric := mf["litestream_replica_operation_total"].GetMetric()[0]
	assert.InDelta(t, float64(150), GetMetricValue(repMetric), 0.001)
	assert.Equal(t, "s3", GetLabelValue(repMetric, "replica_type"))
	assert.Equal(t, "PUT", GetLabelValue(repMetric, "operation"))

	t.Run("invalid syntax returns error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseMetricsReader(strings.NewReader("INVALID { METRIC SYNTAX"))
		require.Error(t, err)
	})
}

func TestGetMetricValue(t *testing.T) {
	t.Parallel()

	val := 42.5

	t.Run("gauge value", func(t *testing.T) {
		t.Parallel()
		m := &dto.Metric{Gauge: &dto.Gauge{Value: &val}}
		assert.InDelta(t, val, GetMetricValue(m), 0.001)
	})

	t.Run("counter value", func(t *testing.T) {
		t.Parallel()
		m := &dto.Metric{Counter: &dto.Counter{Value: &val}}
		assert.InDelta(t, val, GetMetricValue(m), 0.001)
	})

	t.Run("untyped value", func(t *testing.T) {
		t.Parallel()
		m := &dto.Metric{Untyped: &dto.Untyped{Value: &val}}
		assert.InDelta(t, val, GetMetricValue(m), 0.001)
	})

	t.Run("nil or other value returns 0", func(t *testing.T) {
		t.Parallel()
		m := &dto.Metric{}
		assert.InDelta(t, float64(0), GetMetricValue(m), 0.001)
	})
}

func TestScrapePrometheusMetrics(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("successful scrape", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# TYPE foo gauge\nfoo 123\n"))
		}))
		defer ts.Close()

		mf, err := ScrapePrometheusMetrics(ctx, ts.URL, 2*time.Second)
		require.NoError(t, err)
		assert.Contains(t, mf, "foo")
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		_, err := ScrapePrometheusMetrics(ctx, ts.URL, 2*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status code")
	})

	t.Run("connection failure returns error", func(t *testing.T) {
		t.Parallel()
		_, err := ScrapePrometheusMetrics(ctx, "http://127.0.0.1:59998/metrics", 100*time.Millisecond)
		require.Error(t, err)
	})

	t.Run("invalid request url returns error", func(t *testing.T) {
		t.Parallel()
		_, err := ScrapePrometheusMetrics(ctx, "://invalid-url", 100*time.Millisecond)
		require.Error(t, err)
	})
}
