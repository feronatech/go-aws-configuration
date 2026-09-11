# Golang AWS Configuration Utility Library

[![Static Badge](https://img.shields.io/badge/aws-SDK_v2-orange)](go.mod) [![Static Badge](https://img.shields.io/badge/Go-1.25%2B-%2300ADD8?logo=go)](go.mod) [![Go Reference](https://img.shields.io/badge/Reference-%2300ADD8?logo=go&logoColor=white)](https://pkg.go.dev/github.com/feronatech/go-aws-configuration) [![GitHub License](https://img.shields.io/github/license/feronatech/go-aws-configuration?label=License&color=purple)](LICENSE)

A small, generic configuration loader for Go services. It loads values into a caller-defined struct from environment variables, `.env` files, and—through an optional AWS subpackage—AWS Systems Manager Parameter Store or an `aws.Config` region.

Configuration fields opt in to loading with struct tags. Values are converted into common Go scalar types, slices, pointers, `time.Duration`, and types implementing `encoding.TextUnmarshaler`; validation can be enabled with the standard `validate` tag and go-playground/validator.

The core package does not import the AWS SDK. AWS integrations live in `configuration/aws`, so applications that only need local configuration can use the core package without adding AWS loading options.

---

## Installation

```bash
go get github.com/feronatech/go-aws-configuration
```

## Import

```go
import "github.com/feronatech/go-aws-configuration/configuration"
```

Optional AWS helpers:

```go
import "github.com/feronatech/go-aws-configuration/configuration/aws"
```

## Quick start

```go
package main

import (
	"fmt"

	"github.com/feronatech/go-aws-configuration/configuration"
)

type ConfigObj struct {
	Port  int    `env:"PORT" default:"8080"`
	Stage string `env:"STAGE" default:"dev"`
}

func main() {
	cfg, err := configuration.Load[ConfigObj](configuration.WithEnvironment())
	if err != nil {
		panic(err)
	}

	fmt.Printf("port=%d stage=%s\n", cfg.Port, cfg.Stage)
}
```

With no `PORT` or `STAGE` environment variables, the output is:

```text
port=8080 stage=dev
```

---

## Configuration sources and options

`configuration.Load[T any](opts ...loading.Option) (*T, error)` creates a loader with the `env` tag name and `default` tag name, applies options in the order supplied, walks the exported fields of `T`, and optionally validates the result. `T` must be a struct type.

### `WithEnvironment()`

```go
func WithEnvironment() loading.Option
```

Adds the `environment` source. The `env` struct tag supplies the environment-variable name. The source uses `os.LookupEnv`, so an environment variable present with an empty value is still found and parsed as an empty value.

### `WithDotEnv(paths ...string)`

```go
func WithDotEnv(paths ...string) loading.Option
```

Adds a `dotenv` source containing values parsed from each supplied file. Files are read in argument order; later files overwrite duplicate keys within the combined `.env` map. A missing file is ignored. Other read or parse errors stop option processing and are returned as `config: reading <path>: <wrapped error>`.

The parser trims each line, skips blank lines and lines beginning with `#`, splits on the first `=`, trims the key and value, and does not perform shell quoting or variable expansion. A nonblank line without `=` returns `invalid line: "<line>"`.

### `WithValidation()`

```go
func WithValidation() loading.Option
```

Enables validation after source loading, using the `validate` struct tags and go-playground/validator/v10. Validation errors are returned as field errors; loading/conversion errors are reported before validation.

### Source precedence

Sources are searched in the order they were added. The first source that reports a key as present wins. Therefore, option order controls precedence. A field without an `env` tag is skipped, and a missing value uses its `default` tag when present. A missing field tagged `required:"true"` produces a required-field error.

---

## Fields and parsing

Only exported fields are walked. Nested structs are traversed recursively unless the struct type is `time.Duration` or implements `encoding.TextUnmarshaler`. The loader uses these tags:

| Tag | Meaning |
|---|---|
| `env` | Source key to look up; fields without it are ignored. |
| `default` | Fallback raw value when no source contains the `env` key. |
| `required` | When exactly `"true"`, a missing value is an error. |
| `secret` | When exactly `"true"`, marks a conversion `FieldError` as secret. |
| `validate` | Validation rule(s) used when `WithValidation()` is enabled. |

`SetValue` supports strings, booleans accepted by `strconv.ParseBool`, signed and unsigned integers in base 10, floats, slices, pointers, `time.Duration`, and addressable values implementing `encoding.TextUnmarshaler`. Slices are comma-separated; each element is trimmed. An empty raw slice value becomes an empty slice. Unsupported kinds return `unsupported type <type>`.

Conversion errors include `invalid bool "<raw>"`, `invalid integer "<raw>"`, `invalid unsigned integer "<raw>"`, `invalid float "<raw>"`, and `invalid duration "<raw>": <wrapped parse error>`. Slice element errors are wrapped as `element <index>: <error>`.

---

## Validation

Validation is opt-in. `WithValidation()` calls validator on the populated struct pointer. For validator validation failures, each error is converted to a `FieldError` whose path is the validator namespace and whose error is `failed "<tag>" validation`. Non-validation errors from validator are returned unchanged.

`Load` returns `config: Load requires a struct type, got <kind>` when instantiated with a non-struct type. Option errors are returned immediately. Field errors are accumulated while walking and returned together before validation runs.

### Errors at a glance

| Error | Behavior/message |
|---|---|
| Required field | `FieldError.Err` is `ErrRequired` when `required:"true"` is missing. |
| Conversion | `FieldError.Err` contains one of the conversion messages documented above. |
| Validation | `FieldError.Err` is `failed "<tag>" validation`. |
| Non-struct config | `config: Load requires a struct type, got <kind>`. |
| `.env` read failure | `config: reading <path>: <wrapped error>`; missing files are ignored. |
| `.env` syntax failure | `invalid line: "<line>"`. |

`FieldError`, `FieldErrors`, and `ErrRequired` are defined in the internal `configuration/core/error_handling` package used by these files; they are returned or referenced by the loader but are not declarations in the five source files documented here.

---

## Core loading package

The implementation package is `github.com/feronatech/go-aws-configuration/configuration/core` and its loading helpers are `github.com/feronatech/go-aws-configuration/configuration/core/loading`.

```go
type Source interface {
	Lookup(key string) (string, bool)
	Name() string
}

type Option func(*Loader) error

func NewLoader(tagEnv string, tagDefault string) *Loader
func (l *Loader) SetValidate(validate bool)
func (l *Loader) ShouldValidate() bool
func (l *Loader) AddSource(s Source)
func (l *Loader) Walk(v reflect.Value, path []string, errs *error_handling.FieldErrors)
```

The `configuration` package re-exports the option constructors as package variables with the same callable signatures: `WithDotEnv`, `WithEnvironment`, and `WithValidation`.

---

## AWS subpackage

The optional package is imported as:

```go
import awscfg "github.com/feronatech/go-aws-configuration/configuration/aws"
```

### `WithSSM`

```go
func WithSSM(awsCfg *aws.Config) loading.Option
```

Creates an SSM client with `ssm.NewFromConfig(*awsCfg)` and adds an `ssm` source. Only keys beginning with `/` are queried. Each lookup calls `GetParameter` with `context.Background()`, the key as `Name`, and `WithDecryption` set to `true`. AWS lookup errors, nil output, nil parameters, and nil parameter values are treated as “not found”; they are not returned as loader errors. A nil `awsCfg` is dereferenced while creating the client and will panic rather than return an error.

### `WithAwsConfig`

```go
func WithAwsConfig(awsCfg *aws.Config) loading.Option
```

Adds an `aws` source. It supplies `awsCfg.Region` only for the exact key `AWS_REGION`; a nil config or empty region is treated as not found. It does not provide credentials, endpoint, or other AWS configuration fields.

Example:

```go
import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/feronatech/go-aws-configuration/configuration"
	awscfg "github.com/feronatech/go-aws-configuration/configuration/aws"
)

cfg, err := awsconfig.LoadDefaultConfig(context.Background())
if err != nil {
	return err
}

appCfg, err := configuration.Load[ConfigObj](
	configuration.WithEnvironment(),
	awscfg.WithAwsConfig(&cfg),
	awscfg.WithSSM(&cfg),
)
_ = appCfg
_ = err
```

---

## License

See [LICENSE](LICENSE) for details.
