"""Summarize MCP tool errors from the most recent Inspect eval log.

Usage:
    python errors.py [LOG_FILE]

If no log file is given, uses the most recent .eval in logs/.
"""

import json
import re
import sys
import zipfile
from collections import Counter
from pathlib import Path


def find_latest_log() -> Path:
    logs_dir = Path(__file__).resolve().parent.parent / "logs"
    logs = sorted(logs_dir.glob("*.eval"))
    if not logs:
        print("No eval logs found in logs/", file=sys.stderr)
        sys.exit(1)
    return logs[-1]


def extract_errors(log_path: Path) -> tuple[dict, Counter, list]:
    """Return (eval_info, error_counts, per_sample_details)."""
    errors = Counter()
    details = []  # (sample_id, passed, attempt, [errors])

    with zipfile.ZipFile(log_path) as zf:
        with zf.open("header.json") as f:
            header = json.load(f)

        eval_info = {
            "model": header["eval"]["model"],
            "samples": header["results"]["total_samples"],
            "accuracy": header["results"]["scores"][0]["metrics"]["accuracy"]["value"],
        }

        for name in sorted(zf.namelist()):
            if not name.startswith("samples/"):
                continue
            with zf.open(name) as f:
                s = json.load(f)

            sid = s["id"]
            sc = s["scores"]["starlark_output_scorer"]
            passed = sc["value"] == "C"
            expl = sc.get("explanation", "")

            sample_errors = []
            for msg in s.get("messages", []):
                if msg.get("role") != "tool":
                    continue
                err_obj = msg.get("error")
                if not err_obj:
                    continue
                raw = (
                    err_obj.get("message", "")
                    if isinstance(err_obj, dict)
                    else str(err_obj)
                )
                if not raw:
                    continue
                # Normalize: strip location prefix and common wrappers
                err = re.sub(r"LLM supplied program:\d+:\d+: ", "", raw)
                err = re.sub(r"^failed to execute program: ", "", err)
                errors[err] += 1
                sample_errors.append(err)

            details.append((sid, passed, expl, sample_errors))

    return eval_info, errors, details


def main():
    if len(sys.argv) > 1:
        log_path = Path(sys.argv[1])
    else:
        log_path = find_latest_log()

    eval_info, errors, details = extract_errors(log_path)

    print(f"Log:      {log_path.name}")
    print(f"Model:    {eval_info['model']}")
    print(f"Accuracy: {eval_info['accuracy']:.1%} ({eval_info['samples']} samples)")
    print()

    if not errors:
        print("No tool errors found.")
        return

    print(f"Tool errors ({sum(errors.values())} total):")
    print()
    for err, count in errors.most_common():
        print(f"  {count:3d}x  {err[:140]}")

    # Show which failed samples had errors
    failed_with_errors = [
        (sid, errs) for sid, passed, _, errs in details if not passed and errs
    ]
    if failed_with_errors:
        print(f"\nFailed samples with errors ({len(failed_with_errors)}):")
        print()
        for sid, errs in failed_with_errors:
            print(f"  {sid}:")
            for e in errs:
                print(f"    - {e[:120]}")


if __name__ == "__main__":
    main()
