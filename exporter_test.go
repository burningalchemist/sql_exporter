package sql_exporter

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/burningalchemist/sql_exporter/config"
	_ "github.com/mithrandie/csvq-driver"
	"github.com/prometheus/client_golang/prometheus"
)

func TestTrimMissingCtx(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"no leading separator", "job=j,collector=c", "job=j,collector=c"},
		{"leading comma", ",collector=c", "collector=c"},
		{"leading comma and space", ", collector=c", "collector=c"},
		{"leading comma, several items", ",collector=c,metric=m", "collector=c,metric=m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimMissingCtx(tt.in); got != tt.want {
				t.Errorf("TrimMissingCtx(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// In single target mode there is no job or target name, so the parent log context is empty. The context of an error
// must not start with a separator, otherwise parseContextLog warns about an empty item on every scrape error.
func TestSingleTargetErrorContext(t *testing.T) {
	cc := &config.CollectorConfig{
		Name:    "c",
		Metrics: []*config.MetricConfig{{Name: "m"}}, // no values: NewMetricFamily fails
	}
	_, err := NewCollector("", cc, nil, false)
	if err == nil {
		t.Fatal("NewCollector: expected an error for a metric without values")
	}
	if got, want := err.Context(), "collector=c,metric=m"; got != want {
		t.Errorf("error context = %q, want %q", got, want)
	}

	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	want := map[string]string{"collector": "c", "metric": "m"}
	if got := parseContextLog(err.Context()); !maps.Equal(got, want) {
		t.Errorf("parseContextLog(%q) = %v, want %v", err.Context(), got, want)
	}
	if logs.Len() != 0 {
		t.Errorf("parseContextLog logged: %s", logs.String())
	}
}

// Reload keeps the connections of the running exporter, so it cannot switch between single target and jobs mode. It
// must report that in both directions instead of succeeding while the old targets stay in place.
func TestReloadScrapeModeChange(t *testing.T) {
	const (
		collectors = `
collectors:
- collector_name: c
  metrics:
  - metric_name: m
    type: gauge
    help: h
    values: [v]
    query: SELECT 1 AS v
`
		target = `
target:
  data_source_name: "csvq:data"
  collectors: [c]
`
		jobs = `
jobs:
- job_name: j
  collectors: [c]
  static_configs:
  - targets:
      t: "csvq:data"
`
	)
	tests := []struct {
		name     string
		from, to string
	}{
		{"TargetToJobs", target, jobs},
		{"JobsToTarget", jobs, target},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			current, next := filepath.Join(dir, "current.yml"), filepath.Join(dir, "next.yml")
			for file, content := range map[string]string{current: tt.from, next: tt.to} {
				if err := os.WriteFile(file, []byte(content+collectors), 0o644); err != nil {
					t.Fatalf("write config: %v", err)
				}
			}
			e, err := NewExporter(current, prometheus.NewRegistry())
			if err != nil {
				t.Fatalf("NewExporter: %v", err)
			}
			err = Reload(e, &next)
			if err == nil || !strings.Contains(err.Error(), "changing scrape mode") {
				t.Errorf("Reload = %v, want a scrape mode change error", err)
			}
		})
	}
}

// A reload that fails keeps the old targets serving, so the config that /config and the jobs[] filter read must stay
// the old one too. Every exit of Reload that fails after the new config was read has a row, and every next config also
// changes a global setting, so applying the globals before the reload has succeeded shows in the comparison.
func TestReloadFailureKeepsConfig(t *testing.T) {
	// The config load lets these variables override the file, so an ambient value would hide the changed global setting.
	t.Setenv("SQLEXPORTER_GLOBAL_SCRAPE_TIMEOUT", "")
	t.Setenv("SQLEXPORTER_GLOBAL_ENABLE_QUERY_METRICS", "")

	const (
		collectors = `
collectors:
- collector_name: c
  metrics:
  - metric_name: m
    type: gauge
    help: h
    values: [v]
    query: SELECT 1 AS v
`
		// The column v is both a key and a value: Load accepts it, creating the target does not.
		badCollectors = collectors + "    key_labels: [v]\n"
		target        = `
target:
  data_source_name: "csvq:data"
  collectors: [c]
`
		jobs = `
jobs:
- job_name: j
  collectors: [c]
  static_configs:
  - targets:
      t: "csvq:data"
`
		globals = "global:\n  scrape_timeout: 20s\n"
	)
	tests := []struct {
		name     string
		from, to string
	}{
		{"TargetFails", target + collectors, target + "  name: n\n" + badCollectors},
		{"JobsFail", jobs + collectors, strings.Replace(jobs, "job_name: j", "job_name: k", 1) + badCollectors},
		{"ScrapeModeChange", target + collectors, jobs + badCollectors},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			current, next := filepath.Join(dir, "current.yml"), filepath.Join(dir, "next.yml")
			for file, content := range map[string]string{current: tt.from, next: globals + tt.to} {
				if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
					t.Fatalf("write config: %v", err)
				}
			}
			e, err := NewExporter(current, prometheus.NewRegistry())
			if err != nil {
				t.Fatalf("NewExporter: %v", err)
			}
			before, err := e.Config().YAML()
			if err != nil {
				t.Fatalf("YAML: %v", err)
			}
			if err := Reload(e, &next); err == nil {
				t.Fatal("Reload: expected an error")
			}
			after, err := e.Config().YAML()
			if err != nil {
				t.Fatalf("YAML: %v", err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("config after a failed reload:\n%s\nwant the config from before:\n%s", after, before)
			}
		})
	}
}

