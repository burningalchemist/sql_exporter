package sql_exporter

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// testAWSConfig returns a config with fake credentials, so the tests don't depend on the AWS environment.
// Signing a token doesn't call AWS.
func testAWSConfig() aws.Config {
	return aws.Config{
		Region: "eu-west-1",
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "AKID", SecretAccessKey: "SECRET"}, nil
		}),
	}
}

// checkAWSIAMToken checks the endpoint, user and region the token is signed for.
func checkAWSIAMToken(t *testing.T, token, endpoint, user string) {
	t.Helper()
	for _, want := range []string{endpoint + "?", "DBUser=" + user + "&", "%2Feu-west-1%2Frds-db%2F"} {
		if !strings.Contains(token, want) {
			t.Errorf("token %q doesn't contain %q", token, want)
		}
	}
}
