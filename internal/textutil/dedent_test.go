package textutil

import (
	"testing"
)

func TestDedent(t *testing.T) {
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
		{
			name: "multiple leading newlines",
			in:   "\n\n\n\t\t\t\thello\n\t\t\t\tworld\n\t\t\t",
			want: "hello\nworld",
		},
		{
			name: "multiple trailing newlines",
			in:   "\n\t\t\t\thello\n\t\t\t\tworld\n\t\t\t\n\n\n",
			want: "hello\nworld",
		},
		{
			name: "trailing spaces on closing line",
			in:   "\n\t\t\t\thello\n\t\t\t\tworld\n\t\t\t  \t",
			want: "hello\nworld",
		},
		{
			name: "only whitespace",
			in:   "\n\t\t\t\n  \n\t\n",
			want: "",
		},
		{
			name: "no leading newline",
			in:   "\t\t\thello\n\t\t\tworld\n\t\t",
			want: "hello\nworld",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Dedent(tt.in)
			if got != tt.want {
				t.Errorf("Dedent() = %q, want %q", got, tt.want)
			}
		})
	}
}