// The global settings can be reloaded. The scrape timeout is read from the config on every scrape and the query
// metrics are set up when the targets are recreated, so a reload must hand the new globals to both.
func TestReloadAppliesGlobals(t *testing.T) {
	// The flag sets the target label of the jobs, the default of the package is empty.
	prev := config.TargetLabel
	config.TargetLabel = "target"
	t.Cleanup(func() { config.TargetLabel = prev })
	// The config load lets these variables override the file, so an ambient value would change what the rows reload.
	t.Setenv("SQLEXPORTER_GLOBAL_SCRAPE_TIMEOUT", "")
	t.Setenv("SQLEXPORTER_GLOBAL_ENABLE_QUERY_METRICS", "")

	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(data, "t.csv"), []byte("v\n1\n"), 0o644); err != nil {
		t.Fatalf("write table: %v", err)
	}
	const collectors = `
collectors:
- collector_name: c
  metrics:
  - metric_name: m
    type: gauge
    help: h
    values: [v]
    query: SELECT v FROM t
`
	tests := []struct {
		name, scrape string
	}{
		{"Target", fmt.Sprintf("target:\n  data_source_name: \"csvq:%s\"\n  collectors: [c]\n", data)},
		{"Jobs", fmt.Sprintf("jobs:\n- job_name: j\n  collectors: [c]\n  static_configs:\n  - targets:\n      t: \"csvq:%s\"\n", data)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			current, next := filepath.Join(dir, "current.yml"), filepath.Join(dir, "next.yml")
			configs := map[string]string{
				current: "global:\n  scrape_timeout: 10s\n" + tt.scrape + collectors,
				next:    "global:\n  scrape_timeout: 20s\n  enable_query_metrics: true\n" + tt.scrape + collectors,
			}
			for file, content := range configs {
				if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
					t.Fatalf("write config: %v", err)
				}
			}
			e, err := NewExporter(current, prometheus.NewRegistry())
			if err != nil {
				t.Fatalf("NewExporter: %v", err)
			}
			if err := Reload(e, &next); err != nil {
				t.Fatalf("Reload: %v", err)
			}
			t.Cleanup(func() { closeTargets(e.Targets()) })

			if got, want := time.Duration(e.Config().Globals.ScrapeTimeout), 20*time.Second; got != want {
				t.Errorf("scrape_timeout after reload = %s, want %s", got, want)
			}
			mfs, err := (prometheus.Gatherers{e}).Gather()
			if err != nil {
				t.Fatalf("Gather: %v", err)
			}
			found := false
			for _, mf := range mfs {
				found = found || mf.GetName() == queryDurationName
			}
			if !found {
				t.Errorf("no %s after enabling enable_query_metrics in a reload", queryDurationName)
			}
		})
	}
}

// A collector with min_interval repeats its cached metrics until the interval has passed. A query that failed must not
// be cached: the next scrape runs it again instead of repeating the error.
func TestMinIntervalDoesNotCacheQueryError(t *testing.T) {
	data := t.TempDir()
	cfg := filepath.Join(t.TempDir(), "sql_exporter.yml")
	err := os.WriteFile(cfg, []byte(fmt.Sprintf(`
target:
  data_source_name: "csvq:%s"
  collectors: [c]
collectors:
- collector_name: c
  min_interval: 1h
  metrics:
  - metric_name: m
    type: gauge
    help: h
    values: [v]
    query: SELECT v FROM t
`, data)), 0o644)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}
	e, err := NewExporter(cfg, prometheus.NewRegistry())
	if err != nil {
		t.Fatalf("NewExporter: %v", err)
	}
	t.Cleanup(func() {
		for _, target := range e.Targets() {
			_ = target.Close()
		}
	})

	// The table does not exist yet, so the query fails.
	if _, err := (prometheus.Gatherers{e}).Gather(); err == nil {
		t.Fatal("first scrape: expected the query to fail")
	}

	if err := os.WriteFile(filepath.Join(data, "t.csv"), []byte("v\n1\n"), 0o644); err != nil {
		t.Fatalf("write table: %v", err)
	}
	mfs, err := (prometheus.Gatherers{e}).Gather()
	if err != nil {
		t.Fatalf("second scrape: %v", err)
	}
	if len(mfs) != 1 || mfs[0].GetName() != "m" {
		t.Errorf("second scrape: got %v, want metric m", mfs)
	}
}

