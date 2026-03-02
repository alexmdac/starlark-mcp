"""Judging functions for eval cases.

Each judge takes the raw tool output string and returns True if correct.
"""


def exact(output: str, expected: str) -> bool:
    """Exact match after stripping trailing whitespace."""
    return output.rstrip(" \t\n\r") == expected.rstrip(" \t\n\r")


def numeric(output: str, expected: float, tolerance: float) -> bool:
    """Numeric match within tolerance."""
    try:
        val = float(output.strip())
    except ValueError:
        return False
    return abs(val - expected) <= tolerance


def one_of(output: str, accepted: list[str]) -> bool:
    """Match any of the accepted values after stripping trailing whitespace."""
    return output.rstrip(" \t\n\r") in accepted


def topological_sort(output: str, edges: list[list[str]]) -> bool:
    """Validate that output is a valid topological ordering for the given edges."""
    trimmed = output.strip()
    fields = trimmed.split()
    if not fields:
        return False
    vertex_set: set[str] = set()
    for e in edges:
        vertex_set.add(e[0])
        vertex_set.add(e[1])
    output_set = set(fields)
    if len(output_set) != len(vertex_set) or len(fields) != len(vertex_set):
        return False
    if output_set != vertex_set:
        return False
    pos = {f: i for i, f in enumerate(fields)}
    for e in edges:
        if pos[e[0]] >= pos[e[1]]:
            return False
    return True


def n_queens(output: str, n: int) -> bool:
    """Validate an N-Queens solution on an n×n board."""
    lines = output.strip().split("\n")
    if len(lines) != n:
        return False
    queens = 0
    cols: set[int] = set()
    diag1: set[int] = set()
    diag2: set[int] = set()
    for r, line in enumerate(lines):
        fields = line.split()
        if len(fields) != n:
            return False
        for c_idx, cell in enumerate(fields):
            if cell == "Q":
                queens += 1
                if c_idx in cols or (r - c_idx) in diag1 or (r + c_idx) in diag2:
                    return False
                cols.add(c_idx)
                diag1.add(r - c_idx)
                diag2.add(r + c_idx)
            elif cell != ".":
                return False
    return queens == n


def judge(output: str, judge_spec: dict) -> bool:
    """Dispatch to the appropriate judge based on the judge spec from cases.yaml."""
    scorer_type = judge_spec["scorer"]
    if scorer_type == "exact":
        return exact(output, judge_spec["target"])
    elif scorer_type == "numeric":
        return numeric(output, judge_spec["expected"], judge_spec["tolerance"])
    elif scorer_type == "one_of":
        return one_of(output, judge_spec["accepted"])
    elif scorer_type == "topological_sort":
        return topological_sort(output, judge_spec["edges"])
    elif scorer_type == "n_queens":
        return n_queens(output, judge_spec["n"])
    else:
        return False
