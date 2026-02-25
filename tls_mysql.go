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

	onceMap sync.Map
)

// handleMySQLTLSConfig wraps the registration of a MySQL TLS configuration in a thread-safe manner. It uses a
// sync.Once to ensure that the TLS configuration for a given config name is registered only once, even if multiple
// goroutines attempt to register it concurrently.
func handleMySQLTLSConfig(configName string, params url.Values) error {
	onceConn, _ := onceMap.LoadOrStore(configName, &sync.Once{})
	once := onceConn.(*sync.Once)
	var err error
	once.Do(func() {
		err = registerMySQLTLSConfig(configName, params)
		if err != nil {
			slog.Error("Failed to register MySQL TLS config", "error", err)
		}
	})
	return err
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
