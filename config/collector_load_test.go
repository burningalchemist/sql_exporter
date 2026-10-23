package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func validCollectorYAML(name string) string {
	return "collector_name: " + name + "\n" +
		"metrics:\n" +
		"  - metric_name: " + name + "_metric\n" +
		"    type: gauge\n" +
		"    help: test metric\n" +
		"    values: [v]\n" +
		"    query: SELECT 1 AS v\n"
}

func TestLoadCollectorFilesSkipsBrokenUnreferencedFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pricing.collector.yml"), validCollectorYAML("pricing"))
	// Invalid YAML that a greedy glob would also match.
	writeFile(t, filepath.Join(dir, "orders.collector.yml"), "collector_name: \"bad\\xescape\"\n")

	cfgPath := filepath.Join(dir, "sql_exporter.yml")
	writeFile(t, cfgPath, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [pricing]
collector_files:
  - "*.collector.yml"
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() should skip broken unreferenced collector: %v", err)
	}
	if len(cfg.Collectors) != 1 || cfg.Collectors[0].Name != "pricing" {
		t.Fatalf("expected only pricing collector, got %+v", cfg.Collectors)
	}
}

func TestLoadCollectorFilesFailsBrokenReferencedFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pricing.collector.yml"), "collector_name: \"bad\\xescape\"\n")

	cfgPath := filepath.Join(dir, "sql_exporter.yml")
	writeFile(t, cfgPath, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [pricing]
collector_files:
  - "*.collector.yml"
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("Load() should fail for broken referenced collector")
	}
	if !strings.Contains(err.Error(), "pricing.collector.yml") {
		t.Fatalf("expected error to mention pricing.collector.yml, got: %v", err)
	}
}

func TestLoadCollectorFilesFailsBrokenFileMatchingGlobRef(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pricing.collector.yml"), validCollectorYAML("pricing"))
	writeFile(t, filepath.Join(dir, "pricing_extra.collector.yml"), "collector_name: \"bad\\xescape\"\n")

	cfgPath := filepath.Join(dir, "sql_exporter.yml")
	writeFile(t, cfgPath, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [pricing*]
collector_files:
  - "*.collector.yml"
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("Load() should fail when broken file matches a collector glob ref")
	}
}

func TestCollectorIdentities(t *testing.T) {
	names := collectorIdentities("/etc/collectors/pricing.collector.yml", []byte("collector_name: pricing\n"))
	if len(names) < 2 || names[0] != "pricing" {
		t.Fatalf("expected pricing identity first, got %v", names)
	}

	names = collectorIdentities("/etc/collectors/orders.collector.yml", []byte("not: valid: yaml: [[["))
	found := false
	for _, n := range names {
		if n == "orders" || n == "orders.collector" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected filename stem identity for invalid YAML, got %v", names)
	}
}

func TestCollectorQueryRefHonorsNoPreparedStatement(t *testing.T) {
	var cc CollectorConfig
	if err := yaml.Unmarshal([]byte(`
collector_name: test
metrics:
- {metric_name: unprepared, type: gauge, help: h, values: [v], query_ref: q, no_prepared_statement: true}
- {metric_name: shared, type: gauge, help: h, values: [v], query_ref: q}
queries:
- {query_name: q, query: SELECT 1 AS v}
`), &cc); err != nil {
		t.Fatalf("unmarshal collector: %v", err)
	}
	for _, metric := range cc.Metrics {
		t.Run(metric.Name, func(t *testing.T) {
			if !metric.Query().NoPreparedStatement {
				t.Error("query no_prepared_statement = false, want true")
			}
		})
	}
}

func TestMetricStaticLabelCollisions(t *testing.T) {
	tests := []struct {
		name, labels, valueLabel, want string
	}{
		{"key_label", "{env: prod}", "", `duplicate label "env" (defined in both static_labels and key_labels) for metric "app_info"`},
		{"value_label", "{kind: prod}", "kind", `duplicate label "kind" (defined in both static_labels and value_label) for metric "app_info"`},
		{"sorted", "{kind: prod, env: prod}", "kind", `duplicate label "env" (defined in both static_labels and key_labels) for metric "app_info"`},
		{"distinct", "{region: west, service: app}", "kind", ""},
		{"empty_without_value_label", `{"": prod}`, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var metric MetricConfig
			err := yaml.Unmarshal([]byte(`
metric_name: app_info
type: gauge
help: App info
key_labels: [env]
value_label: `+tt.valueLabel+`
values: [v]
query: SELECT env, 1 AS v
static_labels: `+tt.labels), &metric)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("unmarshal metric: %v", err)
				}
			} else if err == nil || err.Error() != tt.want {
				t.Errorf("unmarshal metric = %v, want %q", err, tt.want)
			}
		})
	}
}
