package config

import (
	"fmt"
	"time"

	"github.com/prometheus/common/model"
)

// GlobalConfig contains globally applicable defaults.
type GlobalConfig struct {
	MinInterval             model.Duration `yaml:"min_interval" env:"MIN_INTERVAL"`                             // minimum interval between query executions, default is 0
	ScrapeTimeout           model.Duration `yaml:"scrape_timeout" env:"SCRAPE_TIMEOUT"`                         // per-scrape timeout, global
	TimeoutOffset           model.Duration `yaml:"scrape_timeout_offset" env:"SCRAPE_TIMEOUT_OFFSET"`           // offset to subtract from timeout in seconds
	ScrapeErrorDropInterval model.Duration `yaml:"scrape_error_drop_interval" env:"SCRAPE_ERROR_DROP_INTERVAL"` // interval to drop scrape errors from the error counter, default is 0
	MaxConnLifetime         time.Duration  `yaml:"max_connection_lifetime" env:"MAX_CONNECTION_LIFETIME"`       // maximum amount of time a connection may be reused to any one target

	MaxConns     int `yaml:"max_connections" env:"MAX_CONNECTIONS"`           // maximum number of open connections to any one target
	MaxIdleConns int `yaml:"max_idle_connections" env:"MAX_IDLE_CONNECTIONS"` // maximum number of idle connections to any one target

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]any `yaml:",inline" json:"-"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for GlobalConfig.
func (g *GlobalConfig) UnmarshalYAML(unmarshal func(any) error) error {
	// Default to running the queries on every scrape.
	g.MinInterval = model.Duration(0)
	// Default to 10 seconds, since Prometheus has a 10 second scrape timeout default.
	g.ScrapeTimeout = model.Duration(10 * time.Second)
	// Default to 0 for scrape error drop interval.
	g.ScrapeErrorDropInterval = model.Duration(0)
	// Default to .5 seconds.
	g.TimeoutOffset = model.Duration(500 * time.Millisecond)
	g.MaxConns = 3
	g.MaxIdleConns = 3
	g.MaxConnLifetime = time.Duration(0)

	type plain GlobalConfig
	if err := unmarshal((*plain)(g)); err != nil {
		return err
	}

	if g.TimeoutOffset <= 0 {
		return fmt.Errorf("global.scrape_timeout_offset must be strictly positive, have %s", g.TimeoutOffset)
	}

	return checkOverflow(g.XXX, "global")
}
