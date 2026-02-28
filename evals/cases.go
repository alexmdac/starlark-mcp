//go:build eval

package evals

import (
	"math"
	"strconv"
	"strings"
)

// Case describes a single eval case: a prompt for the LLM and a judge function.
type Case struct {
	Name   string
	Tier   int
	Prompt string
	Judge  func(output string) bool
}

// ExactOutput trims trailing whitespace from both expected and actual, then compares.
func ExactOutput(expected string) func(string) bool {
	return func(output string) bool {
		return strings.TrimRight(output, " \t\n\r") == strings.TrimRight(expected, " \t\n\r")
	}
}

// NumericOutput parses the output as a float and checks if it is within tolerance of expected.
func NumericOutput(expected float64, tolerance float64) func(string) bool {
	return func(output string) bool {
		trimmed := strings.TrimSpace(output)
		v, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return false
		}
		return math.Abs(v-expected) <= tolerance
	}
}

// ValidTopologicalSort checks that the output is a valid topological ordering for the given edges.
// Each edge is [from, to] meaning "from" must appear before "to" in the output.
func ValidTopologicalSort(edges [][2]string) func(string) bool {
	return func(output string) bool {
		trimmed := strings.TrimSpace(output)
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			return false
		}

		// Collect all vertices from edges.
		vertexSet := make(map[string]bool)
		for _, e := range edges {
			vertexSet[e[0]] = true
			vertexSet[e[1]] = true
		}

		// Check that the output contains exactly the right vertices.
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

		// Build position map.
		pos := make(map[string]int)
		for i, f := range fields {
			pos[f] = i
		}

		// Check all edges: from must come before to.
		for _, e := range edges {
			if pos[e[0]] >= pos[e[1]] {
				return false
			}
		}
		return true
	}
}

