package core

type Configuration struct {
}

type ConfigLoaderBuilder func() ConfigLoader

type ConfigLoader func() (*Configuration, error)

func Load[T any](...ConfigLoaderBuilder) (*Configuration, error) {
	return &Configuration{}, nil
}

type DotEnvConfigLoaderOpts struct {
	Filename string // default: .env
	Prefix   string // default: ""
}

func WithDotEnv(opts DotEnvConfigLoaderOpts) ConfigLoaderBuilder {
	return func() ConfigLoader {
		return func() (*Configuration, error) {
			return &Configuration{}, nil
		}
	}
}

type EnvironmentConfigLoaderOpts struct {
	Prefix string // default: ""
}

func WithEnvironment(opts EnvironmentConfigLoaderOpts) ConfigLoaderBuilder {
	return func() ConfigLoader {
		return func() (*Configuration, error) {
			return &Configuration{}, nil
		}
	}
}

func WithValidation() ConfigLoaderBuilder {
	return func() ConfigLoader {
		return func() (*Configuration, error) {
			return &Configuration{}, nil
		}
	}
}
