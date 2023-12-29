package config

import "fmt"

// QueryConfig defines a named query, to be referenced by one or multiple metrics.
type QueryConfig struct {
	Name  string `yaml:"query_name"` // the query name, to be referenced via `query_ref`
	Query string `yaml:"query"`      // the named query

	NoPreparedStatement bool `yaml:"no_prepared_statement,omitempty"` // do not prepare statement

	metrics []*MetricConfig // metrics referencing this query

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]any `yaml:",inline" json:"-"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for QueryConfig.
func (q *QueryConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type plain QueryConfig
	if err := unmarshal((*plain)(q)); err != nil {
		return err
	}

	// Check required fields
	if q.Name == "" {
		return fmt.Errorf("missing name for query %+v", *q)
	}
	if q.Query == "" {
		return fmt.Errorf("missing query literal for query %q", q.Name)
	}

	q.metrics = make([]*MetricConfig, 0, 2)

	return checkOverflow(q.XXX, "metric")
}
