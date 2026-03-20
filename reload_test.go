//go:build custom

package sql_exporter

import (
	"fmt"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/mithrandie/csvq-driver"

	"github.com/prometheus/client_golang/prometheus"
)

// setupCSVDir creates a temp directory with a minimal CSV file usable as a table.
func setupCSVDirs(t *testing.T, n int) []string {
	t.Helper()
	base := t.TempDir()
	dirs := make([]string, n)
	for i := range n {
		dir := filepath.Join(base, fmt.Sprintf("csv_%d", i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir CSV dir %d: %v", i, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "metrics.csv"), []byte("value\n1\n"), 0o644); err != nil {
			t.Fatalf("write CSV %d: %v", i, err)
		}
		dirs[i] = dir
	}
	return dirs
}

// writeConfig writes a sql_exporter YAML config file pointing at csvDir with
// n targets, returning the config file path.
func writeConfig(t *testing.T, dirs []string, n int) string {
	t.Helper()

	var sb strings.Builder
	for i := range n {
		fmt.Fprintf(&sb, `
  - collector_name: col%d
    metrics:
      - metric_name: csvq_value_%d
        type: gauge
        help: "test metric %d"
        values: [value]
        query: "SELECT value FROM metrics"
`, i, i, i)
	}
	collectors := sb.String()

	sb.Reset()
	for i := range n {
		fmt.Fprintf(&sb, "        target%d: csvq:%s\n", i, dirs[i])
	}
	targets := sb.String()

	content := fmt.Sprintf(`
global:
  scrape_timeout: 10s
  scrape_timeout_offset: 500ms
  min_interval: 0s
  max_connections: 3
  max_idle_connections: 3

collector_files: []

collectors:%s

jobs:
  - job_name: test_job
    collectors: [col0, col1, col2, col3, col4, col5, col6, col7, col8, col9]
    static_configs:
    - targets:
%s`, collectors, targets)

	cfgFile := filepath.Join(t.TempDir(), "sql_exporter.yml")
	if err := os.WriteFile(cfgFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return cfgFile
}

func printMemStats(t *testing.T, label string) {
	t.Helper()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	t.Logf("[%s] HeapAlloc=%.2f MB  HeapObjects=%d",
		label, float64(ms.HeapAlloc)/1024/1024, ms.HeapObjects)
}

func runReloadCycles(t *testing.T, e Exporter, cfgFile string, numCycles int) int {
	t.Helper()

	printMemStats(t, "initial")
	initialGoroutines := runtime.NumGoroutine()
	t.Logf("initial goroutines: %d", initialGoroutines)

	for cycle := 1; cycle <= numCycles; cycle++ {
		for _, old := range e.Targets() {
			if err := old.Close(); err != nil {
				t.Logf("cycle %02d close error: %v", cycle, err)
			}
		}

		if err := Reload(e, &cfgFile); err != nil {
			t.Fatalf("cycle %02d Reload: %v", cycle, err)
		}

		if cycle%10 == 0 {
			runtime.GC()
			goroutines := runtime.NumGoroutine()
			printMemStats(t, fmt.Sprintf("cycle %02d", cycle))
			t.Logf("cycle %02d | goroutines: %d (+%d vs initial)",
				cycle, goroutines, goroutines-initialGoroutines)
		}
	}

	return runtime.NumGoroutine() - initialGoroutines
}

func TestReloadMemoryLeak(t *testing.T) {
	const (
		numTargets = 10
		numCycles  = 500
		tolerance  = 5
	)

	dirs := setupCSVDirs(t, numTargets)
	cfgFile := writeConfig(t, dirs, numTargets)

	e, err := NewExporter(cfgFile, prometheus.NewRegistry())
	if err != nil {
		t.Fatalf("NewExporter: %v", err)
	}

	delta := runReloadCycles(t, e, cfgFile, numCycles)

	t.Logf("goroutine delta=%d (expected <= %d)", delta, tolerance)
	if delta > tolerance {
		t.Errorf("expected goroutine delta <= %d, got %d — leak still present", tolerance, delta)
	}
}
