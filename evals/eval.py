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

# Inspect loads this file as a standalone module (not as part of the evals
# package), so normal imports from the package don't work.  Use importlib
# to load judges.py relative to this file.
import importlib.util as _ilu

_spec = _ilu.spec_from_file_location(
    "judges", str(Path(__file__).resolve().parent / "judges.py")
)
_judges_mod = _ilu.module_from_spec(_spec)  # type: ignore[arg-type]
_spec.loader.exec_module(_judges_mod)  # type: ignore[union-attr]
_judge = _judges_mod.judge

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
# Custom scorer
# ---------------------------------------------------------------------------
@scorer(metrics=[accuracy()])
def starlark_output_scorer() -> ...:  # type: ignore[override]
    """Score based on the last successful tool output from the MCP server."""

    async def score(state: TaskState, target: Target) -> Score:
        metadata = state.metadata
        attempts = metadata.get("attempts", "?")
        tool_output = _last_tool_output(state)

        if tool_output is None:
            return Score(
                value=INCORRECT,
                answer="<no tool output>",
                explanation=f"attempts={attempts}, model never called the tool.",
            )

        passed = _judge(tool_output, metadata["judge"])
        return Score(
            value=CORRECT if passed else INCORRECT,
            answer=tool_output,
            explanation=f"scorer={metadata['judge']['scorer']}, attempts={attempts}, passed={passed}",
        )

    return score


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
def _last_tool_output(state: TaskState) -> str | None:
    """Return the text of the last successful tool message, or None."""
    for msg in reversed(state.messages):
        if msg.role == "tool" and msg.error is None:
            return msg.text
    return None


# ---------------------------------------------------------------------------
# Retry solver
# ---------------------------------------------------------------------------
_MAX_ATTEMPTS = 3

_NUDGE = (
    "Your output was not correct. Please try again. "
    "Use the execute_starlark tool and print your answer."
)


@solver
def retry_on_wrong(max_attempts: int = _MAX_ATTEMPTS) -> Solver:
    """Generate, check the tool output, and retry with a nudge if wrong."""

    async def solve(state: TaskState, generate: Generate) -> TaskState:
        for attempt in range(1, max_attempts + 1):
            state = await generate(state)
            output = _last_tool_output(state)
            if output is not None and _judge(output, state.metadata["judge"]):
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
    "Do not explain your work \u2014 just call the appropriate tool."
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
