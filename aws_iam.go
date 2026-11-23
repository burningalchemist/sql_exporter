package sql_exporter

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/xo/dburl"
)

// This file should not import mysql and pgx drivers to allow `nomysql` and `nopgx` build tags.

const (
	// awsIAMParamAuth is the DSN parameter that enables AWS IAM authentication when set to "true".
	awsIAMParamAuth = "aws-iam-auth"
	// awsIAMParamRegion is the optional DSN parameter for the region the token is signed for.
	// It defaults to the region of the AWS environment.
	awsIAMParamRegion = "aws-iam-region"
	// awsIAMParamRoleARN is the optional DSN parameter for the ARN of an IAM role to assume for the token.
	awsIAMParamRoleARN = "aws-iam-role-arn"
)

// awsIAMOpener opens a DB handle for one driver. It gets a DSN without a password but with our parameters, which it
// must remove, and must set a new token as the password for every new connection.
type awsIAMOpener func(dsn string, cfg aws.Config) (*sql.DB, error)

// awsIAMOpeners has the opener of each supported driver. Each driver registers itself in the init function of its own
// file, so the "nomysql" and "nopgx" build tags can leave it out.
var awsIAMOpeners = map[string]awsIAMOpener{}

// openWithAWSIAM opens a DB handle that authenticates with AWS IAM, using the default AWS credential chain.
//
// Tokens expire after 15 minutes, so the before connect hook of the driver sets a new token as the password for every
// new connection. That's why only the drivers with such a hook (pgx and mysql) are used.
func openWithAWSIAM(ctx context.Context, driverName string, dbURL *dburl.URL) (*sql.DB, error) {
	open, ok := awsIAMOpeners[driverName]
	if !ok {
		return nil, fmt.Errorf("%s is only supported for the %v drivers in this build, not %q",
			awsIAMParamAuth, slices.Sorted(maps.Keys(awsIAMOpeners)), driverName)
	}

	// The token replaces the password. We don't silently ignore a password, the user may think it's still in use.
	if _, hasPassword := dbURL.User.Password(); hasPassword || dbURL.Query().Has("password") {
		return nil, fmt.Errorf("%s cannot be used with a password in the DSN", awsIAMParamAuth)
	}

	// The region comes from the DSN parameter if set, then the AWS environment, then the EC2 instance metadata.
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(dbURL.Query().Get(awsIAMParamRegion)),
		awsconfig.WithEC2IMDSRegion(),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("AWS region is not set, use the %s parameter", awsIAMParamRegion)
	}

	// The cache calls STS only when the credentials of the role expire, not for every new connection.
	if roleARN := dbURL.Query().Get(awsIAMParamRoleARN); roleARN != "" {
		cfg.Credentials = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(sts.NewFromConfig(cfg), roleARN))
	}

	return open(dbURL.DSN, cfg)
}

// buildAWSIAMToken builds a token that is signed for the endpoint (host:port), region and user of the database.
func buildAWSIAMToken(ctx context.Context, cfg aws.Config, endpoint, user string) (string, error) {
	token, err := auth.BuildAuthToken(ctx, endpoint, cfg.Region, user, cfg.Credentials)
	if err != nil {
		return "", fmt.Errorf("failed to build AWS IAM auth token: %w", err)
	}
	return token, nil
}
