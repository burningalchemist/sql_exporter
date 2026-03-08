package config

import (
	"fmt"
)

//
// Target
//

// TargetConfig defines a DSN and a set of collectors to be executed on it.
type TargetConfig struct {
	Name          string   `yaml:"name,omitempty" env:"NAME"`               // name of the target
	DSN           Secret   `yaml:"data_source_name" env:"DSN"`              // data source name to connect to
	CollectorRefs []string `yaml:"collectors" env:"COLLECTORS"`             // names of collectors to execute on the target
	EnablePing    *bool    `yaml:"enable_ping,omitempty" env:"ENABLE_PING"` // ping the target before executing the collectors

	collectors []*CollectorConfig // resolved collector references

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]any `yaml:",inline" json:"-"`
}

// Collectors returns the collectors referenced by the target, resolved.
func (t *TargetConfig) Collectors() []*CollectorConfig {
	return t.collectors
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for TargetConfig.
func (t *TargetConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type plain TargetConfig
	if err := unmarshal((*plain)(t)); err != nil {
		return err
	}

	// Check required fields
	if t.DSN == "" {
		return fmt.Errorf("missing data_source_name for target %+v", t)
	}
	if err := checkCollectorRefs(t.CollectorRefs, "target"); err != nil {
		return err
	}

	return checkOverflow(t.XXX, "target")
}
