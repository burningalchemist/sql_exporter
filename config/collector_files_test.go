package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCollectorFile(t *testing.T, dir, name, collectorName string) string {
	t.Helper()
	content := "collector_name: " + collectorName + "\n" +
		"metrics:\n" +
		"  - metric_name: " + collectorName + "_metric\n" +
		"    type: gauge\n" +
		"    help: test metric\n" +
		"    values: [v]\n" +
		"    query: SELECT 1 AS v\n"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write collector file: %v", err)
	}
	return path
}

func writeMinimalConfig(t *testing.T, dir string, body string) string {
	t.Helper()
	path := filepath.Join(dir, "sql_exporter.yml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadCollectorsFromDirectoryPath(t *testing.T) {
	dir := t.TempDir()
	collectorsDir := filepath.Join(dir, "collectors")
	if err := os.Mkdir(collectorsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeCollectorFile(t, collectorsDir, "alpha.collector.yml", "alpha")
	writeCollectorFile(t, collectorsDir, "beta.yaml", "beta")
	// Non-YAML files in the directory must be ignored.
	if err := os.WriteFile(filepath.Join(collectorsDir, "readme.txt"), []byte("ignore me"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}

	cfgPath := writeMinimalConfig(t, dir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [alpha, beta]
collector_path: collectors
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Collectors) != 2 {
		t.Fatalf("expected 2 collectors, got %d", len(cfg.Collectors))
	}

	names := map[string]bool{}
	for _, c := range cfg.Collectors {
		names[c.Name] = true
	}
	if !names["alpha"] || !names["beta"] {
		t.Fatalf("unexpected collectors: %+v", names)
	}
}

func TestLoadCollectorsFromCollectorFilesDirectory(t *testing.T) {
	dir := t.TempDir()
	collectorsDir := filepath.Join(dir, "coll")
	if err := os.Mkdir(collectorsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeCollectorFile(t, collectorsDir, "one.collector.yml", "one")

	cfgPath := writeMinimalConfig(t, dir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [one]
collector_files:
  - coll
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Collectors) != 1 || cfg.Collectors[0].Name != "one" {
		t.Fatalf("unexpected collectors: %+v", cfg.Collectors)
	}
}

func TestCollectorPathOverrideFlag(t *testing.T) {
	dir := t.TempDir()
	fromConfig := filepath.Join(dir, "from_config")
	fromFlag := filepath.Join(dir, "from_flag")
	for _, d := range []string{fromConfig, fromFlag} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeCollectorFile(t, fromConfig, "config_only.yml", "config_only")
	writeCollectorFile(t, fromFlag, "flag_only.yml", "flag_only")

	cfgPath := writeMinimalConfig(t, dir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [flag_only]
collector_path: from_config
`)

	prev := CollectorPathOverride
	CollectorPathOverride = fromFlag
	t.Cleanup(func() { CollectorPathOverride = prev })

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Collectors) != 1 || cfg.Collectors[0].Name != "flag_only" {
		t.Fatalf("expected CLI path to override config path, got %+v", cfg.Collectors)
	}
}

func TestResolveCollectorFilesSkipsMainConfigInSameDirectory(t *testing.T) {
	dir := t.TempDir()
	writeCollectorFile(t, dir, "metrics.collector.yml", "metrics")

	cfgPath := writeMinimalConfig(t, dir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [metrics]
collector_path: .
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() should skip main config in collector_path directory: %v", err)
	}
	if len(cfg.Collectors) != 1 || cfg.Collectors[0].Name != "metrics" {
		t.Fatalf("unexpected collectors: %+v", cfg.Collectors)
	}
}

func TestResolveCollectorFilesAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	collectorsDir := filepath.Join(dir, "abs_collectors")
	if err := os.Mkdir(collectorsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeCollectorFile(t, collectorsDir, "abs.collector.yml", "abs")

	// Config lives elsewhere; collector_path is absolute.
	cfgDir := filepath.Join(dir, "etc")
	if err := os.Mkdir(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgPath := writeMinimalConfig(t, cfgDir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [abs]
collector_path: `+collectorsDir+`
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Collectors) != 1 || cfg.Collectors[0].Name != "abs" {
		t.Fatalf("unexpected collectors: %+v", cfg.Collectors)
	}
}

func TestResolveCollectorFilesGlobStillWorks(t *testing.T) {
	dir := t.TempDir()
	writeCollectorFile(t, dir, "a.collector.yml", "a")
	writeCollectorFile(t, dir, "b.collector.yml", "b")
	writeCollectorFile(t, dir, "ignored.yml", "ignored")

	cfgPath := writeMinimalConfig(t, dir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [a, b]
collector_files:
  - "*.collector.yml"
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Collectors) != 2 {
		t.Fatalf("expected 2 collectors from glob, got %d", len(cfg.Collectors))
	}
}

func TestCollectorPathLoadsOnlyReferencedCollectors(t *testing.T) {
	dir := t.TempDir()
	collectorsDir := filepath.Join(dir, "collectors")
	if err := os.MkdirAll(collectorsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeCollectorFile(t, collectorsDir, "alpha.collector.yml", "alpha")
	writeCollectorFile(t, collectorsDir, "beta.collector.yml", "beta")
	writeCollectorFile(t, collectorsDir, "gamma.collector.yml", "gamma")
	writeCollectorFile(t, collectorsDir, "alpha_extra.collector.yml", "alpha_extra")
	// Unused broken YAML must never be opened/parsed when not referenced.
	if err := os.WriteFile(
		filepath.Join(collectorsDir, "unused_broken.collector.yml"),
		[]byte("collector_name: \"bad\\xescape\"\n"),
		0o644,
	); err != nil {
		t.Fatalf("write broken collector: %v", err)
	}

	cfgPath := writeMinimalConfig(t, dir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [alpha, beta, gamma]
collector_path: collectors
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Collectors) != 3 {
		t.Fatalf("expected only 3 referenced collectors, got %d (%+v)", len(cfg.Collectors), cfg.Collectors)
	}
	if len(cfg.Target.Collectors()) != 3 {
		t.Fatalf("expected target to reference 3 collectors, got %+v", cfg.Target.Collectors())
	}
}

func TestCollectorNameCandidatesFromFilename(t *testing.T) {
	cases := map[string][]string{
		"/etc/sql_exporter/collectors/pricing.collector.yml": {"pricing.collector", "pricing"},
		"pricing.yml": {"pricing"},
	}
	for path, want := range cases {
		got := collectorNameCandidatesFromFilename(path)
		if len(got) != len(want) {
			t.Fatalf("%s: got %v want %v", path, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: got %v want %v", path, got, want)
			}
		}
	}
}

func TestCollectorPathSupportsGlobCollectorRefs(t *testing.T) {
	dir := t.TempDir()
	collectorsDir := filepath.Join(dir, "collectors")
	if err := os.Mkdir(collectorsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeCollectorFile(t, collectorsDir, "db_a.yml", "db_a")
	writeCollectorFile(t, collectorsDir, "db_b.yml", "db_b")
	writeCollectorFile(t, collectorsDir, "other.yml", "other")

	cfgPath := writeMinimalConfig(t, dir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [db_*]
collector_path: collectors
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Collectors) != 2 {
		t.Fatalf("expected 2 db_* collectors, got %d (%+v)", len(cfg.Collectors), cfg.Collectors)
	}
}

func TestCollectorPathRecursiveByDefault(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "collectors", "db")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeCollectorFile(t, filepath.Join(dir, "collectors"), "top.yml", "top")
	writeCollectorFile(t, nested, "nested.yml", "nested")

	cfgPath := writeMinimalConfig(t, dir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [top, nested]
collector_path: collectors
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Collectors) != 2 {
		t.Fatalf("expected recursive load of 2 collectors, got %d", len(cfg.Collectors))
	}
}

func TestCollectorPathRecursiveDisabled(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "collectors", "db")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeCollectorFile(t, filepath.Join(dir, "collectors"), "top.yml", "top")
	writeCollectorFile(t, nested, "nested.yml", "nested")

	cfgPath := writeMinimalConfig(t, dir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [top]
collector_path: collectors
collector_path_recursive: false
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Collectors) != 1 || cfg.Collectors[0].Name != "top" {
		t.Fatalf("expected only top-level collector, got %+v", cfg.Collectors)
	}
}

func TestCollectorPathPatternRegex(t *testing.T) {
	dir := t.TempDir()
	collectorsDir := filepath.Join(dir, "collectors")
	if err := os.Mkdir(collectorsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeCollectorFile(t, collectorsDir, "keep.collector.yml", "keep")
	writeCollectorFile(t, collectorsDir, "skip.yml", "skip")

	cfgPath := writeMinimalConfig(t, dir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [keep]
collector_path: collectors
collector_path_pattern: '.*\.collector\.yml$'
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Collectors) != 1 || cfg.Collectors[0].Name != "keep" {
		t.Fatalf("expected only keep from regex filter, got %+v", cfg.Collectors)
	}
}

func TestCollectorFilesDirectoryAlsoRecursive(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "coll", "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeCollectorFile(t, nested, "from_files.yml", "from_files")

	cfgPath := writeMinimalConfig(t, dir, `
global:
  scrape_timeout: 10s
target:
  data_source_name: 'sqlserver://user:pass@localhost:1433'
  collectors: [from_files]
collector_files:
  - coll
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Collectors) != 1 || cfg.Collectors[0].Name != "from_files" {
		t.Fatalf("expected nested collector via collector_files dir, got %+v", cfg.Collectors)
	}
}
