//go:build !nopgx

package sql_exporter

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xo/dburl"
)

// The password check is the same for every driver. The test is in this file because it needs a supported driver.
func TestOpenWithAWSIAMRejectsPassword(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr string
	}{
		{
			name:    "password in the URL",
			dsn:     "pgx://mon:secret@db.example.com/postgres?sslmode=require&aws-iam-auth=true",
			wantErr: "cannot be used with a password",
		},
		{
			name:    "password as a parameter",
			dsn:     "pgx://mon@db.example.com/postgres?sslmode=require&password=secret&aws-iam-auth=true",
			wantErr: "cannot be used with a password",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := OpenConnection(context.Background(), "", tt.dsn, 1, 1, time.Minute)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("got error %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestPgxAWSIAMConfigRemovesParams(t *testing.T) {
	u, err := dburl.Parse("pgx://mon@db.example.com/postgres?sslmode=require&aws-iam-auth=true" +
		"&aws-iam-region=us-east-1&aws-iam-role-arn=arn:aws:iam::123456789012:role/rds-monitor")
	if err != nil {
		t.Fatal(err)
	}

	c, err := pgxAWSIAMConfig(u.DSN)
	if err != nil {
		t.Fatal(err)
	}

	for _, param := range []string{awsIAMParamAuth, awsIAMParamRegion, awsIAMParamRoleARN} {
		if _, ok := c.RuntimeParams[param]; ok {
			t.Errorf("%s is not removed from the parameters", param)
		}
	}
}

func TestPgxAWSIAMConfigRequiresTLSAndSingleHost(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
	}{
		{name: "sslmode=disable", dsn: "pgx://mon@db.example.com/postgres?sslmode=disable&aws-iam-auth=true"},
		{name: "sslmode=allow", dsn: "pgx://mon@db.example.com/postgres?sslmode=allow&aws-iam-auth=true"},
		{name: "sslmode=prefer", dsn: "pgx://mon@db.example.com/postgres?sslmode=prefer&aws-iam-auth=true"},
		{name: "multiple hosts", dsn: "pgx://mon@db1.example.com,db2.example.com/postgres?sslmode=require&aws-iam-auth=true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := dburl.Parse(tt.dsn)
			if err != nil {
				t.Fatal(err)
			}

			_, err = pgxAWSIAMConfig(u.DSN)
			if err == nil || !strings.Contains(err.Error(), "requires a single host") {
				t.Fatalf("got error %v, want %q", err, "requires a single host")
			}
		})
	}
}

func TestPgxAWSIAMHook(t *testing.T) {
	tests := []struct {
		name         string
		dsn          string
		wantEndpoint string
		wantUser     string
	}{
		{
			name:         "default port",
			dsn:          "pgx://mon@db.example.com/postgres?sslmode=require&aws-iam-auth=true",
			wantEndpoint: "db.example.com:5432",
			wantUser:     "mon",
		},
		{
			name:         "custom port",
			dsn:          "pgx://mon@db.example.com:6432/postgres?sslmode=require&aws-iam-auth=true",
			wantEndpoint: "db.example.com:6432",
			wantUser:     "mon",
		},
		{
			// pgx lets the parameters override the URL, the token must be signed for what pgx connects with.
			name:         "user as a parameter",
			dsn:          "pgx://mon@db.example.com/postgres?user=other&sslmode=require&aws-iam-auth=true",
			wantEndpoint: "db.example.com:5432",
			wantUser:     "other",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := dburl.Parse(tt.dsn)
			if err != nil {
				t.Fatal(err)
			}

			c, err := pgxAWSIAMConfig(u.DSN)
			if err != nil {
				t.Fatal(err)
			}

			if err := pgxAWSIAMHook(testAWSConfig())(context.Background(), c); err != nil {
				t.Fatal(err)
			}

			checkAWSIAMToken(t, c.Password, tt.wantEndpoint, tt.wantUser)
		})
	}
}
