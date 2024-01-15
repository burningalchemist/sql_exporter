package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"k8s.io/klog/v2"
)

//
// Target
//

// TargetConfig defines a DSN and a set of collectors to be executed on it.
type TargetConfig struct {
	Name          string   `yaml:"name,omitempty" env:"NAME"`               // name of the target
	DSN           Secret   `yaml:"data_source_name" env:"DSN"`              // data source name to connect to
	AwsSecretName string   `yaml:"aws_secret_name" env:"AWS_SECRET_NAME"`   // AWS secret name
	CollectorRefs []string `yaml:"collectors" env:"COLLECTORS"`             // names of collectors to execute on the target
	EnablePing    *bool    `yaml:"enable_ping,omitempty" env:"ENABLE_PING"` // ping the target before executing the collectors

	collectors []*CollectorConfig // resolved collector references

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]any `yaml:",inline" json:"-"`
}

// Collectors returns the collectors referenced by the target, resolved.
func (t *TargetConfig) Collectors() []*CollectorConfig {
	return t.collectors
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for TargetConfig.
func (t *TargetConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type plain TargetConfig
	if err := unmarshal((*plain)(t)); err != nil {
		return err
	}

	if t.AwsSecretName != "" {
		t.DSN = readDSNFromAwsSecretManager(t.AwsSecretName)
	}

	// Check required fields
	if t.DSN == "" {
		return fmt.Errorf("missing data_source_name for target %+v", t)
	}
	if err := checkCollectorRefs(t.CollectorRefs, "target"); err != nil {
		return err
	}

	return checkOverflow(t.XXX, "target")
}

// AWS Secret
type AwsSecret struct {
	DSN Secret `json:"data_source_name"`
}

func readDSNFromAwsSecretManager(secretName string) Secret {
	config, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithEC2IMDSRegion())
	if err != nil {
		klog.Fatal(err)
	}

	// Create Secrets Manager client
	svc := secretsmanager.NewFromConfig(config)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	klog.Infof("reading AWS Secret: %s", secretName)
	result, err := svc.GetSecretValue(context.TODO(), input)
	if err != nil {
		// For a list of exceptions thrown, see
		// https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html
		klog.Fatal(err.Error())
	}

	// Decrypts secret using the associated KMS key.
	var secretString string = *result.SecretString

	var awsSecret AwsSecret
	jsonErr := json.Unmarshal([]byte(secretString), &awsSecret)

	if jsonErr != nil {
		klog.Fatal(jsonErr)
	}
	return Secret(awsSecret.DSN)
}
