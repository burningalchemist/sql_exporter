package config

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type awsSecretsManagerProvider struct{}

// getDSN fetches the secret value from AWS Secrets Manager.
// URL format: awssecretsmanager://secret-name?region=us-east-1
func (p awsSecretsManagerProvider) getDSN(ctx context.Context, ref *url.URL) (string, error) {
	opts := []func(*awsconfig.LoadOptions) error{}

	if region := ref.Query().Get("region"); region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("unable to load AWS config: %w", err)
	}

	svc := secretsmanager.NewFromConfig(cfg)

	secretName := ref.Host
	if ref.Path != "" {
		secretName += ref.Path
	}

	result, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"),
	})
	if err != nil {
		return "", fmt.Errorf("unable to get secret value: %w", err)
	}

	return *result.SecretString, nil
}
