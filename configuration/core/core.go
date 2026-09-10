package core

import (
	"bufio"
	"encoding"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	durationType    = reflect.TypeOf(time.Duration(0))
	textUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

type fieldError struct {
	Path   string
	EnvKey string
	Err    error
	Secret bool
}

type fieldErrors []fieldError

func (fe fieldErrors) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "config: %d problem(s):", len(fe))
	for _, e := range fe {
		fmt.Fprintf(&b, "\n  - %s (%s): %v", e.Path, e.EnvKey, e.Err)
	}
	return b.String()
}

type Source interface {
	Lookup(key string) (string, bool)
	Name() string
}

type Loader struct {
	sources    []Source
	validate   bool
	tagEnv     string
	tagDefault string
}

func (l *Loader) AddSource(s Source) {
	l.sources = append(l.sources, s)
}

type Option func(*Loader) error

func isParseableStruct(t reflect.Type) bool {
	if t == durationType {
		return true
	}

	if t.Implements(textUnmarshaler) {
		return true
	}

	return false
}

func (l *Loader) walk(v reflect.Value, path []string, errs *fieldErrors) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fv := v.Field(i)

		if !sf.IsExported() {
			continue
		}

		fieldPath := append(path, sf.Name)

		// Recurse into nested structs, but not types we know how to parse
		// directly (time.Duration is int64, but time.Time is a struct).
		if fv.Kind() == reflect.Struct && !isParseableStruct(sf.Type) {
			l.walk(fv, fieldPath, errs)
			continue
		}

		key, hasKey := sf.Tag.Lookup(l.tagEnv)
		if !hasKey {
			continue
		}

		raw, found := l.lookup(key)
		if !found {
			if def, hasDef := sf.Tag.Lookup(l.tagDefault); hasDef {
				raw, found = def, true
			}
		}

		if !found {
			if sf.Tag.Get("required") == "true" {
				*errs = append(*errs, fieldError{
					Path:   strings.Join(fieldPath, "."),
					EnvKey: key,
					Err:    errors.New("required but not set"),
				})
			}
			continue
		}

		if err := setValue(fv, raw); err != nil {
			*errs = append(*errs, fieldError{
				Path:   strings.Join(fieldPath, "."),
				EnvKey: key,
				Err:    err,
				Secret: sf.Tag.Get("secret") == "true",
			})
		}
	}
}

func setValue(v reflect.Value, raw string) error {
	// Let types opt in to their own parsing.
	if v.CanAddr() {
		if u, ok := v.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return u.UnmarshalText([]byte(raw))
		}
	}

	if v.Type() == durationType {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", raw, err)
		}
		v.SetInt(int64(d))
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("invalid bool %q", raw)
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid integer %q", raw)
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid unsigned integer %q", raw)
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid float %q", raw)
		}
		v.SetFloat(f)
	case reflect.Slice:
		return setSlice(v, raw)
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return setValue(v.Elem(), raw)
	default:
		return fmt.Errorf("unsupported type %s", v.Type())
	}
	return nil
}

func setSlice(v reflect.Value, raw string) error {
	if raw == "" {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		return nil
	}
	parts := strings.Split(raw, ",")
	out := reflect.MakeSlice(v.Type(), len(parts), len(parts))
	for i, p := range parts {
		if err := setValue(out.Index(i), strings.TrimSpace(p)); err != nil {
			return fmt.Errorf("element %d: %w", i, err)
		}
	}
	v.Set(out)
	return nil
}

func (l *Loader) lookup(key string) (string, bool) {
	for _, s := range l.sources {
		if val, ok := s.Lookup(key); ok {
			return val, true
		}
	}
	return "", false
}

func runValidation(v reflect.Value, _ []string) error {
	validate := validator.New()
	if err := validate.Struct(v.Addr().Interface()); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			errs := make(fieldErrors, 0, len(ve))
			for _, fe := range ve {
				errs = append(errs, fieldError{
					Path: fe.Namespace(),
					Err:  fmt.Errorf("failed %q validation", fe.Tag()),
				})
			}
			return errs
		}
		return err
	}
	return nil
}

func parseDotEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m := map[string]string{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line: %q", line)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		m[key] = val
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return m, nil
}

func Load[T any](opts ...Option) (*T, error) {
	l := &Loader{tagEnv: "env", tagDefault: "default"}
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

	var errs fieldErrors
	l.walk(v, nil, &errs)
	if len(errs) > 0 {
		return nil, errs
	}

	if l.validate {
		if err := runValidation(v, nil); err != nil {
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

func WithDotEnv(paths ...string) Option {
	return func(l *Loader) error {
		m := map[string]string{}
		for _, p := range paths {
			vals, err := parseDotEnv(p)
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

func WithEnvironment() Option {
	return func(l *Loader) error {
		l.AddSource(envSource{})
		return nil
	}
}

func WithValidation() Option {
	return func(l *Loader) error {
		l.validate = true
		return nil
	}
}
