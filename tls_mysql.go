//go:build !no_mysql

package sql_exporter

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/go-sql-driver/mysql"
)

// registerTLSConfig registers a custom TLS configuration for MySQL if the "tls" parameter is set to "custom" in the provided URL parameters.
func registerMySQLTLSConfig(params url.Values) error {
	caCert := params.Get("tls-ca-cert")
	clientCert := params.Get("tls-client-cert")
	clientKey := params.Get("tls-client-key")

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
	}
	return mysql.RegisterTLSConfig("custom", tlsConfig)
}
