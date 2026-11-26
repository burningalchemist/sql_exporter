//go:build !nomysql

package sql_exporter

import (
	"context"
	"strings"
	"testing"

	"github.com/xo/dburl"
)

func TestMySQLAWSIAMConfig(t *testing.T) {
	u, err := dburl.Parse("mysql://mon@db.example.com/mydb?tls=true&aws-iam-auth=true&aws-iam-region=us-east-1" +
		"&aws-iam-role-arn=arn:aws:iam::123456789012:role/rds-monitor")
	if err != nil {
		t.Fatal(err)
	}

	c, err := mysqlAWSIAMConfig(u.DSN, testAWSConfig())
	if err != nil {
		t.Fatal(err)
	}

	for _, param := range []string{awsIAMParamAuth, awsIAMParamRegion, awsIAMParamRoleARN} {
		if _, ok := c.Params[param]; ok {
			t.Errorf("%s is not removed from the parameters", param)
		}
	}

	if !c.AllowCleartextPasswords {
		t.Error("AllowCleartextPasswords is not set")
	}
}

func TestMySQLAWSIAMConfigRequiresTLS(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
	}{
		{name: "no tls", dsn: "mysql://mon@db.example.com/mydb?aws-iam-auth=true"},
		{name: "tls=false", dsn: "mysql://mon@db.example.com/mydb?tls=false&aws-iam-auth=true"},
		{name: "tls=preferred", dsn: "mysql://mon@db.example.com/mydb?tls=preferred&aws-iam-auth=true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := dburl.Parse(tt.dsn)
			if err != nil {
				t.Fatal(err)
			}

			_, err = mysqlAWSIAMConfig(u.DSN, testAWSConfig())
			if err == nil || !strings.Contains(err.Error(), "requires TLS") {
				t.Fatalf("got error %v, want %q", err, "requires TLS")
			}
		})
	}
}

func TestMySQLAWSIAMHook(t *testing.T) {
	tests := []struct {
		name         string
		dsn          string
		wantEndpoint string
	}{
		{
			name:         "default port",
			dsn:          "mysql://mon@db.example.com/mydb?tls=true&aws-iam-auth=true",
			wantEndpoint: "db.example.com:3306",
		},
		{
			name:         "custom port",
			dsn:          "mysql://mon@db.example.com:3307/mydb?tls=true&aws-iam-auth=true",
			wantEndpoint: "db.example.com:3307",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := dburl.Parse(tt.dsn)
			if err != nil {
				t.Fatal(err)
			}

			c, err := mysqlAWSIAMConfig(u.DSN, testAWSConfig())
			if err != nil {
				t.Fatal(err)
			}

			if err := mysqlAWSIAMHook(testAWSConfig())(context.Background(), c); err != nil {
				t.Fatal(err)
			}

			checkAWSIAMToken(t, c.Passwd, tt.wantEndpoint, "mon")
		})
	}
}
