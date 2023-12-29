package config

import "fmt"

//
// Jobs
//

// JobConfig defines a set of collectors to be executed on a set of targets.
type JobConfig struct {
	Name          string          `yaml:"job_name"`       // name of this job
	CollectorRefs []string        `yaml:"collectors"`     // names of collectors to apply to all targets in this job
	StaticConfigs []*StaticConfig `yaml:"static_configs"` // collections of statically defined targets

	collectors []*CollectorConfig // resolved collector references

	EnablePing *bool `yaml:"enable_ping,omitempty"` // ping the target before executing the collectors

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]any `yaml:",inline" json:"-"`
}

// Collectors returns the collectors referenced by the job, resolved.
func (j *JobConfig) Collectors() []*CollectorConfig {
	return j.collectors
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for JobConfig.
func (j *JobConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type plain JobConfig
	if err := unmarshal((*plain)(j)); err != nil {
		return err
	}

	// Check required fields
	if j.Name == "" {
		return fmt.Errorf("missing name for job %+v", j)
	}
	if err := checkCollectorRefs(j.CollectorRefs, fmt.Sprintf("job %q", j.Name)); err != nil {
		return err
	}

	if len(j.StaticConfigs) == 0 {
		return fmt.Errorf("no targets defined for job %q", j.Name)
	}

	return checkOverflow(j.XXX, "job")
}

// checkLabelCollisions checks for label collisions between StaticConfig labels and Metric labels.
//
//lint:ignore U1000 - it's unused so far
func (j *JobConfig) checkLabelCollisions() error {
	sclabels := make(map[string]any)
	for _, s := range j.StaticConfigs {
		for _, l := range s.Labels {
			sclabels[l] = nil
		}
	}

	for _, c := range j.collectors {
		for _, m := range c.Metrics {
			for _, l := range m.KeyLabels {
				if _, ok := sclabels[l]; ok {
					return fmt.Errorf(
						"label collision in job %q: label %q is defined both by a static_config and by metric %q of collector %q",
						j.Name, l, m.Name, c.Name)
				}
			}
		}
	}
	return nil
}

// StaticConfig defines a set of targets and optional labels to apply to the metrics collected from them.
type StaticConfig struct {
	Targets map[string]Secret `yaml:"targets"`          // map of target names to data source names
	Labels  map[string]string `yaml:"labels,omitempty"` // labels to apply to all metrics collected from the targets

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]any `yaml:",inline" json:"-"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for StaticConfig.
func (s *StaticConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type plain StaticConfig
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}

	// Check for empty/duplicate target names/data source names
	tnames := make(map[string]any)
	dsns := make(map[string]any)
	for tname, dsn := range s.Targets {
		if tname == "" {
			return fmt.Errorf("empty target name in static config %+v", s)
		}
		if _, ok := tnames[tname]; ok {
			return fmt.Errorf("duplicate target name %q in static_config %+v", tname, s)
		}
		tnames[tname] = nil
		if dsn == "" {
			return fmt.Errorf("empty data source name in static config %+v", s)
		}
		if _, ok := dsns[string(dsn)]; ok {
			return fmt.Errorf("duplicate data source name %q in static_config %+v", tname, s)
		}
		dsns[string(dsn)] = nil
	}

	return checkOverflow(s.XXX, "static_config")
}
