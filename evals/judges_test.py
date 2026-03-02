"""Tests for eval judging functions."""

from evals.eval import (
    _exact as exact,
    _judge as judge,
    _n_queens as n_queens,
    _numeric as numeric,
    _one_of as one_of,
    _topological_sort as topological_sort,
)


# ---------------------------------------------------------------------------
# exact
# ---------------------------------------------------------------------------
class TestExact:
    def test_match(self):
        assert exact("hello", "hello")

    def test_trailing_whitespace(self):
        assert exact("hello  \n", "hello")
        assert exact("hello", "hello\n")

    def test_mismatch(self):
        assert not exact("hello", "world")

    def test_empty(self):
        assert exact("", "")
        assert not exact("x", "")


# ---------------------------------------------------------------------------
# numeric
# ---------------------------------------------------------------------------
class TestNumeric:
    def test_exact_match(self):
        assert numeric("3.14", 3.14, 0.001)

    def test_within_tolerance(self):
        assert numeric("3.141", 3.14, 0.01)

    def test_outside_tolerance(self):
        assert not numeric("3.2", 3.14, 0.01)

    def test_integer_output(self):
        assert numeric("42", 42.0, 0.0)

    def test_non_numeric(self):
        assert not numeric("abc", 1.0, 0.1)

    def test_whitespace(self):
        assert numeric("  3.14  ", 3.14, 0.001)


# ---------------------------------------------------------------------------
# one_of
# ---------------------------------------------------------------------------
class TestOneOf:
    def test_match_first(self):
        assert one_of("a", ["a", "b", "c"])

    def test_match_last(self):
        assert one_of("c", ["a", "b", "c"])

    def test_no_match(self):
        assert not one_of("d", ["a", "b", "c"])

    def test_trailing_whitespace(self):
        assert one_of("a\n", ["a", "b"])


# ---------------------------------------------------------------------------
# topological_sort
# ---------------------------------------------------------------------------
class TestTopologicalSort:
    EDGES = [["a", "b"], ["b", "c"], ["a", "c"]]

    def test_valid_order(self):
        assert topological_sort("a b c", self.EDGES)

    def test_invalid_order(self):
        assert not topological_sort("c b a", self.EDGES)

    def test_partial_invalid(self):
        assert not topological_sort("a c b", self.EDGES)

    def test_missing_vertex(self):
        assert not topological_sort("a b", self.EDGES)

    def test_extra_vertex(self):
        assert not topological_sort("a b c d", self.EDGES)

    def test_duplicate_vertex(self):
        assert not topological_sort("a b b c", self.EDGES)

    def test_empty_output(self):
        assert not topological_sort("", self.EDGES)

    def test_whitespace_padding(self):
        assert topological_sort("  a b c  \n", self.EDGES)

    def test_diamond(self):
        edges = [["a", "b"], ["a", "c"], ["b", "d"], ["c", "d"]]
        assert topological_sort("a b c d", edges)
        assert topological_sort("a c b d", edges)
        assert not topological_sort("a d b c", edges)


# ---------------------------------------------------------------------------
# n_queens
# ---------------------------------------------------------------------------
class TestNQueens:
    def test_valid_4queens(self):
        board = ". Q . .\n. . . Q\nQ . . .\n. . Q ."
        assert n_queens(board, 4)

    def test_same_column(self):
        board = "Q . . .\nQ . . .\n. . . Q\n. . Q ."
        assert not n_queens(board, 4)

    def test_same_diagonal(self):
        board = "Q . . .\n. . Q .\n. Q . .\n. . . Q"
        assert not n_queens(board, 4)

    def test_wrong_queen_count(self):
        board = "Q . . .\n. . . .\n. . . Q\n. . Q ."
        assert not n_queens(board, 4)

    def test_wrong_row_count(self):
        board = ". Q . .\n. . . Q\nQ . . ."
        assert not n_queens(board, 4)

    def test_wrong_column_count(self):
        board = ". Q .\n. . . Q\nQ . . .\n. . Q ."
        assert not n_queens(board, 4)

    def test_invalid_cell(self):
        board = ". Q . .\n. . . X\nQ . . .\n. . Q ."
        assert not n_queens(board, 4)

    def test_valid_1queen(self):
        assert n_queens("Q", 1)


# ---------------------------------------------------------------------------
# judge dispatch
# ---------------------------------------------------------------------------
class TestJudge:
    def test_exact(self):
        spec = {"scorer": "exact", "target": "hello"}
        assert judge("hello", spec)
        assert not judge("world", spec)

    def test_numeric(self):
        spec = {"scorer": "numeric", "expected": 3.14, "tolerance": 0.01}
        assert judge("3.14", spec)
        assert not judge("9.99", spec)

    def test_one_of(self):
        spec = {"scorer": "one_of", "accepted": ["yes", "no"]}
        assert judge("yes", spec)
        assert not judge("maybe", spec)

    def test_topological_sort(self):
        spec = {"scorer": "topological_sort", "edges": [["a", "b"]]}
        assert judge("a b", spec)
        assert not judge("b a", spec)

    def test_n_queens(self):
        spec = {"scorer": "n_queens", "n": 1}
        assert judge("Q", spec)
        assert not judge(".", spec)

    def test_unknown_scorer(self):
        assert not judge("anything", {"scorer": "unknown"})
