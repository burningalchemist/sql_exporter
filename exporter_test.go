package sql_exporter

import (
	"bytes"
	"log/slog"
	"maps"
	"testing"

	"github.com/burningalchemist/sql_exporter/config"
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
