//go:build !nomysql

package sql_exporter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-sql-driver/mysql"
)

func init() {
	awsIAMOpeners["mysql"] = openMySQLWithAWSIAM
}

func openMySQLWithAWSIAM(dsn string, cfg aws.Config) (*sql.DB, error) {
	c, err := mysqlAWSIAMConfig(dsn, cfg)
	if err != nil {
		return nil, err
	}

	connector, err := mysql.NewConnector(c)
	if err != nil {
		return nil, err
	}

	return sql.OpenDB(connector), nil
}

// mysqlAWSIAMConfig parses the DSN, removes our parameters from it and sets what AWS IAM authentication needs.
func mysqlAWSIAMConfig(dsn string, cfg aws.Config) (*mysql.Config, error) {
	c, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}

	// The driver sends the parameters it doesn't know to the server, which would reject ours.
	delete(c.Params, awsIAMParamAuth)
	delete(c.Params, awsIAMParamRegion)
	delete(c.Params, awsIAMParamRoleARN)

	// RDS expects the token as cleartext, which the driver refuses to send by default.
	c.AllowCleartextPasswords = true

	// RDS doesn't accept IAM authentication without TLS, so we reject a DSN that can connect without it.
	if c.TLS == nil || c.AllowFallbackToPlaintext {
		return nil, fmt.Errorf("%s requires TLS, set tls to true, skip-verify or custom", awsIAMParamAuth)
	}

	if err := c.Apply(mysql.BeforeConnect(mysqlAWSIAMHook(cfg))); err != nil {
		return nil, err
	}

	return c, nil
}

// mysqlAWSIAMHook returns a before connect hook that sets a new token as the password.
func mysqlAWSIAMHook(cfg aws.Config) func(context.Context, *mysql.Config) error {
	return func(ctx context.Context, c *mysql.Config) (err error) {
		// Addr is host:port, the driver adds the default port when the DSN has none.
		c.Passwd, err = buildAWSIAMToken(ctx, cfg, c.Addr, c.User)
		return err
	}
}
