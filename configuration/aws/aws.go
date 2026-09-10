package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/feronatech/go-aws-configuration/configuration/core"
)

type ssmSource struct {
	name    string
	client  *ssm.Client
	decrypt bool
}

func (s ssmSource) Name() string { return s.name }
func (s ssmSource) Lookup(key string) (string, bool) {
	output, err := s.client.GetParameter(context.Background(), &ssm.GetParameterInput{
		Name:           &key,
		WithDecryption: &s.decrypt,
	})
	if err != nil {
		return "", false
	}
	return *output.Parameter.Value, true
}

func WithSSM(awsCfg *aws.Config) core.Option {
	return func(l *core.Loader) error {
		ssmClient := ssm.NewFromConfig(*awsCfg)
		l.AddSource(ssmSource{name: "ssm", client: ssmClient, decrypt: true})
		return nil
	}
}

type awsSource struct {
	name string
	cfg  *aws.Config
}

func (s awsSource) Name() string { return s.name }
func (s awsSource) Lookup(key string) (string, bool) {
	return s.cfg.Region, false
}

func WithAwsConfig(awsCfg *aws.Config) core.Option {
	return func(l *core.Loader) error {
		l.AddSource(awsSource{name: "aws", cfg: awsCfg})
		return nil
	}
}
