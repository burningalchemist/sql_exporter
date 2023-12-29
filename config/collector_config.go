package config

import (
	"fmt"

	"github.com/prometheus/common/model"
)

//
// Collectors
//

// CollectorConfig defines a set of metrics and how they are collected.
type CollectorConfig struct {
	Name        string          `yaml:"collector_name"`         // name of this collector
	MinInterval model.Duration  `yaml:"min_interval,omitempty"` // minimum interval between query executions
	Metrics     []*MetricConfig `yaml:"metrics"`                // metrics/queries defined by this collector
	Queries     []*QueryConfig  `yaml:"queries,omitempty"`      // named queries defined by this collector

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]any `yaml:",inline" json:"-"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for CollectorConfig.
func (c *CollectorConfig) UnmarshalYAML(unmarshal func(any) error) error {
	// Default to undefined (a negative value) so it can be overridden by the global default when not explicitly set.
	c.MinInterval = -1

	type plain CollectorConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	if len(c.Metrics) == 0 {
		return fmt.Errorf("no metrics defined for collector %q", c.Name)
	}

	// Set metric.query for all metrics: resolve query references (if any) and generate QueryConfigs for literal queries.
	queries := make(map[string]*QueryConfig, len(c.Queries))
	for _, query := range c.Queries {
		queries[query.Name] = query
	}
	for _, metric := range c.Metrics {
		if metric.QueryRef != "" {
			query, found := queries[metric.QueryRef]
			if !found {
				return fmt.Errorf("unresolved query_ref %q in metric %q of collector %q", metric.QueryRef, metric.Name, c.Name)
			}
			metric.query = query
			query.metrics = append(query.metrics, metric)
		} else {
			// For literal queries generate a QueryConfig with a name based off collector and metric name.
			metric.query = &QueryConfig{
				Name:                metric.Name,
				Query:               metric.QueryLiteral,
				NoPreparedStatement: metric.NoPreparedStatement,
			}
		}
	}

	return checkOverflow(c.XXX, "collector")
}
