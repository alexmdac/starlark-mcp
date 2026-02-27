package main

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
		// pow
		{
			name:           "pow_basic",
			code:           `load("math", "pow"); print(pow(2.0, 3.0))`,
			expectedResult: "8.0",
		},
		{
			name:           "pow_zero_exponent",
			code:           `load("math", "pow"); print(pow(5.0, 0.0))`,
			expectedResult: "1.0",
		},
		{
			name:           "pow_nan",
			code:           `load("math", "pow"); print(pow(-1.0, 0.5))`,
			expectedResult: "nan",
		},

		// sqrt
		{
			name:           "sqrt_perfect_square",
			code:           `load("math", "sqrt"); print(sqrt(16.0))`,
			expectedResult: "4.0",
		},
		{
			name:           "sqrt_zero",
			code:           `load("math", "sqrt"); print(sqrt(0.0))`,
			expectedResult: "0.0",
		},
		{
			name:           "sqrt_negative",
			code:           `load("math", "sqrt"); print(sqrt(-1.0))`,
			expectedResult: "nan",
		},

		{
			name:        "pow_non_float",
			code:        `load("math", "pow"); print(pow("a", 1.0))`,
			expectedErr: "pow: for parameter 1: got string, want float or int",
		},
		{
			name:        "sqrt_non_float",
			code:        `load("math", "sqrt"); print(sqrt(True))`,
			expectedErr: "sqrt: for parameter 1: got bool, want float or int",
		},
		{
			name:           "pow_inf",
			code:           `load("math", "pow"); print(pow(10.0, 1000.0))`,
			expectedResult: "+inf",
		},
		{
			name:        "load_unknown_module",
			code:        `load("foo", "bar")`,
			expectedErr: "no such module: \"foo\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := executeStarlark(context.Background(), tc.code)

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
