package loading

import (
	"reflect"
	"strings"

	"github.com/feronatech/go-aws-configuration/configuration/core/error_handling"
	"github.com/feronatech/go-aws-configuration/configuration/core/loading/utils"
)

type Source interface {
	Lookup(key string) (string, bool)
	Name() string
}

type Option func(*Loader) error

type Loader struct {
	sources    []Source
	validate   bool
	tagEnv     string
	tagDefault string
}

func NewLoader(tagEnv string, tagDefault string) *Loader {
	return &Loader{tagEnv: tagEnv, tagDefault: tagDefault}
}

func (l *Loader) SetValidate(validate bool) {
	l.validate = validate
}

func (l *Loader) ShouldValidate() bool {
	return l.validate
}

func (l *Loader) AddSource(s Source) {
	l.sources = append(l.sources, s)
}

func (l *Loader) Walk(v reflect.Value, path []string, errs *error_handling.FieldErrors) {
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
		if fv.Kind() == reflect.Struct && !utils.IsParseableStruct(sf.Type) {
			l.Walk(fv, fieldPath, errs)
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
				*errs = append(*errs, error_handling.FieldError{
					Path:   strings.Join(fieldPath, "."),
					EnvKey: key,
					Err:    error_handling.ErrRequired,
				})
			}
			continue
		}

		if err := utils.SetValue(fv, raw); err != nil {
			*errs = append(*errs, error_handling.FieldError{
				Path:   strings.Join(fieldPath, "."),
				EnvKey: key,
				Err:    err,
				Secret: sf.Tag.Get("secret") == "true",
			})
		}
	}
}

func (l *Loader) lookup(key string) (string, bool) {
	for _, s := range l.sources {
		if val, ok := s.Lookup(key); ok {
			return val, true
		}
	}
	return "", false
}
