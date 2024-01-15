package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

// MaxInt32 defines the maximum value of allowed integers
// and serves to help us avoid overflow/wraparound issues.
const MaxInt32 int = 1<<31 - 1

// EnvPrefix is the prefix for environment variables.
const (
	EnvPrefix string = "SQLEXPORTER_"

	EnvConfigFile string = EnvPrefix + "CONFIG"
	EnvDebug      string = EnvPrefix + "DEBUG"
)

var (
	EnablePing        bool
	IgnoreMissingVals bool
	DsnOverride       string
	TargetLabel       string
)

// Load attempts to parse the given config file and return a Config object.
func Load(configFile string) (*Config, error) {
	klog.Infof("Loading configuration from %s", configFile)
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

	// Load any externally defined collectors.
	if err := c.loadCollectorFiles(); err != nil {
		return err
	}

	// Process environment variables.
	if err := c.processEnvConfig(); err != nil {
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

// unmarshalConfig unmarshals the config, but does not populate global defaults, process environment variables, or check required fields.
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
		cs, err := resolveCollectorRefs(j.CollectorRefs, colls, fmt.Sprintf("job %q", j.Name))
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
		klog.Infof("External collector files found: %v", len(cfs))
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

			cc := CollectorConfig{}
			err = yaml.Unmarshal(buf, &cc)
			if err != nil {
				return err
			}

			c.Collectors = append(c.Collectors, &cc)
			klog.Infof("Loaded collector '%s' from %s", cc.Name, cf)
		}
	}

	return nil
}
