package config

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricConfig defines a Prometheus metric, the SQL query to populate it and the mapping of columns to metric
// keys/values.
type MetricConfig struct {
	Name         string            `yaml:"metric_name"`             // the Prometheus metric name
	TypeString   string            `yaml:"type"`                    // the Prometheus metric type
	Help         string            `yaml:"help"`                    // the Prometheus metric help text
	KeyLabels    []string          `yaml:"key_labels,omitempty"`    // expose these columns as labels from SQL
	StaticLabels map[string]string `yaml:"static_labels,omitempty"` // fixed key/value pairs as static labels
	ValueLabel   string            `yaml:"value_label,omitempty"`   // with multiple value columns, map their names under this label
	Values       []string          `yaml:"values"`                  // expose each of these columns as a value, keyed by column name
	QueryLiteral string            `yaml:"query,omitempty"`         // a literal query
	QueryRef     string            `yaml:"query_ref,omitempty"`     // references a query in the query map

	NoPreparedStatement bool     `yaml:"no_prepared_statement,omitempty"` // do not prepare statement
	StaticValue         *float64 `yaml:"static_value,omitempty"`
	TimestampValue      string   `yaml:"timestamp_value,omitempty"` // optional column name containing a valid timestamp value

	valueType prometheus.ValueType // TypeString converted to prometheus.ValueType
	query     *QueryConfig         // QueryConfig resolved from QueryRef or generated from Query

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]any `yaml:",inline" json:"-"`
}

// ValueType returns the metric type, converted to a prometheus.ValueType.
func (m *MetricConfig) ValueType() prometheus.ValueType {
	return m.valueType
}

// Query returns the query defined (as a literal) or referenced by the metric.
func (m *MetricConfig) Query() *QueryConfig {
	return m.query
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for MetricConfig.
func (m *MetricConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type plain MetricConfig
	if err := unmarshal((*plain)(m)); err != nil {
		return err
	}

	if err := m.validateRequiredFields(); err != nil {
		return err
	}
	if err := m.setValueType(); err != nil {
		return err
	}
	if err := m.validateKeyLabels(); err != nil {
		return err
	}
	if err := m.validateValues(); err != nil {
		return err
	}

	return checkOverflow(m.XXX, "metric")
}

// Check required fields
func (m *MetricConfig) validateRequiredFields() error {
	if m.Name == "" {
		return fmt.Errorf("missing name for metric %+v", m)
	}
	if m.TypeString == "" {
		return fmt.Errorf("missing type for metric %q", m.Name)
	}
	if m.Help == "" {
		return fmt.Errorf("missing help for metric %q", m.Name)
	}
	if (m.QueryLiteral == "") == (m.QueryRef == "") {
		return fmt.Errorf("exactly one of query and query_ref must be specified for metric %q", m.Name)
	}

	return nil
}

// Set the metric type
func (m *MetricConfig) setValueType() error {
	switch strings.ToLower(m.TypeString) {
	case "counter":
		m.valueType = prometheus.CounterValue
	case "gauge":
		m.valueType = prometheus.GaugeValue
	default:
		return fmt.Errorf("unsupported metric type: %s", m.TypeString)
	}

	return nil
}

// Check for duplicate key labels
func (m *MetricConfig) validateKeyLabels() error {
	for i, li := range m.KeyLabels {
		if err := checkLabel(li, "metric", m.Name); err != nil {
			return err
		}
		for _, lj := range m.KeyLabels[i+1:] {
			if li == lj {
				return fmt.Errorf("duplicate key label %q for metric %q", li, m.Name)
			}
		}
		if m.ValueLabel == li {
			return fmt.Errorf("duplicate label %q (defined in both key_labels and value_label) for metric %q", li, m.Name)
		}
	}

	return nil
}

// Check for duplicate values
func (m *MetricConfig) validateValues() error {
	if len(m.Values) == 0 && m.StaticValue == nil {
		return fmt.Errorf("no values defined for metric %q", m.Name)
	}

	if len(m.Values) > 0 && m.StaticValue != nil {
		return fmt.Errorf("metric %q cannot have both static_value and values defined", m.Name)
	}

	if len(m.Values) > 1 {
		// Multiple value columns but no value label to identify them
		if m.ValueLabel == "" {
			return fmt.Errorf("value_label must be defined for metric with multiple values %q", m.Name)
		}
		if err := checkLabel(m.ValueLabel, "value_label for metric", m.Name); err != nil {
			return err
		}
	}

	return nil
}
