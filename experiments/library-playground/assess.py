"""External behavior assessment only. No build, Agent start, repair or Apply."""

import argparse
import copy
import hashlib
import json
from pathlib import Path
import platform
import sys

from driver import Limits, Process, TransportFailure, strict_json
from oracle import build_cases, evaluate_step, resolve_request, witness_case


ROOT = Path(__file__).resolve().parent


def sha256(path):
    digest = hashlib.sha256()
    with open(path, "rb") as stream:
        for block in iter(lambda: stream.read(65536), b""):
            digest.update(block)
    return digest.hexdigest()


def protected_hashes():
    paths = [ROOT / name for name in ("assess.py", "driver.py", "oracle.py")]
    paths += sorted((ROOT / "expectations").glob("*.json"))
    return {path.relative_to(ROOT).as_posix(): sha256(path) for path in paths}


def data_hash(data):
    return hashlib.sha256(json.dumps(data, sort_keys=True, ensure_ascii=False,
                                     separators=(",", ":")).encode()).hexdigest()


def run_case(case, command, cwd, limits, keep_going=False):
    result = {"id": case["id"], "families": case["families"], "planned_steps": len(case["steps"]),
              "executed_steps": 0, "assertions": 0, "errors": [], "steps": [], "passed": False}
    captures = {}
    process = None
    try:
        process = Process(command, cwd, limits)
        for index, step in enumerate(case["steps"]):
            request = resolve_request(step, captures) if "request" in step else None
            response = process.request(request, step.get("raw"))
            checked = evaluate_step(step, response, captures)
            result["executed_steps"] += 1
            result["assertions"] += checked["assertions"]
            errors = checked["errors"]
            result["steps"].append({"index": index, "witness": step.get("witness"),
                                    "assertions": checked["assertions"], "errors": errors})
            result["errors"].extend("step " + str(index) + ": " + error for error in errors)
            if errors and not keep_going:
                break
        process.finish()
        result["assertions"] += 1  # Normal EOF, no extra response, exit zero.
    except (TransportFailure, OSError, KeyError, ValueError) as exc:
        result["errors"].append(type(exc).__name__ + ": " + str(exc))
    finally:
        if process is not None:
            process.close()
            result["trace"] = process.trace()
    result["passed"] = not result["errors"] and result["executed_steps"] == result["planned_steps"]
    return result


def validate_manifest(manifest):
    if not isinstance(manifest, dict):
        raise ValueError("manifest must be a JSON object")
    for key in ("experiment_id", "artifact_id", "model_digest", "source_tree"):
        if not isinstance(manifest.get(key), str) or not manifest[key].strip():
            raise ValueError("missing external identity: " + key)
    if manifest.get("profile") not in ("m1", "m2"):
        raise ValueError("profile must be m1 or m2")
    command = manifest.get("command")
    if not isinstance(command, list) or not command or any(not isinstance(x, str) or not x for x in command):
        raise ValueError("command must be nonempty literal argv")
    executable = Path(command[0])
    cwd = Path(manifest["cwd"])
    if not executable.is_absolute() or not cwd.is_absolute() or not cwd.is_dir():
        raise ValueError("executable and existing cwd must be absolute")
    if cwd.resolve().is_relative_to(ROOT) or ROOT.is_relative_to(cwd.resolve()):
        raise ValueError("target cwd must be separate from protected evaluator")
    actual = protected_hashes()
    if manifest.get("protected_sha256") != actual:
        raise ValueError("protected evaluator/expectation binding mismatch")
    inputs = manifest.get("program_sha256", {})
    bound_paths = {Path(name).resolve() for name in inputs}
    if executable.resolve() not in bound_paths:
        raise ValueError("executable bytes must be bound in program_sha256")
    # Bind any literal argv file too (script fixtures, configuration, etc.).
    for argument in command[1:]:
        path = Path(argument)
        candidate = path if path.is_absolute() else cwd / path
        if candidate.is_file() and candidate.resolve() not in bound_paths:
            raise ValueError("argv file must be bound: " + str(candidate.resolve()))
    for name, expected in inputs.items():
        path = Path(name)
        if not path.is_absolute() or sha256(path) != expected:
            raise ValueError("program input hash mismatch: " + name)
    return Limits(**manifest["limits"])


