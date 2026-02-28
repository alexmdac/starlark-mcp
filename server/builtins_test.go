package server

import (
	"context"
	"strings"
	"testing"
)

func TestBuiltins(t *testing.T) {
	testCases := []struct {
		name           string
		code           string
		expectedResult string
		expectedErr    string
	}{
		{
			name:           "load_math_module",
			code:           `load("math", "sin", "pi"); print(sin(pi / 2))`,
			expectedResult: "1.0",
		},
		{
			name:        "load_unknown_module",
			code:        `load("foo", "bar")`,
			expectedErr: "no such module: \"foo\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ExecuteStarlark(context.Background(), tc.code)

			if tc.expectedErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, but got none", tc.expectedErr)
				}
				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Fatalf("expected error to contain %q, but got %q", tc.expectedErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if strings.TrimSpace(result) != tc.expectedResult {
				t.Fatalf("expected result %q, but got %q", tc.expectedResult, result)
			}
		})
	}
}
