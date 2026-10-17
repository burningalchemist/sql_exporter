package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/burningalchemist/sql_exporter"
	"github.com/prometheus/client_golang/prometheus"
)

const jobsConfig = `
jobs:
  - job_name: a
    collectors: [c]
    static_configs:
      - targets:
          ta: "postgres://127.0.0.1:1/db"
  - job_name: b
    collectors: [c]
    static_configs:
      - targets:
          tb: "postgres://127.0.0.1:1/db"
collectors:
  - collector_name: c
    metrics:
      - metric_name: m
        type: gauge
        help: m
        values: [v]
        query: SELECT 1 AS v
`

// Scrapes of different jobs that run at the same time must not see each other's jobs[] filter. The targets are
// unreachable on purpose: `up` and `scrape_errors_total` still carry the job label.
func TestExporterHandlerJobFilters(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "sql_exporter.yml")
	if err := os.WriteFile(cfgFile, []byte(jobsConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	registry := prometheus.NewRegistry()
	exporter, err := sql_exporter.NewExporter(cfgFile, registry)
	if err != nil {
		t.Fatal(err)
	}
	handler := ExporterHandlerFor(exporter, registry)

	var wg sync.WaitGroup
	for job, other := range map[string]string{"a": "b", "b": "a"} {
		wg.Go(func() {
			for range 10 {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics?jobs[]="+job, nil))
				body := rec.Body.String()
				if !strings.Contains(body, `job="`+job+`"`) || strings.Contains(body, `job="`+other+`"`) {
					t.Errorf("jobs[]=%s: got metrics of the wrong job:\n%s", job, body)
					return
				}
			}
		})
	}
	wg.Wait()
}
