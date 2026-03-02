"""Inspect AI eval for the Starlark MCP server.

Run with:
    inspect eval evals/eval.py --model anthropic/claude-sonnet-4-6
"""

import os
from pathlib import Path

from inspect_ai import Task, task
from inspect_ai.dataset import Sample
from inspect_ai.scorer import (
    CORRECT,
    INCORRECT,
    Score,
    Target,
    accuracy,
    scorer,
)
from inspect_ai.model import ChatMessageUser
from inspect_ai.solver import (
    Generate,
    Solver,
    TaskState,
    solver,
    system_message,
    use_tools,
)
from inspect_ai.tool import mcp_server_stdio

# ---------------------------------------------------------------------------
# MCP server binary path
# ---------------------------------------------------------------------------
_STARLARK_MCP = os.environ.get(
    "STARLARK_MCP_BIN",
    str(Path(__file__).resolve().parent.parent / "starlark-mcp"),
)


# ---------------------------------------------------------------------------
# Dataset loader
# ---------------------------------------------------------------------------
def _load_dataset() -> list[Sample]:
    """Load cases.yaml and convert to Inspect Samples."""
    import yaml

    cases_path = Path(__file__).resolve().parent / "cases.yaml"
    with open(cases_path) as f:
        cases = yaml.safe_load(f)

    samples = []
    for c in cases:
        # For custom scorers, target is unused by the built-in match scorer;
        # we stash the full case metadata so the scorer can access it.
        target = c.get("judge", {}).get("target", "")
        samples.append(
            Sample(
                id=c["id"],
                input=c["input"],
                target=target,
                metadata=c,
            )
        )
    return samples


# ---------------------------------------------------------------------------
# Judges
# ---------------------------------------------------------------------------


def _exact(output: str, expected: str) -> bool:
    return output.rstrip(" \t\n\r") == expected.rstrip(" \t\n\r")


def _numeric(output: str, expected: float, tolerance: float) -> bool:
    try:
        val = float(output.strip())
    except ValueError:
        return False
    return abs(val - expected) <= tolerance


def _one_of(output: str, accepted: list[str]) -> bool:
    return output.rstrip(" \t\n\r") in accepted


def _topological_sort(output: str, edges: list[list[str]]) -> bool:
    fields = output.strip().split()
    if not fields:
        return False
    vertex_set: set[str] = set()
    for e in edges:
        vertex_set.add(e[0])
        vertex_set.add(e[1])
    if set(fields) != vertex_set or len(fields) != len(vertex_set):
        return False
    pos = {f: i for i, f in enumerate(fields)}
    return all(pos[e[0]] < pos[e[1]] for e in edges)


def _n_queens(output: str, n: int) -> bool:
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
        for c, cell in enumerate(fields):
            if cell == "Q":
                queens += 1
                if c in cols or (r - c) in diag1 or (r + c) in diag2:
                    return False
                cols.add(c)
                diag1.add(r - c)
                diag2.add(r + c)
            elif cell != ".":
                return False
    return queens == n


def _judge(output: str, spec: dict) -> bool:
    s = spec["scorer"]
    if s == "exact":
        return _exact(output, spec["target"])
    elif s == "numeric":
        return _numeric(output, spec["expected"], spec["tolerance"])
    elif s == "one_of":
        return _one_of(output, spec["accepted"])
    elif s == "topological_sort":
        return _topological_sort(output, spec["edges"])
    elif s == "n_queens":
        return _n_queens(output, spec["n"])
    return False


# ---------------------------------------------------------------------------
# Custom scorer
# ---------------------------------------------------------------------------
@scorer(metrics=[accuracy()])
def starlark_output_scorer() -> ...:  # type: ignore[override]
    """Score the model's final assistant message against the judge criteria."""

    async def score(state: TaskState, target: Target) -> Score:
        metadata = state.metadata
        attempts = metadata.get("attempts", "?")
        answer = state.output.completion if state.output else ""

        if not answer.strip():
            return Score(
                value=INCORRECT,
                answer="",
                explanation=f"attempts={attempts}, no answer in final message.",
            )

        passed = _judge(answer, metadata["judge"])
        return Score(
            value=CORRECT if passed else INCORRECT,
            answer=answer,
            explanation=f"scorer={metadata['judge']['scorer']}, attempts={attempts}, passed={passed}",
        )

    return score


# ---------------------------------------------------------------------------
# Retry solver
# ---------------------------------------------------------------------------
_MAX_ATTEMPTS = 3

_NUDGE = (
    "Your output was not correct. Please try again. "
    "Use the execute_starlark tool and print your answer."
)


def _completion(state: TaskState) -> str:
    """Return the model's latest completion text."""
    return state.output.completion if state.output else ""


@solver
def retry_on_wrong(max_attempts: int = _MAX_ATTEMPTS) -> Solver:
    """Generate, check the model's response, and retry with a nudge if wrong."""

    async def solve(state: TaskState, generate: Generate) -> TaskState:
        for attempt in range(1, max_attempts + 1):
            state = await generate(state)
            if _judge(_completion(state), state.metadata["judge"]):
                state.metadata["attempts"] = attempt
                return state
            if attempt < max_attempts:
                state.messages.append(ChatMessageUser(content=_NUDGE))
        state.metadata["attempts"] = max_attempts
        return state

    return solve


# ---------------------------------------------------------------------------
# Task definition
# ---------------------------------------------------------------------------
SYSTEM_PROMPT = (
    "You have access to tools. Use them to solve the task. "
    "After calling the tool, respond with ONLY the exact output from the tool. "
    "No explanation, no formatting, no markdown \u2014 just the raw output."
)


@task
def starlark_eval() -> Task:
    """Evaluate LLM ability to use the Starlark MCP tool."""
    return Task(
        dataset=_load_dataset(),
        solver=[
            system_message(SYSTEM_PROMPT),
            use_tools(
                mcp_server_stdio(
                    name="starlark-mcp",
                    command=_STARLARK_MCP,
                ),
            ),
            retry_on_wrong(),
        ],
        scorer=starlark_output_scorer(),
        max_messages=24,
    )
