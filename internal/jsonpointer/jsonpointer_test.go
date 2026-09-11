package jsonpointer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscape(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "should_return_plain_token_unchanged", in: "prod", want: "prod"},
		{name: "should_escape_slash", in: "a/b", want: "a~1b"},
		{name: "should_escape_tilde", in: "a~b", want: "a~0b"},
		{name: "should_escape_slash_and_tilde", in: "a/b~c", want: "a~1b~0c"},
		{name: "should_return_empty_string", in: "", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Escape(tc.in))
		})
	}
}

func TestUnescape(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "should_return_plain_token_unchanged", in: "prod", want: "prod"},
		{name: "should_unescape_tilde_one", in: "a~1b", want: "a/b"},
		{name: "should_unescape_tilde_zero", in: "a~0b", want: "a~b"},
		{name: "should_unescape_slash_and_tilde", in: "a~1b~0c", want: "a/b~c"},
		{name: "should_return_empty_string", in: "", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Unescape(tc.in))
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "should_round_trip_plain_name", in: "prod"},
		{name: "should_round_trip_name_with_slash", in: "dev/prod"},
		{name: "should_round_trip_name_with_tilde", in: "dev~prod"},
		{name: "should_round_trip_name_with_tilde_one", in: "dev~1prod"},
		{name: "should_round_trip_name_with_tilde_zero", in: "dev~0prod"},
		{name: "should_round_trip_empty_name", in: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.in, Unescape(Escape(tc.in)))
		})
	}
}
