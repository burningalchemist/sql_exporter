package sql_exporter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/xo/dburl"
)

// OpenConnection parses a provided DSN, and opens a DB handle ensuring early termination if the context is closed
// (this is actually prevented by `database/sql` implementation), sets connection limits and returns the handle.
func OpenConnection(ctx context.Context, logContext, dsn string, maxConns, maxIdleConns int, maxConnLifetime time.Duration) (*sql.DB, error) {
	var (
		url  *dburl.URL
		conn *sql.DB
		err  error
		ch   = make(chan error)
	)

	url, err = safeParse(dsn)
	if err != nil {
		return nil, err
	}

	driver := url.Driver
	if url.GoDriver != "" {
		driver = url.GoDriver
	}

	// Register custom TLS config for MySQL if needed
	if driver == "mysql" && url.Query().Get("tls") == "custom" {
		if err := registerMySQLTLSConfig(url.Query()); err != nil {
			return nil, fmt.Errorf("failed to register MySQL TLS config: %w", err)
		}

		// Strip TLS parameters from the URL as they are interpreted as system variables by the MySQL driver which
		// causes connection failure. The TLS configuration is already registered globally.
		q := url.Query()
		for _, param := range mysqlTLSParams {
			q.Del(param)
		}
		url.RawQuery = q.Encode()
		// Regenerate the DSN without TLS parameters for logging and connection purposes
		tlsStripped, _, err := dburl.GenMysql(url)
		if err != nil {
			return nil, fmt.Errorf("failed to generate MySQL DSN: %w", err)
		}
		url.DSN = tlsStripped
	}

	// Open the DB handle in a separate goroutine so we can terminate early if the context closes.
	go func() {
		conn, err = sql.Open(driver, url.DSN)
		close(ch)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ch:
		if err != nil {
			return nil, err
		}
	}

	conn.SetMaxIdleConns(maxIdleConns)
	conn.SetMaxOpenConns(maxConns)
	conn.SetConnMaxLifetime(maxConnLifetime)

	slog.Debug("Database handle successfully opened", "logContext", logContext, "driver", driver)
	return conn, nil
}

// PingDB is a wrapper around sql.DB.PingContext() that terminates as soon as the context is closed.
//
// sql.DB does not actually pass along the context to the driver when opening a connection (which always happens if the
// database is down) and the driver uses an arbitrary timeout which may well be longer than ours. So we run the ping
// call in a goroutine and terminate immediately if the context is closed.
func PingDB(ctx context.Context, conn *sql.DB) error {
	ch := make(chan error, 1)

	go func() {
		ch <- conn.PingContext(ctx)
		close(ch)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ch:
		return err
	}
}

// safeParse wraps dburl.Parse method in order to prevent leaking credentials
// if underlying url parse failed. By default it returns a raw url string in error message,
// which most likely contains a password. It's undesired here.
func safeParse(rawURL string) (*dburl.URL, error) {
	parsed, err := dburl.Parse(expandEnv(rawURL))
	if err != nil {
		if uerr := new(url.Error); errors.As(err, &uerr) {
			return nil, uerr.Err
		}
		return nil, errors.New("invalid URL")
	}
	return parsed, nil
}

// expandEnv falls back to the original env variable if not found for better readability
func expandEnv(env string) string {
	lookupFunc := func(env string) string {
		if value, ok := os.LookupEnv(env); ok {
			return value
		}
		slog.Error("Environment variable is not found, cannot expand", "env", env)
		return fmt.Sprintf("$%s", env)
	}
	return os.Expand(env, lookupFunc)
}
