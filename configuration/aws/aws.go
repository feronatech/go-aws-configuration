// aws package provides loading options for AWS services, including SSM Parameter Store and AWS configuration.
// It can be optionally imported additionally if you want to load configuration from AWS services.
// It's separated from the core functions on purpose to avoid unnecessary dependencies on AWS SDK for users who don't need it.
package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/feronatech/go-aws-configuration/configuration/core/loading"
)

// ParameterGetter is the subset of *ssm.Client this package uses.
type parameterGetter interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

type ssmSource struct {
	name    string
	client  parameterGetter
	decrypt bool
}

func (s ssmSource) Name() string { return s.name }
func (s ssmSource) Lookup(key string) (string, bool) {
	if !strings.HasPrefix(key, "/") {
		return "", false
	}
	out, err := s.client.GetParameter(context.Background(), &ssm.GetParameterInput{
		Name:           aws.String(key),
		WithDecryption: aws.Bool(s.decrypt),
	})
	if err != nil {
		return "", false
	}
	if out.Parameter == nil {
		return "", false
	}
	return aws.ToString(out.Parameter.Value), true
}

// WithSSM returns a loading.Option that adds AWS SSM Parameter Store as a source for configuration values.
// It requires an AWS configuration to create an SSM client.
func WithSSM(awsCfg *aws.Config) loading.Option {
	return withSSMClient(ssm.NewFromConfig(*awsCfg))
}

func withSSMClient(client parameterGetter) loading.Option {
	return func(l *loading.Loader) error {
		l.AddSource(ssmSource{name: "ssm", client: client, decrypt: true})
		return nil
	}
}

type awsSource struct {
	name string
	cfg  *aws.Config
}

func (s awsSource) Name() string { return s.name }
func (s awsSource) Lookup(key string) (string, bool) {
	if key != "AWS_REGION" {
		return "", false
	}
	if s.cfg == nil || s.cfg.Region == "" {
		return "", false
	}
	return s.cfg.Region, true
}

// WithAwsConfig returns a loading.Option that adds AWS configuration as a source for configuration values.
// It requires an AWS configuration to provide values for the configuration struct.
func WithAwsConfig(awsCfg *aws.Config) loading.Option {
	return func(l *loading.Loader) error {
		l.AddSource(awsSource{name: "aws", cfg: awsCfg})
		return nil
	}
}
