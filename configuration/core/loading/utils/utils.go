package utils

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

	"github.com/feronatech/go-aws-configuration/configuration/core/error_handling"
	"github.com/go-playground/validator/v10"
)

var (
	durationType    = reflect.TypeOf(time.Duration(0))
	textUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

func IsParseableStruct(t reflect.Type) bool {
	if t == durationType {
		return true
	}

	if t.Implements(textUnmarshaler) {
		return true
	}

	return false
}

func SetValue(v reflect.Value, raw string) error {
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
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return SetValue(v.Elem(), raw)
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
		if err := SetValue(out.Index(i), strings.TrimSpace(p)); err != nil {
			return fmt.Errorf("element %d: %w", i, err)
		}
	}
	v.Set(out)
	return nil
}

func RunValidation(v reflect.Value, _ []string) error {
	validate := validator.New()
	if err := validate.Struct(v.Addr().Interface()); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			errs := make(error_handling.FieldErrors, 0, len(ve))
			for _, fe := range ve {
				errs = append(errs, error_handling.FieldError{
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

func ParseDotEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer (func() { _ = f.Close() })()

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
