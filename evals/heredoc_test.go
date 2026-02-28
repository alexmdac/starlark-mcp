//go:build eval

package main

import (
	"testing"
)

func TestHereDoc(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple",
			in: `
				hello
				world
			`,
			want: "hello\nworld",
		},
		{
			name: "mixed indent",
			in: `
				line1
					indented
				line3
			`,
			want: "line1\n\tindented\nline3",
		},
		{
			name: "empty lines preserved",
			in: `
				a

				b
			`,
			want: "a\n\nb",
		},
		{
			name: "single line",
			in: `
				just one
			`,
			want: "just one",
		},
		{
			name: "empty",
			in: `
			`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hereDoc(tt.in)
			if got != tt.want {
				t.Errorf("hereDoc() = %q, want %q", got, tt.want)
			}
		})
	}
}