// Cases is the full set of eval cases.
var Cases = []Case{
	// ── Tier 1: Basics ──
	{
		Name:   "print_numbers_1_to_20",
		Tier:   1,
		Prompt: "Print the integers 1 to 20, one per line. Each line should contain just the number, nothing else.",
		Judge:  ExactOutput("1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n14\n15\n16\n17\n18\n19\n20"),
	},
	{
		Name:   "reverse_string",
		Tier:   1,
		Prompt: `Reverse the string "Hello, World!" and print the result. Print only the reversed string, nothing else.`,
		Judge:  ExactOutput("!dlroW ,olleH"),
	},
	{
		Name:   "sin_pi_over_6",
		Tier:   1,
		Prompt: "Compute sin(π/6) and print the numeric result. Print only the number, nothing else.",
		Judge:  NumericOutput(0.5, 0.001),
	},

	// ── Tier 2: Simple Algorithms ──
	{
		Name:   "fizzbuzz",
		Tier:   2,
		Prompt: `Print FizzBuzz for numbers 1 through 30, one entry per line. For multiples of 3 print "Fizz", for multiples of 5 print "Buzz", for multiples of both print "FizzBuzz", otherwise print the number. Print only the output, nothing else.`,
		Judge:  ExactOutput("1\n2\nFizz\n4\nBuzz\nFizz\n7\n8\nFizz\nBuzz\n11\nFizz\n13\n14\nFizzBuzz\n16\n17\nFizz\n19\nBuzz\nFizz\n22\n23\nFizz\nBuzz\n26\nFizz\n28\n29\nFizzBuzz"),
	},
	{
		Name:   "is_prime_104729",
		Tier:   2,
		Prompt: `Determine whether 104729 is a prime number. Print "true" if it is prime, or "false" if it is not. Print only that single word, nothing else.`,
		Judge:  ExactOutput("true"),
	},
	{
		Name:   "gcd_48_18",
		Tier:   2,
		Prompt: "Compute the greatest common divisor (GCD) of 48 and 18. Print only the number, nothing else.",
		Judge:  ExactOutput("6"),
	},
	{
		Name:   "count_vowels",
		Tier:   2,
		Prompt: `Count the number of vowels (a, e, i, o, u, case-insensitive) in the string "The quick brown fox jumps over the lazy dog". Print only the count, nothing else.`,
		Judge:  ExactOutput("11"),
	},
	{
		Name:   "decimal_to_binary",
		Tier:   2,
		Prompt: "Convert the decimal number 255 to its binary string representation with no prefix (no \"0b\"). Print only the binary string, nothing else.",
		Judge:  ExactOutput("11111111"),
	},
	{
		Name:   "pascals_triangle",
		Tier:   2,
		Prompt: "Print the first 10 rows of Pascal's triangle (rows 0 through 9). Print one row per line, with numbers separated by single spaces. Row 0 is \"1\", row 1 is \"1 1\", etc. Print only the triangle, nothing else.",
		Judge:  ExactOutput("1\n1 1\n1 2 1\n1 3 3 1\n1 4 6 4 1\n1 5 10 10 5 1\n1 6 15 20 15 6 1\n1 7 21 35 35 21 7 1\n1 8 28 56 70 56 28 8 1\n1 9 36 84 126 126 84 36 9 1"),
	},

	// ── Tier 3: Intermediate ──
	{
		Name:   "sieve_of_eratosthenes",
		Tier:   3,
		Prompt: "Use the Sieve of Eratosthenes to find all prime numbers below 10000. Print three lines: first line is the count of primes found, second line is the first 10 primes separated by spaces, third line is the last 10 primes separated by spaces. Print only these three lines, nothing else.",
		Judge:  ExactOutput("1229\n2 3 5 7 11 13 17 19 23 29\n9887 9901 9907 9923 9929 9931 9941 9949 9967 9973"),
	},
	{
		Name:   "fibonacci_30",
		Tier:   3,
		Prompt: "Print the first 30 Fibonacci numbers F(0) through F(29), one per line. F(0)=0, F(1)=1, F(n)=F(n-1)+F(n-2). Print only the numbers, one per line, nothing else.",
		Judge:  ExactOutput("0\n1\n1\n2\n3\n5\n8\n13\n21\n34\n55\n89\n144\n233\n377\n610\n987\n1597\n2584\n4181\n6765\n10946\n17711\n28657\n46368\n75025\n121393\n196418\n317811\n514229"),
	},
	{
		Name:   "balanced_parentheses",
		Tier:   3,
		Prompt: `Check whether each of the following strings has balanced parentheses. For each string, print "true" if balanced or "false" if not, one result per line in order. The strings are: "(()())", "(()", "()()", ")(", "", "((()))", "(()))". Print only "true" or "false" on each line, nothing else.`,
		Judge:  ExactOutput("true\nfalse\ntrue\nfalse\ntrue\ntrue\nfalse"),
	},
	{
		Name:   "longest_common_subsequence",
		Tier:   3,
		Prompt: `Find the length of the longest common subsequence of "ABCBDAB" and "BDCAB". Print only the number, nothing else.`,
		Judge:  ExactOutput("4"),
	},
	{
		Name:   "roman_numerals",
		Tier:   3,
		Prompt: `Convert each of the following integers to Roman numerals and print each on its own line: 1, 4, 9, 14, 42, 99, 1994, 3999. Print only the Roman numeral strings, one per line, nothing else.`,
		Judge:  ExactOutput("I\nIV\nIX\nXIV\nXLII\nXCIX\nMCMXCIV\nMMMCMXCIX"),
	},
	{
		Name:   "run_length_encoding",
		Tier:   3,
		Prompt: `Run-length encode the string "aaabbbccccdddddeee". Output format: each character followed immediately by its count, concatenated together. For example, "aabbc" becomes "a2b2c1". Print only the encoded string, nothing else.`,
		Judge:  ExactOutput("a3b3c4d5e3"),
	},

	// ── Tier 4: Hard ──
	{
		Name:   "max_subarray_sum",
		Tier:   4,
		Prompt: "Find the maximum contiguous subarray sum (Kadane's algorithm) of the array [-2, 1, -3, 4, -1, 2, 1, -5, 4]. Print only the number, nothing else.",
		Judge:  ExactOutput("6"),
	},
	{
		Name:   "count_islands",
		Tier:   4,
		Prompt: "Count the number of islands in a 2D grid. An island is a group of 1s connected horizontally or vertically. The grid (4 rows, 5 columns) is:\nRow 0: 1 1 0 0 0\nRow 1: 1 1 0 0 0\nRow 2: 0 0 1 0 0\nRow 3: 0 0 0 1 1\nPrint only the count of islands, nothing else.",
		Judge:  ExactOutput("3"),
	},
	{
		Name:   "levenshtein_distance",
		Tier:   4,
		Prompt: `Compute the Levenshtein (edit) distance between "kitten" and "sitting". Print only the number, nothing else.`,
		Judge:  ExactOutput("3"),
	},
	{
		Name:   "minimum_coins",
		Tier:   4,
		Prompt: "Find the minimum number of coins from denominations [1, 5, 10, 25] needed to make exactly 63 cents. Print only the number, nothing else.",
		Judge:  ExactOutput("6"),
	},
	{
		Name:   "topological_sort",
		Tier:   4,
		Prompt: `Perform a topological sort on a directed acyclic graph with these edges: A→B, A→C, B→D, C→D, D→E. Print the vertices in a valid topological order, separated by spaces, on a single line. Print only the vertex names separated by spaces, nothing else.`,
		Judge: ValidTopologicalSort([][2]string{
			{"A", "B"}, {"A", "C"}, {"B", "D"}, {"C", "D"}, {"D", "E"},
		}),
	},
	{
		Name: "sudoku_solver",
		Tier: 4,
		Prompt: `Solve this Sudoku puzzle. The grid uses 0 for empty cells:
5 3 0 0 7 0 0 0 0
6 0 0 1 9 5 0 0 0
0 9 8 0 0 0 0 6 0
8 0 0 0 6 0 0 0 3
4 0 0 8 0 3 0 0 1
7 0 0 0 2 0 0 0 6
0 6 0 0 0 0 2 8 0
0 0 0 4 1 9 0 0 5
0 0 0 0 8 0 0 7 9
Print the completed 9x9 grid with numbers separated by spaces, one row per line. Print only the grid, nothing else.`,
		Judge: ExactOutput("5 3 4 6 7 8 9 1 2\n6 7 2 1 9 5 3 4 8\n1 9 8 3 4 2 5 6 7\n8 5 9 7 6 1 4 2 3\n4 2 6 8 5 3 7 9 1\n7 1 3 9 2 4 8 5 6\n9 6 1 5 3 7 2 8 4\n2 8 7 4 1 9 6 3 5\n3 4 5 2 8 6 1 7 9"),
	},
}
