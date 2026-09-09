package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/feronatech/go-aws-configuration/configuration/core"
)

type SsmConfigLoaderOpts struct {
	Store string // default: ""
}

func WithSSM(opts SsmConfigLoaderOpts) core.ConfigLoaderBuilder {
	return func() core.ConfigLoader {
		return func() (*core.Configuration, error) {
			return &core.Configuration{}, nil
		}
	}
}

func WithAwsConfig(awsCfg *aws.Config) core.ConfigLoaderBuilder {
	return func() core.ConfigLoader {
		return func() (*core.Configuration, error) {
			return &core.Configuration{}, nil
		}
	}
}