// PgBouncer rejects the driver's ping, so its targets set enable_ping: false. The ping that follows opening the
// connection must be skipped too, or the first scrape after a start or reload reports the target as down.
func TestPingDisabledSkipsFirstConnectionPing(t *testing.T) {
	tests := []struct {
		name       string
		enablePing bool
		wantErr    bool
	}{
		{"ping enabled", true, true},
		{"ping disabled", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The handle opens lazily and csvq refuses a missing directory only when it connects, as a ping does.
			dsn := "csvq:" + filepath.Join(t.TempDir(), "missing")
			tgt, err := NewTarget("", "", "", dsn, nil, nil, &config.GlobalConfig{}, &tt.enablePing)
			if err != nil {
				t.Fatalf("NewTarget: %v", err)
			}
			t.Cleanup(func() { _ = tgt.Close() })
			if err := tgt.(*target).ping(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("ping error = %v, want an error: %t", err, tt.wantErr)
			}
			if tgt.(*target).conn == nil {
				t.Error("connection was not kept")
			}
		})
	}
}

// The warmup runs the collectors one after another, warmup_delay apart. scrape_timeout limits each run, not the whole
// warmup: neither the delays nor a collector that runs into the timeout may leave the later collectors cold.
func TestWarmupDoesNotShareScrapeTimeout(t *testing.T) {
	tests := []struct {
		name    string
		delay   string
		timeout string
		rows    int       // rows in the first table, whose query joins it with itself, far slower than the timeout
		want    []float64 // m1 to m3 on the first scrape: 1 was cached by the warmup, 2 was read from the changed table
	}{
		{"delays add up past the timeout", "200ms", "300ms", 1, []float64{1, 1, 1}},
		{"first collector runs into the timeout", "10ms", "300ms", 5000, []float64{2, 1, 1}},
		{"zero scrape timeout", "10ms", "0s", 1, []float64{1, 1, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := t.TempDir()
			writeTables := func(rows ...int) {
				for i, n := range rows {
					table := filepath.Join(data, fmt.Sprintf("t%d.csv", i+1))
					if err := os.WriteFile(table, []byte("v\n"+strings.Repeat("1\n", n)), 0o644); err != nil {
						t.Fatalf("write table: %v", err)
					}
				}
			}
			writeTables(tt.rows, 1, 1)
			var cfg strings.Builder
			fmt.Fprintf(&cfg, "global:\n  scrape_timeout: %s\n  warmup_delay: %s\n", tt.timeout, tt.delay)
			fmt.Fprintf(&cfg, "target:\n  data_source_name: \"csvq:%s\"\n  collectors: [c1, c2, c3]\ncollectors:\n", data)
			for i := 1; i <= 3; i++ {
				fmt.Fprintf(&cfg, `- collector_name: c%[1]d
  min_interval: 1h
  metrics:
  - metric_name: m%[1]d
    type: gauge
    help: h
    values: [v]
    query: SELECT COUNT(*) AS v FROM t%[1]d a LEFT JOIN t%[1]d b ON a.v + b.v < 0
`, i)
			}
			file := filepath.Join(t.TempDir(), "sql_exporter.yml")
			if err := os.WriteFile(file, []byte(cfg.String()), 0o644); err != nil {
				t.Fatalf("write config: %v", err)
			}
			e, err := NewExporter(file, prometheus.NewRegistry())
			if err != nil {
				t.Fatalf("NewExporter: %v", err)
			}
			t.Cleanup(func() { closeTargets(e.Targets()) })

			globals := e.Config().Globals
			NewWarmup().Run(e.Targets(), time.Duration(globals.WarmupDelay), time.Duration(globals.ScrapeTimeout))

			writeTables(2, 2, 2)
			mfs, err := (prometheus.Gatherers{e}).Gather()
			if err != nil {
				t.Fatalf("scrape: %v", err)
			}
			if len(mfs) != len(tt.want) {
				t.Fatalf("got %d metric families, want %d", len(mfs), len(tt.want))
			}
			for i, mf := range mfs {
				if got := mf.GetMetric()[0].GetGauge().GetValue(); got != tt.want[i] {
					t.Errorf("%s = %v, want %v", mf.GetName(), got, tt.want[i])
				}
			}
		})
	}
}
