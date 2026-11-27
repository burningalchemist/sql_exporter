package sql_exporter

import (
	"database/sql"
	"slices"

	"github.com/xo/dburl"
)

type overrideRule struct {
	shadowedDriver string
	claims         []string
	fallback       []string
}

var overrideMap = map[string]overrideRule{
	// moderncsqlite driver is used if available, otherwise falls back to mattn/go-sqlite3 (requires CGO)
	"sqlite": {
		shadowedDriver: "sqlite3",
		claims:         []string{"mq", "moderncsqlite", "modernsqlite"},
		fallback:       []string{"sqlite3", "file"},
	},
	// pgx driver is used as the new default, otherwise falls back to lib/pq
	"postgres": {
		shadowedDriver: "pgx",
		claims:         []string{"pq", "libpq", "postgresql", "pg", "pgsql"},
		fallback:       []string{"pgx", "px"},
	},
}

// The purpose of this init call is to remap certain database driver schemes to their available implementations. This
// ensures that when a user specifies a driver, it resolves to the correct implementation, even if multiple drivers
// claim the same name.
func init() {
	available := sql.Drivers()

	for targetDriver, rule := range overrideMap {
		if !slices.Contains(available, targetDriver) {
			continue
		}

		if slices.Contains(available, rule.shadowedDriver) {
			continue
		}

		var baseScheme *dburl.Scheme
		for _, name := range slices.Concat(rule.claims, []string{rule.shadowedDriver, targetDriver}) {
			if old := dburl.Unregister(name); old != nil && baseScheme == nil {
				baseScheme = old
			}
		}
		// Fallback to a default scheme if none was found
		if baseScheme == nil {
			baseScheme = &dburl.Scheme{}
		}

		// Re-register with the remapped drivers
		origGen := baseScheme.Generator
		driver := targetDriver
		gen := func(u *dburl.URL) (string, string, error) {
			dsn, drv, err := origGen(u)
			if drv == rule.shadowedDriver || drv == "" {
				drv = driver
			}
			return dsn, drv, err
		}

		dburl.Register(dburl.Scheme{
			Name:      targetDriver,
			Generator: gen,
			Transport: baseScheme.Transport,
			Opaque:    baseScheme.Opaque,
			Dialect:   baseScheme.Dialect,
			Aliases:   slices.Concat(rule.claims, rule.fallback),
		})
	}
}
