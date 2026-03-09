// Package config provides the configuration structures and functions for sql_exporter.
package config

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v3"
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

			// Inspect yaml to ensure strict parsing and expect an object, not a list.
			var node yaml.Node
			if err := yaml.Unmarshal(buf, &node); err != nil {
				return fmt.Errorf("error parsing collector file %s: %w", cf, err)
			}
			if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
				return fmt.Errorf("collector file %s is not a valid YAML document", cf)
			}

			top := node.Content[0]
			if top.Kind != yaml.MappingNode {
				return fmt.Errorf("collector file %s must define a single YAML map/object at the top level",
					cf)
			}

			// Check for 'collectors' key with a sequence value
			for i := 0; i < len(top.Content); i += 2 {
				keyNode := top.Content[i]
				valNode := top.Content[i+1]
				if keyNode.Value == "collectors" && valNode.Kind == yaml.SequenceNode {
					return fmt.Errorf(
						"collector file %s contains a 'collectors' list. Each file must define a single collector object",
						cf,
					)
				}
			}

			// Now unmarshal into a CollectorConfig.
			cc := CollectorConfig{}
			if err := node.Decode(&cc); err != nil {
				return fmt.Errorf("error parsing collector file %s: %w", cf, err)
			}
			if cc.Name == "" {
				return fmt.Errorf("collector file %s must define a collector with a name", cf)
			}

			// Append to the config's collectors.
			c.Collectors = append(c.Collectors, &cc)
			slog.Debug("Loaded collector", "name", cc.Name, "file", cf)
		}
	}

	return nil
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
