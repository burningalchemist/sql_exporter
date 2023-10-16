// Copyright 2022 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/burningalchemist/sql_exporter"
)

func handleScrape(configFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Query().Get("target")
		if target == "" {
			http.Error(w, "target is required", http.StatusBadRequest)
			return
		}

		var collectors []string
		collectorStr := r.URL.Query().Get("collectors")
		if collectorStr != "" {
			collectors = strings.Split(collectorStr, ",")
		}

		exporter, err := sql_exporter.NewExporter(configFile, target, collectors)

		// Cleanup underlying connections to prevent connection leaks
		defer func() {
			for _, target := range exporter.GetTarget() {
				target.DB().Close()
			}
		}()

		if err != nil {
			http.Error(w, fmt.Sprintf("ERROR: %v", err), http.StatusBadRequest)
		} else {
			ExporterHandlerFor(exporter).ServeHTTP(w, r)
		}
	}
}
