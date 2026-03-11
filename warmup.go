package sql_exporter

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Warmup pre-populates collector caches on startup by sequentially triggering
// each collector with a configurable delay between them, avoiding the thundering
// herd problem where all collectors hit the database simultaneously on first scrape.
type Warmup struct {
	done chan struct{}
}

// NewWarmup creates a new Warmup instance.
func NewWarmup() *Warmup {
	return &Warmup{done: make(chan struct{})}
}

// Wait blocks until warmup is complete or the context is cancelled.
func (w *Warmup) Done() bool {
	select {
	case <-w.done:
		return true
	default:
		return false
	}
}

// Run sequentially triggers each collector across all targets with a delay
// between each, pre-populating the in-memory caches before the first real scrape.
func (w *Warmup) Run(targets []Target, delay time.Duration, timeout time.Duration) {
	defer close(w.done)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var wg sync.WaitGroup
	for _, t := range targets {
		tc, ok := t.(*target)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(tc *target) {
			defer wg.Done()
			if err := tc.ping(ctx); err != nil {
				slog.Warn("Warmup: skipping target, ping failed", "target", tc.name, "error", err)
				return
			}
			for i, c := range tc.collectors {
				if ctx.Err() != nil {
					return
				}
				ch := make(chan Metric, capMetricChan)
				var cwg sync.WaitGroup
				cwg.Add(1)
				go func() {
					defer cwg.Done()
					for range ch {
					}
				}()
				c.Collect(ctx, tc.conn, ch)
				close(ch)
				cwg.Wait()

				slog.Debug("Warmup collector done", "target", tc.name,
					"progress", fmt.Sprintf("%d/%d", i+1, len(tc.collectors)))

				if i < len(tc.collectors)-1 {
					select {
					case <-ctx.Done():
						return
					case <-time.After(delay):
					}
				}
			}
		}(tc)
	}
	wg.Wait()
	slog.Info("Warmup completed")
}

type warmupTask struct {
	target    *target
	collector Collector
}
