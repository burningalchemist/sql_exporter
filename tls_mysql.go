//go:build !no_mysql

package sql_exporter

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/go-sql-driver/mysql"
)

const (
	mysqlTLSParamCACert     = "tls-ca-cert"
	mysqlTLSParamClientCert = "tls-client-cert"
	mysqlTLSParamClientKey  = "tls-client-key"
)

// mysqlTLSParams holds all custom TLS DSN parameters that must be stripped
// before passing the DSN to the MySQL driver.
var mysqlTLSParams = []string{mysqlTLSParamCACert, mysqlTLSParamClientCert, mysqlTLSParamClientKey}

// registerMySQLTLSConfig registers a custom TLS configuration for MySQL if the "tls" parameter is set to "custom" in the provided URL parameters.
func registerMySQLTLSConfig(params url.Values) error {
	caCert := params.Get(mysqlTLSParamCACert)
	clientCert := params.Get(mysqlTLSParamClientCert)
	clientKey := params.Get(mysqlTLSParamClientKey)
	slog.Debug("params", params)

	slog.Debug("Registering MySQL TLS configuration")
	slog.Debug("CA Certificate", "path", caCert)
	slog.Debug("Client Certificate", "path", clientCert)
	slog.Debug("Client Key", "path", clientKey)

	var rootCertPool *x509.CertPool
	if caCert != "" {
		rootCertPool = x509.NewCertPool()
		pem, err := os.ReadFile(caCert)
		if err != nil {
			return fmt.Errorf("failed to read CA certificate: %w", err)
		}
		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			return errors.New("failed to append PEM")
		}
	}

	var certs []tls.Certificate
	if clientCert != "" || clientKey != "" {
		if clientCert == "" || clientKey == "" {
			return errors.New("both tls-client-cert and tls-client-key must be provided for client authentication")
		}
		cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
		if err != nil {
			return fmt.Errorf("failed to load client certificate and key: %w", err)
		}
		certs = append(certs, cert)
	}

	tlsConfig := &tls.Config{
		RootCAs:      rootCertPool,
		Certificates: certs,
		MinVersion:   tls.VersionTLS12,
	}

	slog.Debug("MySQL TLS configuration registered successfully")
	slog.Debug("TLS Config", "RootCAs", rootCertPool, "Certificates", certs)
	return mysql.RegisterTLSConfig("custom", tlsConfig)
}
