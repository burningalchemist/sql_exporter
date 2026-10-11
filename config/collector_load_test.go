package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
