package sql_exporter

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/burningalchemist/sql_exporter/config"
	"github.com/burningalchemist/sql_exporter/errors"
	dto "github.com/prometheus/client_model/go"
)

// Collector is a self-contained group of SQL queries and metric families to collect from a specific database. It is
// conceptually similar to a prometheus.Collector.
type Collector interface {
	// Collect is the equivalent of prometheus.Collector.Collect() but takes a context to run in and a database to run on.
	Collect(context.Context, *sql.DB, chan<- Metric)
}

// collector implements Collector. It wraps a collection of queries, metrics and the database to collect them from.
type collector struct {
	config     *config.CollectorConfig
	queries    []*Query
	logContext string
}

// NewCollector returns a new Collector with the given configuration and database. The metrics it creates will all have
// the provided const labels applied.
func NewCollector(logContext string, cc *config.CollectorConfig, constLabels []*dto.LabelPair) (Collector, errors.WithContext) {
	logContext = TrimMissingCtx(fmt.Sprintf(`%s,collector=%s`, logContext, cc.Name))

	// Maps each query to the list of metric families it populates.
	queryMFs := make(map[*config.QueryConfig][]*MetricFamily, len(cc.Metrics))

	// Instantiate metric families.
	for _, mc := range cc.Metrics {
		mf, err := NewMetricFamily(logContext, mc, constLabels)
		if err != nil {
			return nil, err
		}
		mfs, found := queryMFs[mc.Query()]
		if !found {
			mfs = make([]*MetricFamily, 0, 2)
		}
		queryMFs[mc.Query()] = append(mfs, mf)
	}

	// Instantiate queries.
	queries := make([]*Query, 0, len(cc.Metrics))
	for qc, mfs := range queryMFs {
		q, err := NewQuery(logContext, qc, mfs...)
		if err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}

	c := collector{
		config:     cc,
		queries:    queries,
		logContext: logContext,
	}
	if c.config.MinInterval > 0 {
		slog.Warn("Non-zero min_interval, using cached collector.", "logContext", logContext, "min_interval", c.config.MinInterval)
		return newCachingCollector(&c), nil
	}
	return &c, nil
}

// Collect implements Collector.
func (c *collector) Collect(ctx context.Context, conn *sql.DB, ch chan<- Metric) {
	var wg sync.WaitGroup
	wg.Add(len(c.queries))
	for _, q := range c.queries {
		go func(q *Query) {
			defer wg.Done()
			q.Collect(ctx, conn, ch)
		}(q)
	}
	// Only return once all queries have been processed
	wg.Wait()
}

// newCachingCollector returns a new Collector wrapping the provided raw Collector.
func newCachingCollector(rawColl *collector) Collector {
	cc := &cachingCollector{
		rawColl:     rawColl,
		minInterval: time.Duration(rawColl.config.MinInterval),
		cacheSem:    make(chan time.Time, 1),
	}
	cc.cacheSem <- time.Time{}
	return cc
}

// Collector with a cache for collected metrics. Only used when min_interval is non-zero.
type cachingCollector struct {
	// Underlying collector, which is being cached.
	rawColl *collector
	// Convenience copy of rawColl.config.MinInterval.
	minInterval time.Duration

	// Used as a non=blocking semaphore protecting the cache. The value in the channel is the time of the cached metrics.
	cacheSem chan time.Time
	// Metrics saved from the last Collect() call.
	cache []Metric
}

// Collect implements Collector.
func (cc *cachingCollector) Collect(ctx context.Context, conn *sql.DB, ch chan<- Metric) {
	if ctx.Err() != nil {
		ch <- NewInvalidMetric(errors.Wrap(cc.rawColl.logContext, ctx.Err()))
		return
	}
	slog.Debug("Cache size", "length", len(cc.cache))
	collTime := time.Now()
	select {
	case cacheTime := <-cc.cacheSem:
		// Have the lock.
		if age := collTime.Sub(cacheTime); age > cc.minInterval || len(cc.cache) == 0 {
			// Cache contents are older than minInterval, collect fresh metrics, cache them and pipe them through.
			slog.Debug("Collecting fresh metrics", "logContext", cc.rawColl.logContext, "min_interval", cc.minInterval.Seconds(), "cache_age", age.Seconds())
			cacheChan := make(chan Metric, capMetricChan)
			cc.cache = make([]Metric, 0, len(cc.cache))
			go func() {
				cc.rawColl.Collect(ctx, conn, cacheChan)
				close(cacheChan)
			}()
			for metric := range cacheChan {
				// catch invalid metrics and return them immediately, don't cache them
				if ctx.Err() != nil {
					slog.Debug("Context closed, returning invalid metric", "logContext", cc.rawColl.logContext)
					ch <- NewInvalidMetric(errors.Wrap(cc.rawColl.logContext, ctx.Err()))
					continue
				}

				cc.cache = append(cc.cache, metric)
				ch <- metric
			}
			cacheTime = collTime
		} else {
			slog.Debug("Returning cached metrics", "logContext", cc.rawColl.logContext, "min_interval", cc.minInterval.Seconds(), "cache_age", age.Seconds())
			for _, metric := range cc.cache {
				ch <- metric
			}
		}
		// Always replace the value in the semaphore channel.
		cc.cacheSem <- cacheTime

	case <-ctx.Done():
		// Context closed, record an error and return
		ch <- NewInvalidMetric(errors.Wrap(cc.rawColl.logContext, ctx.Err()))
	}
}
