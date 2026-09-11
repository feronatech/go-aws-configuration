// configuration package provides a simple way to load configuration from various sources,
// including environment variables, AWS SSM Parameter Store, and .env files.
// It supports validation of the loaded configuration and allows for custom loading options.
package configuration

import (
	"github.com/feronatech/go-aws-configuration/configuration/core"
	"github.com/feronatech/go-aws-configuration/configuration/core/loading"
)

// Load loads the configuration of type T from the specified sources and options.
// It returns a pointer to the loaded configuration and an error if any occurred during loading or validation.
func Load[T any](opts ...loading.Option) (*T, error) {
	return core.Load[T](opts...)
}

// WithDotEnv returns a loading.Option that adds a .env file as a source for configuration values.
// It can be parameterized with one or more file paths to .env files.
// If a specified file does not exist, it will be ignored.
var WithDotEnv = core.WithDotEnv

// WithEnvironment returns a loading.Option that adds environment variables as a source for configuration values.
// It uses the "env" struct tag to determine which environment variable corresponds to each field in the configuration struct.
var WithEnvironment = core.WithEnvironment

// WithValidation returns a loading.Option that enables validation of the loaded configuration.
// It uses the "validate" struct tag to determine the validation rules for each field in the configuration struct.
var WithValidation = core.WithValidation
