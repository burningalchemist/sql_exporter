//go:build integration

package sql_exporter

import (
	"fmt"
	"io"
	"log/slog"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/mithrandie/csvq-driver"
	"go.yaml.in/yaml/v3"

	"github.com/prometheus/client_golang/prometheus"
)

// setupCSVDirs creates a temp directory with a minimal CSV file usable as a table.
func setupCSVDirs(t *testing.T, n int) []string {
	t.Helper()
	base := t.TempDir()
	dirs := make([]string, n)
	for i := range dirs {
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

func writeConfig(t *testing.T, dirs []string) string {
	t.Helper()

	type metric struct {
		Name   string   `yaml:"metric_name"`
		Type   string   `yaml:"type"`
		Help   string   `yaml:"help"`
		Values []string `yaml:"values"`
		Query  string   `yaml:"query"`
	}
	type collector struct {
		Name    string   `yaml:"collector_name"`
		Metrics []metric `yaml:"metrics"`
	}
	type staticConfig struct {
		Targets map[string]string `yaml:"targets"`
	}
	type job struct {
		Name          string         `yaml:"job_name"`
		Collectors    []string       `yaml:"collectors"`
		StaticConfigs []staticConfig `yaml:"static_configs"`
	}
	type global struct {
		ScrapeTimeout       string `yaml:"scrape_timeout"`
		ScrapeTimeoutOffset string `yaml:"scrape_timeout_offset"`
		MinInterval         string `yaml:"min_interval"`
		MaxConnections      int    `yaml:"max_connections"`
		MaxIdleConnections  int    `yaml:"max_idle_connections"`
	}
	type cfg struct {
		Global     global      `yaml:"global"`
		Collectors []collector `yaml:"collectors"`
		Jobs       []job       `yaml:"jobs"`
	}

	n := len(dirs)
	collectors := make([]collector, n)
	collectorNames := make([]string, n)
	targets := make(map[string]string, n)

	for i := range dirs {
		name := fmt.Sprintf("col%d", i)
		collectorNames[i] = name
		collectors[i] = collector{
			Name: name,
			Metrics: []metric{{
				Name:   fmt.Sprintf("csvq_value_%d", i),
				Type:   "gauge",
				Help:   fmt.Sprintf("test metric %d", i),
				Values: []string{"value"},
				Query:  "SELECT value FROM metrics",
			}},
		}
		targets[fmt.Sprintf("target%d", i)] = "csvq:" + dirs[i]
	}

	c := cfg{
		Global: global{
			ScrapeTimeout:       "10s",
			ScrapeTimeoutOffset: "500ms",
			MinInterval:         "0s",
			MaxConnections:      3,
			MaxIdleConnections:  3,
		},
		Collectors: collectors,
		Jobs: []job{{
			Name:          "test_job",
			Collectors:    collectorNames,
			StaticConfigs: []staticConfig{{Targets: targets}},
		}},
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	cfgFile := filepath.Join(t.TempDir(), "sql_exporter.yml")
	if err := os.WriteFile(cfgFile, data, 0o644); err != nil {
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
	cfgFile := writeConfig(t, dirs)

	e, err := NewExporter(cfgFile, prometheus.NewRegistry())
	if err != nil {
		t.Fatalf("NewExporter: %v", err)
	}

	delta := runReloadCycles(t, e, cfgFile, numCycles)

	t.Logf("goroutine delta=%d (expected <= %d)", delta, tolerance)
	if delta > tolerance {
		t.Errorf("expected goroutine delta <= %d, got %d — leak suspected", tolerance, delta)
	}
}

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}
