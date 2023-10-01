package sql_exporter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/burningalchemist/sql_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/proto"
)

var (
	SvcRegistry     = prometheus.NewRegistry()
	svcMetricLabels = []string{"job", "target", "collector", "query"}
)

// Exporter is a prometheus.Gatherer that gathers SQL metrics from targets and merges them with the default registry.
type Exporter interface {
	prometheus.Gatherer

	// WithContext returns a (single use) copy of the Exporter, which will use the provided context for Gather() calls.
	WithContext(context.Context) Exporter
	// Config returns the Exporter's underlying Config object.
	Config() *config.Config
	UpdateTarget([]Target)
}

type exporter struct {
	config  *config.Config
	targets []Target

	ctx          context.Context
	scrapeErrors *prometheus.CounterVec
}

// NewExporter returns a new Exporter with the provided config.
func NewExporter(configFile string) (Exporter, error) {
	c, err := config.Load(configFile)
	if err != nil {
		return nil, err
	}

	if val, ok := os.LookupEnv(config.EnvDsnOverride); ok {
		config.DsnOverride = val
	}
	// Override the DSN if requested (and in single target mode).
	if config.DsnOverride != "" {
		if len(c.Jobs) > 0 {
			return nil, fmt.Errorf("the config.data-source-name flag (value %q) only applies in single target mode", config.DsnOverride)
		}
		c.Target.DSN = config.Secret(config.DsnOverride)
	}

	var targets []Target
	if c.Target != nil {
		if c.Target.EnablePing == nil {
			c.Target.EnablePing = &config.EnablePing
		}
		target, err := NewTarget("", c.Target.Name, string(c.Target.DSN), c.Target.Collectors(), nil, c.Globals, c.Target.EnablePing)
		if err != nil {
			return nil, err
		}
		targets = []Target{target}
	} else {
		if len(c.Jobs) > (config.MaxInt32 / 3) {
			return nil, errors.New("'jobs' list is too large")
		}
		targets = make([]Target, 0, len(c.Jobs)*3)
		for _, jc := range c.Jobs {
			job, err := NewJob(jc, c.Globals)
			if err != nil {
				return nil, err
			}
			targets = append(targets, job.Targets()...)
		}
	}

	scrapeErrors := registerSvcMetrics()

	return &exporter{
		config:       c,
		targets:      targets,
		ctx:          context.Background(),
		scrapeErrors: scrapeErrors,
	}, nil
}

func (e *exporter) WithContext(ctx context.Context) Exporter {
	return &exporter{
		config:       e.config,
		targets:      e.targets,
		ctx:          ctx,
		scrapeErrors: e.scrapeErrors,
	}
}

// Gather implements prometheus.Gatherer.
func (e *exporter) Gather() ([]*dto.MetricFamily, error) {
	var (
		metricChan = make(chan Metric, capMetricChan)
		errs       prometheus.MultiError
	)

	var wg sync.WaitGroup
	wg.Add(len(e.targets))
	for _, t := range e.targets {
		go func(target Target) {
			defer wg.Done()
			target.Collect(e.ctx, metricChan)
		}(t)
	}

	// Wait for all collectors to complete, then close the channel.
	go func() {
		wg.Wait()
		close(metricChan)
	}()

	// Drain metricChan in case of premature return.
	defer func() {
		for range metricChan {
		}
	}()

	// Gather.
	dtoMetricFamilies := make(map[string]*dto.MetricFamily, 10)
	for metric := range metricChan {
		dtoMetric := &dto.Metric{}
		if err := metric.Write(dtoMetric); err != nil {
			errs = append(errs, err)
			if err.Context() != "" {
				ctxLabels := parseContextLog(err.Context())
				values := make([]string, len(svcMetricLabels))
				for i, label := range svcMetricLabels {
					values[i] = ctxLabels[label]
				}
				e.scrapeErrors.WithLabelValues(values...).Inc()
			}
			continue
		}
		metricDesc := metric.Desc()
		dtoMetricFamily, ok := dtoMetricFamilies[metricDesc.Name()]
		if !ok {
			dtoMetricFamily = &dto.MetricFamily{}
			dtoMetricFamily.Name = proto.String(metricDesc.Name())
			dtoMetricFamily.Help = proto.String(metricDesc.Help())
			switch {
			case dtoMetric.Gauge != nil:
				dtoMetricFamily.Type = dto.MetricType_GAUGE.Enum()
			case dtoMetric.Counter != nil:
				dtoMetricFamily.Type = dto.MetricType_COUNTER.Enum()
			default:
				errs = append(errs, fmt.Errorf("don't know how to handle metric %v", dtoMetric))
				continue
			}
			dtoMetricFamilies[metricDesc.Name()] = dtoMetricFamily
		}
		dtoMetricFamily.Metric = append(dtoMetricFamily.Metric, dtoMetric)
	}

	// No need to sort metric families, prometheus.Gatherers will do that for us when merging.
	result := make([]*dto.MetricFamily, 0, len(dtoMetricFamilies))
	for _, mf := range dtoMetricFamilies {
		result = append(result, mf)
	}
	return result, errs
}

// Config implements Exporter.
func (e *exporter) Config() *config.Config {
	return e.config
}

func (e *exporter) UpdateTarget(target []Target) {
	e.targets = target
}

// registerSvcMetrics registers the metrics for the exporter itself.
func registerSvcMetrics() *prometheus.CounterVec {
	scrapeErrors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "scrape_errors",
		Help: "Total number of scrape errors per job, target, collector and query.",
	}, svcMetricLabels)
	SvcRegistry.MustRegister(scrapeErrors)
	return scrapeErrors
}

// split comma separated list of key=value pairs and return a map of key value pairs
func parseContextLog(list string) map[string]string {
	m := make(map[string]string)
	for _, item := range strings.Split(list, ",") {
		parts := strings.SplitN(item, "=", 2)
		m[parts[0]] = parts[1]
	}
	return m
}
