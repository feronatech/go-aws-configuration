package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/feronatech/go-aws-configuration/configuration/core"
)

type fakeSSM struct {
	params map[string]string
	err    error
	calls  []ssm.GetParameterInput
}

func (f *fakeSSM) GetParameter(
	ctx context.Context,
	in *ssm.GetParameterInput,
	_ ...func(*ssm.Options),
) (*ssm.GetParameterOutput, error) {
	f.calls = append(f.calls, *in)
	if f.err != nil {
		return nil, f.err
	}
	v, ok := f.params[aws.ToString(in.Name)]
	if !ok {
		return nil, &types.ParameterNotFound{}
	}
	return &ssm.GetParameterOutput{
		Parameter: &types.Parameter{
			Name:  in.Name,
			Value: aws.String(v),
			Type:  types.ParameterTypeSecureString,
		},
	}, nil
}

type ssmConfig struct {
	Database struct {
		URL string `env:"/myapp/database_url"`
	}
}

func Test_Load_WithSSM(t *testing.T) {
	postgresURL := "postgres://localhost/app"

	fake := &fakeSSM{params: map[string]string{"/myapp/database_url": postgresURL}}

	cfg, err := core.Load[ssmConfig](withSSMClient(fake))

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if cfg.Database.URL != postgresURL {
		t.Errorf("expected Database.URL to be '%s', got %s", postgresURL, cfg.Database.URL)
	}
}

type awsConfig struct {
	Region string `env:"AWS_REGION"`
}

func Test_Load_WithAwsConfig(t *testing.T) {
	awsCfg := &aws.Config{
		Region: "us-east-1",
	}

	cfg, err := core.Load[awsConfig](
		WithAwsConfig(awsCfg),
	)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("expected Region to be 'us-east-1', got %s", cfg.Region)
	}
}
