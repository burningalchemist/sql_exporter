//go:build !nomysql

package sql_exporter

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestHandleMySQLTLSConfigRetriesAfterFailure(t *testing.T) {
	dir := t.TempDir()
	name := "custom-retry-" + filepath.Base(filepath.Dir(dir))
	caPath := filepath.Join(dir, "ca.pem")
	params := url.Values{mysqlTLSParamCACert: {caPath}}

	if err := handleMySQLTLSConfig(name, params); err == nil {
		t.Fatal("expected an error while the CA file is missing")
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, priv)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := handleMySQLTLSConfig(name, params); err != nil {
		t.Fatalf("expected the second call to succeed: %v", err)
	}
	if _, err := mysql.ParseDSN("u@tcp(127.0.0.1:3306)/db?tls=" + name); err != nil {
		t.Fatalf("config %q is not registered: %v", name, err)
	}
}
