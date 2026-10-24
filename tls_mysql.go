//go:build !nomysql

package sql_exporter

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"sync"

	"github.com/go-sql-driver/mysql"
)

const (
	mysqlTLSParamCACert     = "tls-ca"
	mysqlTLSParamClientCert = "tls-cert"
	mysqlTLSParamClientKey  = "tls-key"
)

// mysqlTLSParams is a list of TLS parameters that can be used in MySQL DSNs. It is used to identify and strip TLS
// parameters from the DSN after registering the TLS configuration, as these parameters are not recognized by the MySQL
// driver and would cause connection failure if left in the DSN.
var (
	mysqlTLSParams = []string{mysqlTLSParamCACert, mysqlTLSParamClientCert, mysqlTLSParamClientKey}

	mysqlTLSMu         sync.Mutex
	mysqlTLSRegistered = map[string]bool{}
)

// handleMySQLTLSConfig wraps the registration of a MySQL TLS configuration in a thread-safe manner. A configuration is
// registered once per config name, even if multiple goroutines attempt to register it concurrently. A failed
// registration is not remembered, so the next call for the same name tries again and every caller gets the result of
// its own attempt.
func handleMySQLTLSConfig(configName string, params url.Values) error {
	mysqlTLSMu.Lock()
	defer mysqlTLSMu.Unlock()

	if mysqlTLSRegistered[configName] {
		return nil
	}
	if err := registerMySQLTLSConfig(configName, params); err != nil {
		slog.Error("Failed to register MySQL TLS config", "error", err)
		return err
	}
	mysqlTLSRegistered[configName] = true
	return nil
}

// registerMySQLTLSConfig registers a custom TLS configuration for MySQL with the given config name and parameters.
func registerMySQLTLSConfig(configName string, params url.Values) error {
	caCert := params.Get(mysqlTLSParamCACert)
	clientCert := params.Get(mysqlTLSParamClientCert)
	clientKey := params.Get(mysqlTLSParamClientKey)

	slog.Debug("MySQL TLS config", "configName", configName, mysqlTLSParamCACert, caCert,
		mysqlTLSParamClientCert, clientCert, mysqlTLSParamClientKey, clientKey)

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
			return errors.New("both tls-cert and tls-key must be provided for client authentication")
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

	return mysql.RegisterTLSConfig(configName, tlsConfig)
}