def discrimination(profile, command, cwd, limits):
    """An expected mismatch counts only at the three declared witness observations."""
    other = "m2" if profile == "m1" else "m1"
    case = copy.deepcopy(witness_case(other))
    # Full snapshots belong to the primary witness; the opposite-profile probe
    # isolates the three intended observations from that already checked state.
    case["steps"] = [step for step in case["steps"] if not step.get("snapshot")]
    for step in case["steps"]:
        if step.get("witness") == "open-loan-count":
            # The primary witness checks the full list. The opposite policy can
            # expect an uncaptured fourth ID; here discriminate count separately.
            step["expect"]["result"].pop("loans", None)
    result = run_case(case, command, cwd, limits, keep_going=True)
    mismatch_labels = {step["witness"] for step in result["steps"] if step["errors"]}
    desired = {"fourth-admission", "open-loan-count", "copy-4-availability"}
    # A transport/crash error has no step assertion and cannot prove discrimination.
    errors_from_steps = sum(len(step["errors"]) for step in result["steps"])
    result["expected_mismatch_confirmed"] = (
        result["executed_steps"] == result["planned_steps"]
        and mismatch_labels == desired
        and len(result["errors"]) == errors_from_steps
        and result["trace"]["exit_code"] == 0)
    result["opposite_profile"] = other
    result["case_sha256"] = data_hash(case)
    return result


def assess(manifest):
    report = {"schema": "markitect.library-assessment/v1", "kind": "behavior-only",
              "manifest": manifest, "manifest_sha256": data_hash(manifest), "passed": False,
              "environment": {"python": sys.version, "platform": platform.platform()},
              "isolation": {"credentials": "filtered environment", "network": "not-enforced",
                            "filesystem": "not-sandboxed", "process_cleanup": "direct child only"},
              "external_identities": "caller-supplied; compiler/source/history not verified here",
              "source_review": "not-run; inspect above-limit preservation in exact A2 source",
              "live_upgrade": "unproven; fresh process cannot contain above-limit M2 loans",
              "apply_evidence": "not-run", "human_acceptance": "not-recorded",
              "cases": [], "errors": []}
    try:
        limits = validate_manifest(manifest)
        cases = build_cases(manifest["profile"])
        report["cases_sha256"] = data_hash(cases)
        for case in cases:
            report["cases"].append(run_case(case, manifest["command"], manifest["cwd"], limits))
        report["discrimination"] = discrimination(manifest["profile"], manifest["command"], manifest["cwd"], limits)
        validate_manifest(manifest)  # Before AND after: changes invalidate evidence.
        report["passed"] = (bool(cases) and all(case["passed"] for case in report["cases"])
                            and report["discrimination"]["expected_mismatch_confirmed"])
        if not report["passed"]:
            report["errors"].append("realization/behavior: required cases failed")
    except (OSError, ValueError, KeyError, TypeError) as exc:
        report["errors"].append("assessment/integrity or preparation/environment: " + str(exc))
    report["planned_cases"] = len(report["cases"])
    report["executed_cases"] = sum(case["executed_steps"] > 0 for case in report["cases"])
    report["executed_steps"] = sum(case["executed_steps"] for case in report["cases"])
    report["assertions"] = sum(case["assertions"] for case in report["cases"])
    report["missing_steps"] = sum(case["planned_steps"] - case["executed_steps"] for case in report["cases"])
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--hashes", action="store_true", help="print protected file hashes; do not execute")
    parser.add_argument("--manifest", type=Path)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    if args.hashes:
        print(json.dumps(protected_hashes(), indent=2))
        return 0
    if args.manifest is None or args.output is None:
        parser.error("--manifest and --output are required")
    # Refuse overwrites so a failed attempt cannot disappear behind a later run.
    result = None
    try:
        manifest = strict_json(args.manifest.read_text(encoding="utf-8"))
    except (OSError, ValueError) as exc:
        manifest = None
        result = {"passed": False, "errors": ["preparation/environment: " + str(exc)]}
    destination = args.output.resolve()
    if destination.is_relative_to(ROOT) or destination == args.manifest.resolve():
        parser.error("output must be outside protected evaluator and cannot replace the manifest")
    if isinstance(manifest, dict) and isinstance(manifest.get("cwd"), str):
        if destination.is_relative_to(Path(manifest["cwd"]).resolve()):
            parser.error("output must be outside target working directory")
    with args.output.open("x", encoding="utf-8") as output:
        if result is None:
            result = assess(manifest)
        json.dump(result, output, indent=2, ensure_ascii=False)
        output.write("\n")
    return 0 if result["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
