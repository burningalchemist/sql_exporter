package sql_exporter

import (
	"fmt"
	"testing"

	"github.com/burningalchemist/sql_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.yaml.in/yaml/v3"
)

func TestNewQueryAutoMetricsDisabled(t *testing.T) {
	q, err := NewQuery("", &config.QueryConfig{Name: "q1", Query: "SELECT 1"}, nil, false)
	if err != nil {
		t.Fatalf("NewQuery: %v", err)
	}
	if q.durationDesc != nil || q.rowsDesc != nil {
		t.Fatalf("expected no auto-metric descs when disabled, got duration=%v rows=%v", q.durationDesc, q.rowsDesc)
	}
}

func TestNewQueryAutoMetricsEnabled(t *testing.T) {
	targetName, targetVal := "target", "db1"
	constLabels := []*dto.LabelPair{{Name: &targetName, Value: &targetVal}}

	q, err := NewQuery("", &config.QueryConfig{Name: "q1", Query: "SELECT 1"}, constLabels, true)
	if err != nil {
		t.Fatalf("NewQuery: %v", err)
	}
	if q.durationDesc == nil || q.rowsDesc == nil {
		t.Fatalf("expected auto-metric descs to be set when enabled")
	}
	if got := q.durationDesc.Name(); got != queryDurationName {
		t.Errorf("duration metric name = %q, want %q", got, queryDurationName)
	}
	if got := q.rowsDesc.Name(); got != queryRowsName {
		t.Errorf("rows metric name = %q, want %q", got, queryRowsName)
	}

	gotLabels := q.durationDesc.ConstLabels()
	if len(gotLabels) != 2 {
		t.Fatalf("expected 2 const labels (target, query), got %d", len(gotLabels))
	}
	labels := make(map[string]string, len(gotLabels))
	for _, lp := range gotLabels {
		labels[lp.GetName()] = lp.GetValue()
	}
	if labels[queryLabelName] != "q1" {
		t.Errorf("query label = %q, want q1", labels[queryLabelName])
	}
	if labels["target"] != "db1" {
		t.Errorf("target label = %q, want db1", labels["target"])
	}
}

func TestNewQueryAutoMetricsEnabledNoConstLabels(t *testing.T) {
	q, err := NewQuery("", &config.QueryConfig{Name: "singleton", Query: "SELECT 1"}, nil, true)
	if err != nil {
		t.Fatalf("NewQuery: %v", err)
	}
	gotLabels := q.durationDesc.ConstLabels()
	if len(gotLabels) != 1 {
		t.Fatalf("expected just the query label, got %d labels", len(gotLabels))
	}
	if gotLabels[0].GetName() != queryLabelName || gotLabels[0].GetValue() != "singleton" {
		t.Errorf("expected query=singleton, got %s=%s", gotLabels[0].GetName(), gotLabels[0].GetValue())
	}
}

// Query names are local to a collector. The automatic query metrics need the collector label so two collectors can use
// the same query name without producing duplicate series.
func TestQueryMetricsDistinguishCollectors(t *testing.T) {
	metrics := make([]*dto.Metric, 0, 2)
	for _, collectorName := range []string{"a", "b"} {
		var cc config.CollectorConfig
		if err := yaml.Unmarshal([]byte(fmt.Sprintf(`
collector_name: %s
metrics:
- {metric_name: %s_metric, type: gauge, help: h, values: [v], query_ref: q}
queries:
- {query_name: q, query: SELECT 1 AS v}
`, collectorName, collectorName)), &cc); err != nil {
			t.Fatalf("unmarshal collector %s: %v", collectorName, err)
		}
		c, err := NewCollector("", &cc, nil, true)
		if err != nil {
			t.Fatalf("NewCollector(%s): %v", collectorName, err)
		}
		metric := &dto.Metric{}
		if err := NewMetric(c.(*collector).queries[0].durationDesc, 0).Write(metric); err != nil {
			t.Fatalf("write metric for collector %s: %v", collectorName, err)
		}
		metrics = append(metrics, metric)
	}

	name, help := queryDurationName, queryDurationHelp
	metricType := dto.MetricType_GAUGE
	gatherer := prometheus.GathererFunc(func() ([]*dto.MetricFamily, error) {
		return []*dto.MetricFamily{{Name: &name, Help: &help, Type: &metricType, Metric: metrics}}, nil
	})
	if _, err := (prometheus.Gatherers{gatherer}).Gather(); err != nil {
		t.Fatalf("Gather: %v", err)
	}
}
