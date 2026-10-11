// Package config provides the configuration structures and functions for sql_exporter.
package config

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sethvargo/go-envconfig"
	"go.yaml.in/yaml/v3"
)

// MaxInt32 defines the maximum value of allowed integers and serves to help us avoid overflow/wraparound issues.
const MaxInt32 int = 1<<31 - 1

// EnvPrefix is the prefix for environment variables.
const (
	EnvPrefix string = "SQLEXPORTER_"

	EnvConfigFile string = EnvPrefix + "CONFIG"
	EnvDebug      string = EnvPrefix + "DEBUG"
)

// secretResolutionTimeout is the maximum time allowed for resolving secrets from secret providers to prevent hanging
// indefinitely if a secret provider is unresponsive.
const secretResolutionTimeout = 30 * time.Second

var (
	EnablePing        bool
	IgnoreMissingVals bool
	DsnOverride       string
	TargetLabel       string
)

// Load attempts to parse the given config file and return a Config object.
func Load(configFile string) (*Config, error) {
	slog.Debug("Loading configuration", "file", configFile)
	buf, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	c := Config{configFile: configFile}
	err = yaml.Unmarshal(buf, &c)
	if err != nil {
		return nil, err
	}

	if c.Globals == nil {
		return nil, fmt.Errorf("empty or no configuration provided")
	}

	if err := c.resolveSecrets(); err != nil {
		return nil, err
	}

	return &c, nil
}

//
// Top-level config
//

