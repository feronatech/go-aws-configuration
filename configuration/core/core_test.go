package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Config struct {
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
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("DATABASE_URL", "postgres://localhost/app")
	t.Setenv("HTTP_TIMEOUT", "30s")

	cfg, err := Load[Config](WithEnvironment())
	require.NoError(t, err)

	assert.Equal(t, "us-east-1", cfg.AWS.Region)
	assert.Equal(t, 30*time.Second, cfg.HTTP.Timeout)
	assert.Equal(t, "local", cfg.Environment) // default applied
}

func Test_Load_MissingRequired_ReportsAll(t *testing.T) {
	_, err := Load[Config](WithEnvironment())
	require.Error(t, err)

	//assert.ErrorIs(t, err, ErrRequired)
	assert.Contains(t, err.Error(), "AWS_REGION")
	assert.Contains(t, err.Error(), "DATABASE_URL") // both, not just the first
}
