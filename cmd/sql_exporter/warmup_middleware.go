package main

import (
	"net/http"
	"time"

	sql_exporter "github.com/burningalchemist/sql_exporter"
)

// initWarmup starts the warmup process if configured and returns a middleware
// that blocks /metrics requests until warmup is complete.
func initWarmup(exporter sql_exporter.Exporter) func(http.Handler) http.Handler {
	wd := exporter.Config().Globals.WarmupDelay
	if wd == 0 {
		return nil
	}

	warmup := sql_exporter.NewWarmup()
	go warmup.Run(exporter.Targets(), time.Duration(wd), time.Duration(exporter.Config().Globals.ScrapeTimeout))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !warmup.Done() {
				http.Error(w, "Warmup in progress, please retry", http.StatusTooEarly)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
