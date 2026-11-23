//go:build !nopgx

package sql_exporter

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func init() {
	awsIAMOpeners["pgx"] = openPgxWithAWSIAM
}

func openPgxWithAWSIAM(dsn string, cfg aws.Config) (*sql.DB, error) {
	c, err := pgxAWSIAMConfig(dsn)
	if err != nil {
		return nil, err
	}

	return stdlib.OpenDB(*c, stdlib.OptionBeforeConnect(pgxAWSIAMHook(cfg))), nil
}

// pgxAWSIAMConfig parses the DSN, removes our parameters from it and checks that it has a single host with TLS.
func pgxAWSIAMConfig(dsn string) (*pgx.ConnConfig, error) {
	c, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// pgx sends the parameters it doesn't know to the server, which would reject ours.
	delete(c.RuntimeParams, awsIAMParamAuth)
	delete(c.RuntimeParams, awsIAMParamRegion)
	delete(c.RuntimeParams, awsIAMParamRoleARN)

	// IAM authentication needs TLS and a single host. pgx keeps non-TLS attempts and extra hosts in Fallbacks.
	if c.TLSConfig == nil || len(c.Fallbacks) > 0 {
		return nil, fmt.Errorf("%s requires a single host and sslmode set to require, verify-ca or verify-full",
			awsIAMParamAuth)
	}

	return c, nil
}

// pgxAWSIAMHook returns a before connect hook that sets a new token as the password.
func pgxAWSIAMHook(cfg aws.Config) func(context.Context, *pgx.ConnConfig) error {
	return func(ctx context.Context, c *pgx.ConnConfig) (err error) {
		endpoint := net.JoinHostPort(c.Host, strconv.Itoa(int(c.Port)))
		c.Password, err = buildAWSIAMToken(ctx, cfg, endpoint, c.User)
		return err
	}
}
