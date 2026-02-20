//go:build no_mysql

package sql_exporter

import (
	"errors"
	"net/url"
)

func registerMySQLTLSConfig(params url.Values) error {
	return errors.New("MySQL TLS support disabled (built with -tags no_mysql)")
}
