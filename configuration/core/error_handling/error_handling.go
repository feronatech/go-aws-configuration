package error_handling

import (
	"errors"
	"fmt"
	"strings"
)

var ErrRequired = errors.New("required but not set")

type FieldError struct {
	Path   string
	EnvKey string
	Err    error
	Secret bool
}

type FieldErrors []FieldError

func (fe FieldErrors) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "config: %d problem(s):", len(fe))
	for _, e := range fe {
		fmt.Fprintf(&b, "\n  - %s (%s): %v", e.Path, e.EnvKey, e.Err)
	}
	return b.String()
}
