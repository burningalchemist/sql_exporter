//go:build nomysql

package sql_exporter

import (
	"errors"
	"net/url"
)

// There are no TLS parameters to strip when MySQL support is disabled, but we need to define the variable to avoid compilation errors in sql.go.
var mysqlTLSParams = []string{}

// registerMySQLTLSConfig is a stub function that returns an error indicating that MySQL TLS support is disabled when the "nomysql" build tag is used.
func handleMySQLTLSConfig(_ url.Values) error {
	return errors.New("MySQL TLS support disabled (built with -tags nomysql)")
}