// Config is a collection of jobs and collectors.
type Config struct {
	Globals        *GlobalConfig      `yaml:"global,omitempty" env:", prefix=GLOBAL_"`
	CollectorFiles []string           `yaml:"collector_files,omitempty" env:"COLLECTOR_FILES"`
	Target         *TargetConfig      `yaml:"target,omitempty" env:", prefix=TARGET_"`
	Jobs           []*JobConfig       `yaml:"jobs,omitempty"`
	Collectors     []*CollectorConfig `yaml:"collectors,omitempty"`

	configFile string

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]any `yaml:",inline" json:"-"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for Config.
func (c *Config) UnmarshalYAML(unmarshal func(any) error) error {
	// unmarshalConfig does the actual unmarshalling
	if err := c.unmarshalConfig(unmarshal); err != nil {
		return err
	}
	// Populate global defaults.
	if err := c.populateGlobalDefaults(); err != nil {
		return err
	}

	// Process environment variables.
	if err := c.processEnvConfig(); err != nil {
		return err
	}

	// Load any externally defined collectors.
	if err := c.loadCollectorFiles(); err != nil {
		return err
	}

	// Check required fields
	if err := c.checkRequiredFields(); err != nil {
		return err
	}

	// Populate collector references for the target/jobs.
	if err := c.populateCollectorReferences(); err != nil {
		return err
	}

	return checkOverflow(c.XXX, "config")
}

// unmarshalConfig unmarshals the config, but does not populate global defaults, process environment variables, or
// check required fields.
func (c *Config) unmarshalConfig(unmarshal func(any) error) error {
	type plain Config
	return unmarshal((*plain)(c))
}

// populateGlobalDefaults populates any unset global defaults.
func (c *Config) populateGlobalDefaults() error {
	if c.Globals == nil {
		c.Globals = &GlobalConfig{}
		// Force a dummy unmarshall to populate global defaults
		return c.Globals.UnmarshalYAML(func(any) error { return nil })
	}
	return nil
}

// processEnvConfig processes environment variables.
func (c *Config) processEnvConfig() error {
	return envconfig.ProcessWith(context.Background(), &envconfig.Config{
		Target:           c,
		Lookuper:         envconfig.PrefixLookuper(EnvPrefix, envconfig.OsLookuper()),
		DefaultNoInit:    true,
		DefaultOverwrite: true,
		DefaultDelimiter: ";",
	})
}

// checkRequiredFields checks that all required fields are present.
func (c *Config) checkRequiredFields() error {
	if (len(c.Jobs) == 0) == (c.Target == nil) {
		return fmt.Errorf("exactly one of `jobs` and `target` must be defined")
	}

	// Check target configuration
	if c.Target != nil {
		if c.Target.DSN == "" {
			return fmt.Errorf("target.data_source_name is required")
		}
		if len(c.Target.CollectorRefs) == 0 {
			return fmt.Errorf("target.collectors is required")
		}
	}

	// Check jobs configuration
	for i, job := range c.Jobs {
		if job.Name == "" {
			return fmt.Errorf("job[%d].job_name is required", i)
		}
		if len(job.CollectorRefs) == 0 {
			return fmt.Errorf("job[%d].collectors is required", i)
		}
		if len(job.StaticConfigs) == 0 {
			return fmt.Errorf("job[%d].static_configs is required", i)
		}
		for j, staticConfig := range job.StaticConfigs {
			if len(staticConfig.Targets) == 0 {
				return fmt.Errorf("job[%d].static_configs[%d].targets is required", i, j)
			}
		}
	}

	return nil
}

// populateCollectorReferences populates collector references for the target/jobs.
func (c *Config) populateCollectorReferences() error {
	colls := make(map[string]*CollectorConfig)
	for _, coll := range c.Collectors {
		if coll.MinInterval < 0 {
			coll.MinInterval = c.Globals.MinInterval
		}
		if _, found := colls[coll.Name]; found {
			return fmt.Errorf("duplicate collector name: %s", coll.Name)
		}
		colls[coll.Name] = coll
	}

	if c.Target != nil {
		cs, err := resolveCollectorRefs(c.Target.CollectorRefs, colls, "target")
		if err != nil {
			return err
		}
		c.Target.collectors = cs
	}

	for _, j := range c.Jobs {
		cs, err := resolveCollectorRefs(j.CollectorRefs, colls,
			fmt.Sprintf("job %q", j.Name))
		if err != nil {
			return err
		}
		j.collectors = cs
	}
	return nil
}

// YAML marshals the config into YAML format.
func (c *Config) YAML() ([]byte, error) {
	return yaml.Marshal(c)
}

// loadCollectorFiles resolves all collector file globs to files and loads the collectors they define.
func (c *Config) loadCollectorFiles() error {
	baseDir := filepath.Dir(c.configFile)
	refs := c.referencedCollectorPatterns()

	for _, cfglob := range c.CollectorFiles {
		// Resolve relative paths by joining them to the configuration file's directory.
		if len(cfglob) > 0 && !filepath.IsAbs(cfglob) {
			cfglob = filepath.Join(baseDir, cfglob)
		}

		// Resolve the glob to actual filenames.
		cfs, err := filepath.Glob(cfglob)
		slog.Debug("External collector files found", "count", len(cfs), "glob", cfglob)
		if err != nil {
			// The only error can be a bad pattern.
			return fmt.Errorf("error resolving collector files for %s: %w", cfglob, err)
		}

		// And load the CollectorConfig defined in each file.
		for _, cf := range cfs {
			buf, err := os.ReadFile(cf)
			if err != nil {
				return err
			}

			cc, err := parseCollectorFile(cf, buf)
			if err != nil {
				// Broken files that are not referenced by target/jobs should not fail startup
				// when a greedy glob matches them. Referenced collectors still fail hard.
				if skip, skipErr := shouldSkipBrokenCollectorFile(cf, buf, refs); skipErr != nil {
					return skipErr
				} else if skip {
					slog.Warn("Skipping broken collector file not referenced by target/jobs",
						"file", cf, "error", err)
					continue
				}
				return err
			}

			c.Collectors = append(c.Collectors, cc)
			slog.Debug("Loaded collector", "name", cc.Name, "file", cf)
		}
	}

	return nil
}

// parseCollectorFile parses a single external collector definition file.
func parseCollectorFile(path string, buf []byte) (*CollectorConfig, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(buf, &node); err != nil {
		return nil, fmt.Errorf("error parsing collector file %s: %w", path, err)
	}
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		return nil, fmt.Errorf("collector file %s is not a valid YAML document", path)
	}

	top := node.Content[0]
	if top.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("collector file %s must define a single YAML map/object at the top level", path)
	}

	// Each external file must define one collector object, not a collectors list.
	for i := 0; i < len(top.Content); i += 2 {
		keyNode := top.Content[i]
		valNode := top.Content[i+1]
		if keyNode.Value == "collectors" && valNode.Kind == yaml.SequenceNode {
			return nil, fmt.Errorf(
				"collector file %s contains a 'collectors' list. Each file must define a single collector object",
				path,
			)
		}
	}

	cc := CollectorConfig{}
	if err := node.Decode(&cc); err != nil {
		return nil, fmt.Errorf("error parsing collector file %s: %w", path, err)
	}
	if cc.Name == "" {
		return nil, fmt.Errorf("collector file %s must define a collector with a name", path)
	}
	return &cc, nil
}

// referencedCollectorPatterns returns collector name patterns from target and jobs.
func (c *Config) referencedCollectorPatterns() []string {
	refs := make([]string, 0)
	if c.Target != nil {
		refs = append(refs, c.Target.CollectorRefs...)
	}
	for _, job := range c.Jobs {
		refs = append(refs, job.CollectorRefs...)
	}
	return refs
}

// shouldSkipBrokenCollectorFile reports whether a failed collector file can be skipped.
// It is skipped only when none of its known identities match target/jobs collector refs.
func shouldSkipBrokenCollectorFile(path string, buf []byte, refs []string) (bool, error) {
	if len(refs) == 0 {
		return false, nil
	}

	for _, name := range collectorIdentities(path, buf) {
		matched, err := collectorNameMatchesRefs(name, refs)
		if err != nil {
			return false, err
		}
		if matched {
			return false, nil
		}
	}
	return true, nil
}

// collectorIdentities returns names used to decide whether a broken file is referenced.
// Prefer collector_name from the file when readable; also include the filename stem so
// completely invalid YAML can still be skipped for unreferenced files under a greedy glob.
func collectorIdentities(path string, buf []byte) []string {
	names := make([]string, 0, 2)

	var peek struct {
		Name string `yaml:"collector_name"`
	}
	if err := yaml.Unmarshal(buf, &peek); err == nil && peek.Name != "" {
		names = append(names, peek.Name)
	}

	base := filepath.Base(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	names = append(names, stem)
	if strings.EqualFold(filepath.Ext(stem), ".collector") {
		names = append(names, strings.TrimSuffix(stem, filepath.Ext(stem)))
	}
	return names
}

// collectorNameMatchesRefs reports whether name matches any collector reference pattern.
func collectorNameMatchesRefs(name string, refs []string) (bool, error) {
	for _, ref := range refs {
		matched, err := filepath.Match(ref, name)
		if err != nil {
			return false, fmt.Errorf("bad collector reference %q: %w", ref, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (c *Config) resolveSecrets() error {
	// Create a context with timeout for secret resolution to avoid hanging indefinitely if a secret provider is
	// unresponsive.
	ctx, cancel := context.WithTimeout(context.Background(), secretResolutionTimeout)
	defer cancel()
	resolver := &secretResolver{} // scoped here, GC'd when resolveSecrets returns

	if c.Target != nil {
		if isSecretRef(string(c.Target.DSN)) {
			dsn, err := resolver.resolve(ctx, string(c.Target.DSN))
			if err != nil {
				return fmt.Errorf("error resolving target DSN: %w", err)
			}
			c.Target.DSN = Secret(dsn)
		}
	}
	// Maps are reference types, so this will update the DSNs in place for all targets defined in jobs.
	for _, job := range c.Jobs {
		for _, staticConfig := range job.StaticConfigs {
			for targetName, dsn := range staticConfig.Targets {
				if !isSecretRef(string(dsn)) {
					continue
				}
				resolved, err := resolver.resolve(ctx, string(dsn))
				if err != nil {
					return fmt.Errorf("error resolving DSN for target %q in job %q: %w", targetName,
						job.Name, err)
				}
				staticConfig.Targets[targetName] = Secret(resolved)
			}
		}
	}

	return nil
}

// isSecretRef checks if the given value is a secret reference by parsing it as a URL and checking if the scheme matches
// any registered secret provider.
func isSecretRef(value string) bool {
	u, err := url.Parse(value)
	if err != nil {
		return false
	}

	_, ok := secretProviders[u.Scheme]
	return ok
}
