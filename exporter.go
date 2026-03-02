// Package sql_exporter provides a Prometheus exporter for SQL metrics. It
// gathers metrics from SQL databases and exposes them in a format suitable for
// Prometheus scraping. The package supports multiple targets, collectors, and
// queries, and allows for flexible configuration through YAML files.
package sql_exporter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/burningalchemist/sql_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"google.golang.org/protobuf/proto"
)

var (
	SvcRegistry        = prometheus.NewRegistry()
	svcMetricLabels    = []string{"job", "target", "collector", "query"}
	scrapeErrorsMetric *prometheus.CounterVec
)

// Exporter is a prometheus.Gatherer that gathers SQL metrics from targets and merges them with the custom registry.
type Exporter interface {
	prometheus.Gatherer

	// WithContext returns a (single use) copy of the Exporter, which will use the provided context for Gather() calls.
	WithContext(context.Context) Exporter
	// Config returns the Exporter's underlying Config object.
	Config() *config.Config
	// UpdateTarget updates the targets field
	UpdateTarget([]Target)
	// SetJobFilters sets the jobFilters field
	SetJobFilters([]string) error
	// DropErrorMetrics resets the scrape_errors_total metric
	DropErrorMetrics()
	// FilterScrapeErrorsTotal filters the scrape_errors_total metric family to only include metrics for the jobs in
	// the jobFilters list.
	FilterScrapeErrorsTotal([]*dto.MetricFamily) []*dto.MetricFamily
}

type exporter struct {
	config     *config.Config
	targets    []Target
	jobFilters []string

	ctx      context.Context
	registry prometheus.Registerer
}

