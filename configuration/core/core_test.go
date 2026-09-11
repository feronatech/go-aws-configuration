package core

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/feronatech/go-aws-configuration/configuration/core/error_handling"
)

type ConfigA struct {
	Environment string `env:"ENVIRONMENT" default:"local" validate:"oneof=local staging production"`
	LogLevel    string `env:"LOG_LEVEL" default:"info"`

	AWS struct {
		Region string `env:"AWS_REGION" required:"true"`
	}

	HTTP struct {
		Timeout time.Duration `env:"HTTP_TIMEOUT" default:"5s"`
	}

	Database struct {
		URL string `env:"DATABASE_URL" required:"true" secret:"true"`
	}
}

func Test_Load_Nested(t *testing.T) {
	region := "us-east-1"
	postgresURL := "postgres://localhost/app"
	httpTimeout := "30s"

	t.Setenv("AWS_REGION", region)
	t.Setenv("DATABASE_URL", postgresURL)
	t.Setenv("HTTP_TIMEOUT", httpTimeout)

	cfg, err := Load[ConfigA](WithEnvironment())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.AWS.Region != region {
		t.Errorf("expected AWS.Region to be '%s', got %s", region, cfg.AWS.Region)
	}
	if cfg.Database.URL != postgresURL {
		t.Errorf("expected Database.URL to be '%s', got %s", postgresURL, cfg.Database.URL)
	}
	if cfg.HTTP.Timeout.String() != httpTimeout {
		t.Errorf("expected HTTP.Timeout to be '%s', got %s", httpTimeout, cfg.HTTP.Timeout.String())
	}
}

func Test_Load_MissingRequired_ReportsAll(t *testing.T) {
	_, err := Load[ConfigA](WithEnvironment())
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	for _, fieldErr := range err.(error_handling.FieldErrors) {
		if !errors.Is(fieldErr.Err, error_handling.ErrRequired) {
			t.Errorf("expected error type to be ErrRequired, got %v", fieldErr.Err)
		}
	}
}

type ConfigB struct {
	Environment string `env:"ENVIRONMENT" default:"local" validate:"oneof=local staging production"`
	LogLevel    string `env:"LOG_LEVEL" default:"info"`

	HTTP struct {
		Timeout time.Duration `env:"HTTP_TIMEOUT" default:"5s"`
	}

	Database struct {
		URL string `env:"DATABASE_URL" secret:"true"`
	}
}

func Test_Load_WithDotEnv(t *testing.T) {
	environment := "local"
	logLevel := "debug"
	postgresURL := "postgres://localhost/app"
	httpTimeout := "30s"

	fakeDotEnv := `
ENVIRONMENT=` + environment + `
LOG_LEVEL=` + logLevel + `
HTTP_TIMEOUT=` + httpTimeout + `
`

	f, _ := os.CreateTemp("", "sample.env")
	defer os.Remove(f.Name())
	f.WriteString(fakeDotEnv)
	f.Close()

	cfg, err := Load[ConfigB](WithDotEnv(f.Name()))
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.Environment != environment {
		t.Errorf("expected Environment to be '%s', got %s", environment, cfg.Environment)
	}
	if cfg.LogLevel != logLevel {
		t.Errorf("expected LogLevel to be '%s', got %s", logLevel, cfg.LogLevel)
	}
	if cfg.Database.URL == postgresURL {
		t.Errorf("expected Database.URL to be unset, got %s", cfg.Database.URL)
	}
	if cfg.HTTP.Timeout.String() != httpTimeout {
		t.Errorf("expected HTTP.Timeout to be '%s', got %s", httpTimeout, cfg.HTTP.Timeout.String())
	}
}

func Test_Load_WithEnvironment(t *testing.T) {
	environment := "local"
	logLevel := "debug"
	postgresURL := "postgres://localhost/app"
	httpTimeout := "30s"

	t.Setenv("ENVIRONMENT", environment)
	t.Setenv("LOG_LEVEL", logLevel)
	t.Setenv("HTTP_TIMEOUT", httpTimeout)

	cfg, err := Load[ConfigB](WithEnvironment())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.Environment != environment {
		t.Errorf("expected Environment to be '%s', got %s", environment, cfg.Environment)
	}
	if cfg.LogLevel != logLevel {
		t.Errorf("expected LogLevel to be '%s', got %s", logLevel, cfg.LogLevel)
	}
	if cfg.Database.URL == postgresURL {
		t.Errorf("expected Database.URL to be unset, got %s", cfg.Database.URL)
	}
	if cfg.HTTP.Timeout.String() != httpTimeout {
		t.Errorf("expected HTTP.Timeout to be '%s', got %s", httpTimeout, cfg.HTTP.Timeout.String())
	}
}

func Test_Load_ValidateOneOf(t *testing.T) {
	t.Setenv("ENVIRONMENT", "invalid")

	_, err := Load[ConfigB](WithEnvironment())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	_, err = Load[ConfigB](WithEnvironment(), WithValidation())
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func Test_Load_OrderedLoading(t *testing.T) {
	fakeDotEnv := `
LOG_LEVEL=ERROR
`

	t.Setenv("DOTENV", fakeDotEnv)
	t.Setenv("LOG_LEVEL", "DEBUG")

	f, _ := os.CreateTemp("", "sample.env")
	defer os.Remove(f.Name())
	f.WriteString(fakeDotEnv)
	f.Close()

	cfg, err := Load[ConfigB](WithDotEnv(f.Name()), WithEnvironment())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.LogLevel != "ERROR" {
		t.Errorf("expected LogLevel to be 'ERROR', got %s from fallback", cfg.LogLevel)
	}
}
