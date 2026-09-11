package core

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/feronatech/go-aws-configuration/configuration/core/error_handling"
	"github.com/feronatech/go-aws-configuration/configuration/core/loading"
	"github.com/feronatech/go-aws-configuration/configuration/core/loading/utils"
)

func Load[T any](opts ...loading.Option) (*T, error) {
	l := loading.NewLoader("env", "default")
	for _, opt := range opts {
		if err := opt(l); err != nil {
			return nil, err
		}
	}

	var cfg T
	v := reflect.ValueOf(&cfg).Elem()
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("config: Load requires a struct type, got %s", v.Kind())
	}

	var errs error_handling.FieldErrors
	l.Walk(v, nil, &errs)
	if len(errs) > 0 {
		return nil, errs
	}

	if l.ShouldValidate() {
		if err := utils.RunValidation(v, nil); err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

type mapSource struct {
	name   string
	values map[string]string
}

func (m mapSource) Name() string                     { return m.name }
func (m mapSource) Lookup(key string) (string, bool) { val, ok := m.values[key]; return val, ok }

func WithDotEnv(paths ...string) loading.Option {
	return func(l *loading.Loader) error {
		m := map[string]string{}
		for _, p := range paths {
			vals, err := utils.ParseDotEnv(p)
			if errors.Is(err, os.ErrNotExist) {
				continue // absent .env is normal in deployed environments
			}
			if err != nil {
				return fmt.Errorf("config: reading %s: %w", p, err)
			}
			for k, v := range vals {
				m[k] = v
			}
		}
		l.AddSource(mapSource{name: "dotenv", values: m})
		return nil
	}
}

type envSource struct{}

func (envSource) Name() string                     { return "environment" }
func (envSource) Lookup(key string) (string, bool) { return os.LookupEnv(key) }

func WithEnvironment() loading.Option {
	return func(l *loading.Loader) error {
		l.AddSource(envSource{})
		return nil
	}
}

func WithValidation() loading.Option {
	return func(l *loading.Loader) error {
		l.SetValidate(true)
		return nil
	}
}
