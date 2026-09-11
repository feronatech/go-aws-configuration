package utils

import (
	"reflect"
	"testing"
	"time"
)

type customTextUnmarshaler struct{}

func (c customTextUnmarshaler) UnmarshalText(text []byte) error {
	return nil
}

type customNonTextUnmarshaler struct{}

func Test_IsParseableStruct(t *testing.T) {
	testCases := []struct {
		name  string
		input reflect.Type
		want  bool
	}{
		{
			name:  "time.Time is not parseable",
			input: reflect.TypeOf(time.Time{}),
			want:  false,
		},
		{
			name:  "time.Duration is parseable",
			input: reflect.TypeOf(time.Duration(0)),
			want:  true,
		},
		{
			name:  "custom type implementing TextUnmarshaler is parseable",
			input: reflect.TypeOf(customTextUnmarshaler{}),
			want:  true,
		},
		{
			name:  "custom type not implementing TextUnmarshaler is not parseable",
			input: reflect.TypeOf(customNonTextUnmarshaler{}),
			want:  false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsParseableStruct(tc.input)
			if got != tc.want {
				t.Errorf("IsParseableStruct() = %v, want %v", got, tc.want)
			}
		})
	}
}
