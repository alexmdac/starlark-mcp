//go:build eval

package main

import (
	"math"
	"strconv"
	"strings"
)

// evalCase describes a single eval case: a prompt for the LLM and a judge function.
type evalCase struct {
	name   string
	tier   int
	prompt string
	judge  func(output string) bool
}

// exactOutput trims trailing whitespace from both expected and actual, then compares.
func exactOutput(expected string) func(string) bool {
	return func(output string) bool {
		return strings.TrimRight(output, " \t\n\r") == strings.TrimRight(expected, " \t\n\r")
	}
}

// oneOf accepts any of the given expected values (after trimming whitespace).
func oneOf(accepted ...string) func(string) bool {
	return func(output string) bool {
		trimmed := strings.TrimRight(output, " \t\n\r")
		for _, a := range accepted {
			if trimmed == a {
				return true
			}
		}
		return false
	}
}

// numericOutput parses the output as a float and checks if it is within tolerance of expected.
func numericOutput(expected float64, tolerance float64) func(string) bool {
	return func(output string) bool {
		trimmed := strings.TrimSpace(output)
		v, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return false
		}
		return math.Abs(v-expected) <= tolerance
	}
}

// validTopologicalSort checks that the output is a valid topological ordering for the given edges.
func validTopologicalSort(edges [][2]string) func(string) bool {
	return func(output string) bool {
		trimmed := strings.TrimSpace(output)
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			return false
		}

		vertexSet := make(map[string]bool)
		for _, e := range edges {
			vertexSet[e[0]] = true
			vertexSet[e[1]] = true
		}

		outputSet := make(map[string]bool)
		for _, f := range fields {
			outputSet[f] = true
		}
		if len(outputSet) != len(vertexSet) || len(fields) != len(vertexSet) {
			return false
		}
		for v := range vertexSet {
			if !outputSet[v] {
				return false
			}
		}

		pos := make(map[string]int)
		for i, f := range fields {
			pos[f] = i
		}

		for _, e := range edges {
			if pos[e[0]] >= pos[e[1]] {
				return false
			}
		}
		return true
	}
}

// validNQueens checks that the output is a valid N-queens solution.
func validNQueens(n int) func(string) bool {
	return func(output string) bool {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != n {
			return false
		}
		queens := 0
		cols := make(map[int]bool)
		diag1 := make(map[int]bool) // row - col
		diag2 := make(map[int]bool) // row + col
		for r, line := range lines {
			fields := strings.Fields(line)
			if len(fields) != n {
				return false
			}
			for c, cell := range fields {
				if cell == "Q" {
					queens++
					if cols[c] || diag1[r-c] || diag2[r+c] {
						return false
					}
					cols[c] = true
					diag1[r-c] = true
					diag2[r+c] = true
				} else if cell != "." {
					return false
				}
			}
		}
		return queens == n
	}
}

