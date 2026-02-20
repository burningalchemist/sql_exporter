//go:build no_mysql

package sql_exporter

import (
	"errors"
	"net/url"
)

// registerMySQLTLSConfig is a stub function that returns an error indicating that MySQL TLS support is disabled when the "no_mysql" build tag is used.
func registerMySQLTLSConfig(_ url.Values) error {
	return errors.New("MySQL TLS support disabled (built with -tags no_mysql)")
}