// NewExporter returns a new Exporter with the provided config.
func NewExporter(configFile string, registry prometheus.Registerer) (Exporter, error) {
	c, err := config.Load(configFile)
	if err != nil {
		return nil, err
	}

	// Override the DSN if requested (and in single target mode).
	if config.DsnOverride != "" {
		if len(c.Jobs) > 0 {
			return nil, errors.New("the config.data-source-name flag only applies in single target mode")
		}
		c.Target.DSN = config.Secret(config.DsnOverride)
	}

	var targets []Target
	if c.Target != nil {
		target, err := NewTarget("", c.Target.Name, "", string(c.Target.DSN),
			c.Target.Collectors(), nil, c.Globals, c.Target.EnablePing)
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

	scrapeErrorsMetric, err = registerScrapeErrorMetric(registry)
	if err != nil {
		return nil, err
	}

	return &exporter{
		config:     c,
		targets:    targets,
		jobFilters: nil,
		ctx:        context.Background(),
		registry:   registry,
	}, nil
}

func (e *exporter) WithContext(ctx context.Context) Exporter {
	return &exporter{
		config:     e.config,
		targets:    e.targets,
		jobFilters: e.jobFilters,
		ctx:        ctx,
		registry:   e.registry,
	}
}

// Gather implements prometheus.Gatherer. Should be called with a context-aware Exporter returned by WithContext() to
// ensure the context is respected by collectors.
func (e *exporter) Gather() ([]*dto.MetricFamily, error) {
	var (
		metricChan = make(chan Metric, capMetricChan)
		errs       prometheus.MultiError
	)

	// Take a local snapshot to avoid mutating e.targets while collectors are running.
	targets := e.filteredTargets()

	if len(targets) == 0 {
		return nil, errors.New("no targets found")
	}

	var wg sync.WaitGroup
	wg.Add(len(targets))
	for _, t := range targets {
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
				scrapeErrorsMetric.WithLabelValues(values...).Inc()
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

func (e *exporter) filteredTargets() []Target {
	if len(e.jobFilters) == 0 {
		return e.targets
	}

	var filtered []Target
	for _, target := range e.targets {
		if slices.Contains(e.jobFilters, target.JobGroup()) {
			filtered = append(filtered, target)
		}
	}
	return filtered
}

// FilterScrapeErrorsTotal filters the scrape_errors_total metric family to only include metrics for the jobs in the
// jobFilters list. If jobFilters is empty it returns the original metric families unmodified.
func (e *exporter) FilterScrapeErrorsTotal(mfs []*dto.MetricFamily) []*dto.MetricFamily {
	if len(e.jobFilters) == 0 {
		return mfs
	}

	result := make([]*dto.MetricFamily, 0, len(mfs))
	for _, mf := range mfs {
		if mf.GetName() != "scrape_errors_total" {
			result = append(result, mf)
			continue
		}

		// Filter metrics in this family to only include those with a job label in the jobFilters list.
		filtered := make([]*dto.Metric, 0, len(mf.Metric))
		for _, metric := range mf.Metric {
			for _, label := range metric.Label {
				if label.GetName() == "job" {
					if slices.Contains(e.jobFilters, label.GetValue()) {
						filtered = append(filtered, metric)
					}
					break
				}
			}
		}

		if len(filtered) > 0 {
			result = append(result, &dto.MetricFamily{
				Name:   mf.Name,
				Help:   mf.Help,
				Type:   mf.Type,
				Metric: filtered,
			})
		}
	}
	return result
}

// Config implements Exporter.
func (e *exporter) Config() *config.Config {
	return e.config
}

// UpdateTarget implements Exporter.
func (e *exporter) UpdateTarget(target []Target) {
	e.targets = target
}

// SetJobFilters implements Exporter.
func (e *exporter) SetJobFilters(filters []string) error {
	// If the filters list contains a single empty string, treat it as no filters.
	if len(filters) == 0 || (len(filters) == 1 && filters[0] == "") {
		slog.Debug("Received empty job filter, treating as no filters")
		e.jobFilters = nil
		return nil
	}

	// Single target mode has no jobs - filters are not applicable
	if len(e.config.Jobs) == 0 {
		slog.Warn("Job filters are not applicable in single target mode, ignoring", "filters", filters)
		e.jobFilters = nil
		return nil
	}

	for _, name := range filters {
		if !slices.ContainsFunc(e.config.Jobs, func(j *config.JobConfig) bool {
			return j.Name == name
		}) {
			return fmt.Errorf("invalid job name: %s", name)
		}
	}

	e.jobFilters = filters
	return nil
}

// DropErrorMetrics implements Exporter.
func (e *exporter) DropErrorMetrics() {
	scrapeErrorsMetric.Reset()
	slog.Debug("Dropped scrape_errors_total metric")
}

// registerScrapeErrorMetric registers the metrics for the exporter itself.
func registerScrapeErrorMetric(registry prometheus.Registerer) (*prometheus.CounterVec, error) {
	scrapeErrors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "scrape_errors_total",
		Help: "Total number of scrape errors per job, target, collector and query",
	}, svcMetricLabels)

	if err := registry.Register(scrapeErrors); err != nil {
		var alreadyRegisteredErr prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegisteredErr) {
			slog.Debug("scrape_errors_total metric already registered, using existing metric")
			return alreadyRegisteredErr.ExistingCollector.(*prometheus.CounterVec), nil
		}
		slog.Error("failed to register scrape_errors_total metric", "error", err)
		return nil, err
	}
	return scrapeErrors, nil
}

// split comma separated list of key=value pairs and return a map of key value pairs
func parseContextLog(list string) map[string]string {
	m := make(map[string]string)
	for item := range strings.SplitSeq(list, ",") {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			slog.Warn("Invalid context log item, ignoring", "item", item)
			continue
		}
		m[parts[0]] = parts[1]
	}
	return m
}

// TrimMissingCtx trims the leading comma and space from the log context string.
// Leading comma appears when previous parameter is undefined, which is a side-effect of running in single target mode.
func TrimMissingCtx(logContext string) string {
	return strings.TrimPrefix(logContext, ", ")
}