// cases is the full set of eval cases.
var cases = []evalCase{
	// ── Tier 1: Basics ──
	{
		name: "print_numbers_1_to_20",
		tier: 1,
		prompt: dedent(`
			Print the integers 1 to 20, one per line. Each line should contain just
			the number, nothing else.
		`),
		judge: exactOutput(dedent(`
			1
			2
			3
			4
			5
			6
			7
			8
			9
			10
			11
			12
			13
			14
			15
			16
			17
			18
			19
			20
		`)),
	},
	{
		name: "reverse_string",
		tier: 1,
		prompt: dedent(`
			Reverse the string "Hello, World!" and print the result. Print only the
			reversed string, nothing else.
		`),
		judge: exactOutput("!dlroW ,olleH"),
	},
	{
		name:   "sin_pi_over_6",
		tier:   1,
		prompt: `Compute sin(π/6) and print the numeric result. Print only the number, nothing else.`,
		judge:  numericOutput(0.5, 0.001),
	},

	// ── Tier 2: Simple Algorithms ──
	{
		name: "fizzbuzz",
		tier: 2,
		prompt: dedent(`
			Print FizzBuzz for numbers 1 through 30, one entry per line. For multiples
			of 3 print "Fizz", for multiples of 5 print "Buzz", for multiples of both
			print "FizzBuzz", otherwise print the number. Print only the output,
			nothing else.
		`),
		judge: exactOutput(dedent(`
			1
			2
			Fizz
			4
			Buzz
			Fizz
			7
			8
			Fizz
			Buzz
			11
			Fizz
			13
			14
			FizzBuzz
			16
			17
			Fizz
			19
			Buzz
			Fizz
			22
			23
			Fizz
			Buzz
			26
			Fizz
			28
			29
			FizzBuzz
		`)),
	},
	{
		name: "is_prime_104729",
		tier: 2,
		prompt: dedent(`
			Determine whether 104729 is a prime number. Print "true" if it is prime,
			or "false" if it is not. Print only that single word, nothing else.
		`),
		judge: exactOutput("true"),
	},
	{
		name: "gcd_48_18",
		tier: 2,
		prompt: dedent(`
			Compute the greatest common divisor (GCD) of 48 and 18. Print only the
			number, nothing else.
		`),
		judge: exactOutput("6"),
	},
	{
		name: "count_vowels",
		tier: 2,
		prompt: dedent(`
			Count the number of vowels (a, e, i, o, u, case-insensitive) in the string
			"The quick brown fox jumps over the lazy dog". Print only the count,
			nothing else.
		`),
		judge: exactOutput("11"),
	},
	{
		name: "decimal_to_binary",
		tier: 2,
		prompt: dedent(`
			Convert the decimal number 255 to its binary string representation with
			no prefix (no "0b"). Print only the binary string, nothing else.
		`),
		judge: exactOutput("11111111"),
	},
	{
		name: "pascals_triangle",
		tier: 2,
		prompt: dedent(`
			Print the first 10 rows of Pascal's triangle (rows 0 through 9). Print
			one row per line, with numbers separated by single spaces. Row 0 is "1",
			row 1 is "1 1", etc. Print only the triangle, nothing else.
		`),
		judge: exactOutput(dedent(`
			1
			1 1
			1 2 1
			1 3 3 1
			1 4 6 4 1
			1 5 10 10 5 1
			1 6 15 20 15 6 1
			1 7 21 35 35 21 7 1
			1 8 28 56 70 56 28 8 1
			1 9 36 84 126 126 84 36 9 1
		`)),
	},

	// ── Tier 3: Intermediate ──
	{
		name: "sieve_of_eratosthenes",
		tier: 3,
		prompt: dedent(`
			Use the Sieve of Eratosthenes to find all prime numbers below 10000.
			Print three lines: first line is the count of primes found, second line
			is the first 10 primes separated by spaces, third line is the last 10
			primes separated by spaces. Print only these three lines, nothing else.
		`),
		judge: exactOutput(dedent(`
			1229
			2 3 5 7 11 13 17 19 23 29
			9887 9901 9907 9923 9929 9931 9941 9949 9967 9973
		`)),
	},
	{
		name: "fibonacci_30",
		tier: 3,
		prompt: dedent(`
			Print the first 30 Fibonacci numbers F(0) through F(29), one per line.
			F(0)=0, F(1)=1, F(n)=F(n-1)+F(n-2). Print only the numbers, one per
			line, nothing else.
		`),
		judge: exactOutput(dedent(`
			0
			1
			1
			2
			3
			5
			8
			13
			21
			34
			55
			89
			144
			233
			377
			610
			987
			1597
			2584
			4181
			6765
			10946
			17711
			28657
			46368
			75025
			121393
			196418
			317811
			514229
		`)),
	},
	{
		name: "balanced_parentheses",
		tier: 3,
		prompt: dedent(`
			Check whether each of the following strings has balanced parentheses.
			For each string, print "true" if balanced or "false" if not, one result
			per line in order. The strings are: "(()())", "(()", "()()", ")(", "",
			"((()))", "(()))". Print only "true" or "false" on each line, nothing else.
		`),
		judge: exactOutput(dedent(`
			true
			false
			true
			false
			true
			true
			false
		`)),
	},
	{
		name: "longest_common_subsequence",
		tier: 3,
		prompt: dedent(`
			Find the length of the longest common subsequence of "ABCBDAB" and
			"BDCAB". Print only the number, nothing else.
		`),
		judge: exactOutput("4"),
	},
	{
		name: "roman_numerals",
		tier: 3,
		prompt: dedent(`
			Convert each of the following integers to Roman numerals and print each
			on its own line: 1, 4, 9, 14, 42, 99, 1994, 3999. Print only the Roman
			numeral strings, one per line, nothing else.
		`),
		judge: exactOutput(dedent(`
			I
			IV
			IX
			XIV
			XLII
			XCIX
			MCMXCIV
			MMMCMXCIX
		`)),
	},
	{
		name: "run_length_encoding",
		tier: 3,
		prompt: dedent(`
			Run-length encode the string "aaabbbccccdddddeee". Output format: each
			character followed immediately by its count, concatenated together. For
			example, "aabbc" becomes "a2b2c1". Print only the encoded string,
			nothing else.
		`),
		judge: exactOutput("a3b3c4d5e3"),
	},

	// ── Tier 4: Hard ──
	{
		name: "max_subarray_sum",
		tier: 4,
		prompt: dedent(`
			Find the maximum contiguous subarray sum (Kadane's algorithm) of the
			array [-2, 1, -3, 4, -1, 2, 1, -5, 4]. Print only the number,
			nothing else.
		`),
		judge: exactOutput("6"),
	},
	{
		name: "count_islands",
		tier: 4,
		prompt: dedent(`
			Count the number of islands in a 2D grid. An island is a group of 1s
			connected horizontally or vertically. The grid (4 rows, 5 columns) is:
			Row 0: 1 1 0 0 0
			Row 1: 1 1 0 0 0
			Row 2: 0 0 1 0 0
			Row 3: 0 0 0 1 1
			Print only the count of islands, nothing else.
		`),
		judge: exactOutput("3"),
	},
	{
		name: "levenshtein_distance",
		tier: 4,
		prompt: dedent(`
			Compute the Levenshtein (edit) distance between "kitten" and "sitting".
			Print only the number, nothing else.
		`),
		judge: exactOutput("3"),
	},
	{
		name: "minimum_coins",
		tier: 4,
		prompt: dedent(`
			Find the minimum number of coins from denominations [1, 5, 10, 25]
			needed to make exactly 63 cents. Print only the number, nothing else.
		`),
		judge: exactOutput("6"),
	},
	{
		name: "topological_sort",
		tier: 4,
		prompt: dedent(`
			Perform a topological sort on a directed acyclic graph with these edges:
			A→B, A→C, B→D, C→D, D→E. Print the vertices in a valid topological
			order, separated by spaces, on a single line. Print only the vertex
			names separated by spaces, nothing else.
		`),
		judge: validTopologicalSort([][2]string{
			{"A", "B"}, {"A", "C"}, {"B", "D"}, {"C", "D"}, {"D", "E"},
		}),
	},
	{
		name: "matrix_multiply",
		tier: 4,
		prompt: dedent(`
			Multiply these two matrices and print the result.
			Matrix A (2x3): [[1, 2, 3], [4, 5, 6]]
			Matrix B (3x2): [[7, 8], [9, 10], [11, 12]]
			Print the resulting 2x2 matrix, one row per line, with numbers separated
			by spaces. Print only the matrix, nothing else.
		`),
		judge: exactOutput(dedent(`
			58 64
			139 154
		`)),
	},
	{
		name: "spiral_matrix",
		tier: 4,
		prompt: dedent(`
			Generate a 5x5 spiral matrix filled with numbers 1 to 25 in clockwise
			spiral order starting from the top-left. Print the matrix with one row
			per line, numbers separated by spaces. Print only the matrix,
			nothing else.
		`),
		judge: exactOutput(dedent(`
			1 2 3 4 5
			16 17 18 19 6
			15 24 25 20 7
			14 23 22 21 8
			13 12 11 10 9
		`)),
	},
	{
		name: "knapsack_01",
		tier: 4,
		prompt: dedent(`
			Solve the 0/1 knapsack problem. Capacity: 50. Items (weight, value):
			(10, 60), (20, 100), (30, 120). Print the maximum total value
			achievable. Print only the number, nothing else.
		`),
		judge: exactOutput("220"),
	},
	{
		name: "longest_palindrome_substring",
		tier: 4,
		prompt: dedent(`
			Find the longest palindromic substring of "babad". If there are multiple
			of the same length, print the one that appears first. Print only the
			substring, nothing else.
		`),
		judge: oneOf("bab", "aba"),
	},
	{
		name: "sudoku_solver",
		tier: 4,
		prompt: dedent(`
			Solve this Sudoku puzzle. The grid uses 0 for empty cells:
			5 3 0 0 7 0 0 0 0
			6 0 0 1 9 5 0 0 0
			0 9 8 0 0 0 0 6 0
			8 0 0 0 6 0 0 0 3
			4 0 0 8 0 3 0 0 1
			7 0 0 0 2 0 0 0 6
			0 6 0 0 0 0 2 8 0
			0 0 0 4 1 9 0 0 5
			0 0 0 0 8 0 0 7 9
			Print the completed 9x9 grid with numbers separated by spaces, one row
			per line. Print only the grid, nothing else.
		`),
		judge: exactOutput(dedent(`
			5 3 4 6 7 8 9 1 2
			6 7 2 1 9 5 3 4 8
			1 9 8 3 4 2 5 6 7
			8 5 9 7 6 1 4 2 3
			4 2 6 8 5 3 7 9 1
			7 1 3 9 2 4 8 5 6
			9 6 1 5 3 7 2 8 4
			2 8 7 4 1 9 6 3 5
			3 4 5 2 8 6 1 7 9
		`)),
	},

	// ── Tier 5: Expert ──
	{
		name: "game_of_life",
		tier: 5,
		prompt: dedent(`
			Simulate 10 steps of Conway's Game of Life on an 8x8 grid. The initial
			state has live cells (1) at positions (row, col, 0-indexed): (1,2),
			(2,3), (3,1), (3,2), (3,3). All other cells are dead (0). Print the
			final 8x8 grid after 10 steps, one row per line, with cells separated
			by spaces. Print only the grid, nothing else.
		`),
		judge: exactOutput(dedent(`
			0 0 0 0 0 0 0 0
			0 0 0 0 0 0 0 0
			0 0 0 0 0 0 0 0
			0 0 0 0 0 0 0 0
			0 0 0 0 0 1 0 0
			0 0 0 1 0 1 0 0
			0 0 0 0 1 1 0 0
			0 0 0 0 0 0 0 0
		`)),
	},
	{
		name: "n_queens",
		tier: 5,
		prompt: dedent(`
			Solve the 8-queens problem: place 8 queens on an 8x8 chessboard so that
			no two queens attack each other. Print the board as 8 lines of 8
			characters each, using "Q" for a queen and "." for empty. Separate
			characters with spaces. Print only the board, nothing else.
		`),
		judge: validNQueens(8),
	},
	{
		name: "bigint_factorial_50",
		tier: 5,
		prompt: dedent(`
			Compute 50! (50 factorial). Starlark supports arbitrary-precision
			integers. Print only the number, nothing else.
		`),
		judge: exactOutput("30414093201713378043612608166064768844377641568960512000000000000"),
	},
	{
		name: "postfix_eval",
		tier: 5,
		prompt: dedent(`
			Evaluate the postfix (reverse Polish notation) expression:
			"3 4 + 2 * 7 /"
			Operators are +, -, *, / (integer division). Print only the result as
			an integer, nothing else.
		`),
		judge: exactOutput("2"),
	},
	{
		name: "text_histogram",
		tier: 5,
		prompt: dedent(`
			Count the frequency of each word (case-insensitive) in the text:
			"the cat sat on the mat the cat sat"
			Print each word and its count in the format "word count", one per line,
			sorted by count descending then alphabetically. Print only the
			word-count lines, nothing else.
		`),
		judge: exactOutput(dedent(`
			the 3
			cat 2
			sat 2
			mat 1
			on 1
		`)),
	},
}
