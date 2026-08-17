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
	"sqlite": {
		shadowedDriver: "sqlite3",
		claims:         []string{"mq", "moderncsqlite", "modernsqlite"},
		fallback:       []string{"sqlite", "sqlite3", "file"},
	},
	"pgx": {
		shadowedDriver: "postgres",
		claims:         []string{"pgx", "px"},
		fallback:       []string{"pg", "pgsql", "postgres", "postgresql"},
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

		schemes := append([]string{targetDriver}, rule.claims...)

		if !slices.Contains(available, rule.shadowedDriver) {
			schemes = append(schemes, rule.fallback...)
		}

		var baseScheme *dburl.Scheme

		// Unregister all aliases to clear possible conflicts
		for _, alias := range schemes {
			if old := dburl.Unregister(alias); old != nil && baseScheme == nil {
				baseScheme = old
			}
		}

		// Fallback to a default scheme if none was found
		if baseScheme == nil {
			baseScheme = &dburl.Scheme{}
		}

		// Re-register with the remapped drivers
		dburl.Register(dburl.Scheme{
			Driver:    targetDriver,
			Generator: baseScheme.Generator,
			Opaque:    baseScheme.Opaque,
			Aliases:   schemes,
		})
	}
}
