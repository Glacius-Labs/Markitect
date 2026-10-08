#!/usr/bin/env python3
"""Run the two-cycle Classic Commerce intent-change example.

This example is deliberately finite and fixture-specific. It reuses the
published v0.14.1 CLI smoke helpers, accepts only one exact synthetic intent
change, and records every command and result. It is not a provider adapter,
general intent translator, or human-acceptance mechanism.
"""
from __future__ import annotations

import argparse
import base64
import difflib
import hashlib
import json
import os
import platform
import shutil
import signal
import subprocess
import sys
import tempfile
import time
from pathlib import Path
from typing import Any

sys.dont_write_bytecode = True
try:
    import smoke
except ImportError as exc:  # pragma: no cover - exercised by direct script invocation
    raise SystemExit("change_smoke.py must be run beside smoke.py") from exc

BASELINE_PURPOSE = (
    "  Creates an order total from a positive integer quantity and a non-negative\n"
    "  decimal unit price. Returns quantity multiplied by unit price. Rejects zero\n"
    "  or negative quantity and negative unit price with ArgumentOutOfRangeException.\n"
)
MINIMUM_TWO_PURPOSE = (
    "  Creates an order total from an integer quantity of at least two and a\n"
    "  non-negative decimal unit price. Returns quantity multiplied by unit price.\n"
    "  Rejects quantity below two and negative unit price with ArgumentOutOfRangeException.\n"
)
BASELINE_PROBE = 'AssertEqual(1m, handler.Handle(1, 1m), "one item allowed");\n'
MINIMUM_TWO_PROBE = 'AssertThrowsArgumentOutOfRange(() => handler.Handle(1, 1m), "one item below accepted minimum");\n'
USE_CASE_PATH = "examples/classic-commerce/definitions/create-order.use-case.yaml"
PROGRAM_PATH = "examples/classic-commerce/checks/Program.cs"
MAX_WALL_SECONDS = 1800
MAX_COMMAND_SECONDS = 480
MAX_NATIVE_MUTATING_STARTS = 12
MAX_DETERMINISTIC_ROLE_RESERVATIONS = 24


class ChangeSmokeFailure(RuntimeError):
    pass


class ProcessBudget:
    """A sequential process-tree runner with a single cumulative deadline."""

    def __init__(self, output: Path, deadline: float) -> None:
        self.output = output
        self.deadline = deadline
        self.logs = output / "process-logs"
        self.logs.mkdir()
        self.commands: list[dict[str, Any]] = []
        self.native_mutating_starts = 0
        self.role_reservations = 0
        self.active = 0
        self.max_active = 0

    def reserve(self, argv: list[str]) -> None:
        action = None
        if len(argv) >= 4 and argv[1] == "canonical":
            for index, arg in enumerate(argv[:-1]):
                if arg == "--action":
                    action = argv[index + 1]
                    break
        if action in {"controller-execute", "controller-apply", "controller-verify"}:
            if self.native_mutating_starts >= MAX_NATIVE_MUTATING_STARTS:
                raise ChangeSmokeFailure("native controller start budget exhausted before command launch")
            self.native_mutating_starts += 1
        if action == "controller-execute":
            reserve_roles = 1  # One .NET Executor; Markdown is a deterministic Module.
        elif action == "controller-verify":
            reserve_roles = 2  # Parent and child assurance Verifier calls.
        else:
            reserve_roles = 0
        if self.role_reservations + reserve_roles > MAX_DETERMINISTIC_ROLE_RESERVATIONS:
            raise ChangeSmokeFailure("deterministic role-call reservation budget exhausted before command launch")
        self.role_reservations += reserve_roles

    def __call__(self, argv: list[str], cwd: Path, env: dict[str, str] | None = None,
                 input_bytes: bytes | None = None) -> subprocess.CompletedProcess[bytes]:
        remaining = self.deadline - time.monotonic()
        if remaining <= 0:
            raise ChangeSmokeFailure("1800-second cumulative wall-clock budget expired before command launch")
        self.reserve(argv)
        # Reserve bounded cleanup time inside, rather than beyond, the total
        # wall-clock budget in case the process needs tree termination.
        timeout = min(MAX_COMMAND_SECONDS, max(0.01, remaining - 5.0))
        index = len(self.commands) + 1
        stdout_path = self.logs / f"{index:04d}.stdout.bin"
        stderr_path = self.logs / f"{index:04d}.stderr.bin"
        started = time.monotonic()
        self.active += 1
        self.max_active = max(self.max_active, self.active)
        popen_kwargs: dict[str, Any] = {
            "cwd": cwd,
            "env": env,
            "stdin": subprocess.PIPE if input_bytes is not None else subprocess.DEVNULL,
            "stdout": subprocess.PIPE,
            "stderr": subprocess.PIPE,
            "shell": False,
        }
        if os.name == "nt":
            popen_kwargs["creationflags"] = subprocess.CREATE_NEW_PROCESS_GROUP
        else:
            popen_kwargs["start_new_session"] = True
        try:
            process = subprocess.Popen(argv, **popen_kwargs)
        except OSError as exc:
            self.active -= 1
            stdout_path.write_bytes(b"")
            stderr_path.write_bytes(str(exc).encode("utf-8", errors="replace"))
            self.commands.append({
                "index": index, "argv": argv, "cwd": str(cwd),
                "durationSeconds": round(time.monotonic() - started, 3),
                "exitCode": None, "startError": f"{type(exc).__name__}: {exc}",
                "stdoutPath": str(stdout_path), "stderrPath": str(stderr_path),
                "stdoutSha256": sha256(b""), "stderrSha256": sha256(str(exc).encode("utf-8", errors="replace")),
            })
            raise
        timed_out = False
        tree_termination = "not-needed"
        post_kill_timeout = False
        try:
            try:
                stdout, stderr = process.communicate(input=input_bytes, timeout=timeout)
            except subprocess.TimeoutExpired as first_timeout:
                timed_out = True
                partial_stdout = first_timeout.output or b""
                partial_stderr = first_timeout.stderr or b""
                cleanup_deadline = min(self.deadline, time.monotonic() + 5.0)
                def kill_root_process() -> None:
                    try:
                        process.kill()
                    except OSError:
                        pass
                if os.name == "nt":
                    try:
                        killed = subprocess.run(
                            ["taskkill", "/PID", str(process.pid), "/T", "/F"],
                            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                            timeout=max(0.05, min(2.0, cleanup_deadline - time.monotonic())),
                            check=False, creationflags=subprocess.CREATE_NO_WINDOW,
                        )
                        if killed.returncode == 0:
                            tree_termination = "taskkill-tree-succeeded"
                        else:
                            tree_termination = f"taskkill-tree-failed-{killed.returncode}"
                            kill_root_process()
                    except (OSError, subprocess.TimeoutExpired) as exc:
                        tree_termination = f"taskkill-tree-error-{type(exc).__name__}"
                        kill_root_process()
                else:
                    try:
                        os.killpg(process.pid, signal.SIGKILL)
                        tree_termination = "kill-process-group-sent"
                    except ProcessLookupError:
                        tree_termination = "process-group-already-exited"
                    except OSError as exc:
                        tree_termination = f"kill-process-group-error-{type(exc).__name__}"
                        kill_root_process()
                try:
                    cleanup_timeout = max(0.05, cleanup_deadline - time.monotonic())
                    stdout, stderr = process.communicate(timeout=cleanup_timeout)
                except subprocess.TimeoutExpired as cleanup_timeout_error:
                    post_kill_timeout = True
                    stdout = cleanup_timeout_error.output or partial_stdout
                    stderr = cleanup_timeout_error.stderr or partial_stderr
                    kill_root_process()
                    process.poll()
                    if os.name == "nt":
                        tree_termination = "unknown-after-bounded-cleanup-timeout"
                    else:
                        tree_termination += ";wait-timed-out"
                stderr += b"\nchange_smoke: command exceeded its bounded timeout; see treeTermination for cleanup evidence\n"
        finally:
            self.active -= 1
        exit_code = 124 if timed_out else int(process.returncode)
        stdout_path.write_bytes(stdout)
        stderr_path.write_bytes(stderr)
        self.commands.append({
            "index": index,
            "argv": argv,
            "cwd": str(cwd),
            "startedMonotonic": round(started, 3),
            "durationSeconds": round(time.monotonic() - started, 3),
            "timeoutSeconds": round(timeout, 3),
            "timedOut": timed_out,
            "treeTermination": tree_termination,
            "postKillTimeout": post_kill_timeout,
            "exitCode": exit_code,
            "stdoutPath": str(stdout_path),
            "stderrPath": str(stderr_path),
            "stdoutSha256": hashlib.sha256(stdout).hexdigest(),
            "stderrSha256": hashlib.sha256(stderr).hexdigest(),
        })
        if timed_out:
            raise ChangeSmokeFailure(f"command exceeded {timeout:.1f}s; raw output preserved; tree cleanup={tree_termination}")
        return subprocess.CompletedProcess(argv, exit_code, stdout, stderr)


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def git(repo: Path, *args: str, input_bytes: bytes | None = None) -> bytes:
    return smoke.git(repo, *args, input_bytes=input_bytes)


def file_tree_hashes(root: Path) -> dict[str, str]:
    if not root.exists():
        return {}
    if not root.is_dir():
        return {root.name: sha256(root.read_bytes())}
    return {
        path.relative_to(root).as_posix(): sha256(path.read_bytes())
        for path in sorted(root.rglob("*")) if path.is_file()
    }


def run_step(trial: dict[str, Any], name: str, argv: list[str], cwd: Path,
             env: dict[str, str], expected_exit: int = 0) -> dict[str, Any]:
    return smoke.execute_step(trial, name, argv, cwd, env, expected_exit)


def preserve_plan(trial_root: Path, step: dict[str, Any], filename: str) -> Path:
    path = trial_root / filename
    path.write_bytes(Path(step["stdoutPath"]).read_bytes())
    return path


def review_minimum_two(run: dict[str, Any]) -> dict[str, bytes]:
    if run.get("status") != "planned" or run.get("proposal", {}).get("status") != "planned":
        raise ChangeSmokeFailure(f"fresh Execute was not planned: {run.get('status')!r}")
    expected_projections = {smoke.DOTNET_PROJECTION, smoke.MARKDOWN_PROJECTION}
    work = run.get("work")
    if not isinstance(work, list) or {item.get("projectionId") for item in work} != expected_projections:
        raise ChangeSmokeFailure("fresh Execute must contain both declared Commerce projections")
    planned_work = {
        item.get("projectionId")
        for item in run.get("proposal", {}).get("plan", {}).get("proposals", [])
        if item.get("decision") == "work"
    }
    if planned_work != expected_projections:
        raise ChangeSmokeFailure("changed intent must fan out to both .NET and Markdown projections")
    expected_dotnet = smoke.expected_candidate_files("positive")
    expected_dotnet["src/Commerce/CreateOrderHandler.cs"] = expected_dotnet[
        "src/Commerce/CreateOrderHandler.cs"
    ].replace(b"quantity <= 0", b"quantity < 2")
    outputs: dict[str, bytes] = {}
    markdown: bytes | None = None
    for item in work:
        projection = item.get("projectionId")
        encoded = item.get("outputs")
        if not isinstance(encoded, dict):
            raise ChangeSmokeFailure(f"projection {projection!r} returned no candidate output map")
        decoded: dict[str, bytes] = {}
        for path, content in encoded.items():
            if not isinstance(path, str) or not isinstance(content, str):
                raise ChangeSmokeFailure("candidate output map has an invalid shape")
            try:
                decoded[path] = base64.b64decode(content, validate=True)
            except Exception as exc:
                raise ChangeSmokeFailure(f"candidate output {path!r} is invalid base64") from exc
        if projection == smoke.DOTNET_PROJECTION:
            if decoded != expected_dotnet:
                raise ChangeSmokeFailure(".NET candidate differs from the fixed accepted min-two contract")
        elif projection == smoke.MARKDOWN_PROJECTION:
            if set(decoded) != {smoke.MARKDOWN_OUTPUT}:
                raise ChangeSmokeFailure("Markdown Module must produce only its declared projection path")
            markdown = decoded[smoke.MARKDOWN_OUTPUT]
            if b"quantity of at least two" not in markdown:
                raise ChangeSmokeFailure("Markdown projection does not reflect the accepted minimum-two intent")
        else:
            raise ChangeSmokeFailure(f"unexpected work projection: {projection!r}")
        for path, data in decoded.items():
            if path in outputs:
                raise ChangeSmokeFailure(f"multiple projections target {path}")
            outputs[path] = data
    if markdown is None or len(outputs) != 4:
        raise ChangeSmokeFailure("candidate must contain three .NET files and one Markdown file")
    return outputs


def validate_exact_source(repo: Path) -> tuple[bytes, bytes]:
    purpose_path = repo / USE_CASE_PATH
    program_path = repo / PROGRAM_PATH
    purpose = purpose_path.read_bytes()
    program = program_path.read_bytes()
    if purpose.count(BASELINE_PURPOSE.encode()) != 1 or program.count(BASELINE_PROBE.encode()) != 1:
        raise ChangeSmokeFailure("prepared baseline does not contain the exact known intent and one-item probe markers")
    if b"at least two" in purpose or b"one item below accepted minimum" in program:
        raise ChangeSmokeFailure("prepared baseline already contains the selected minimum-two change")
    return purpose, program


def accept_exact_intent_change(repo: Path, accept: bool, receipt: dict[str, Any]) -> str:
    purpose_path = repo / USE_CASE_PATH
    program_path = repo / PROGRAM_PATH
    purpose_before, program_before = validate_exact_source(repo)
    purpose_after = purpose_before.replace(BASELINE_PURPOSE.encode(), MINIMUM_TWO_PURPOSE.encode(), 1)
    program_after = program_before.replace(BASELINE_PROBE.encode(), MINIMUM_TWO_PROBE.encode(), 1)
    if purpose_after == purpose_before or program_after == program_before:
        raise ChangeSmokeFailure("known source edit did not produce both expected changes")
    diff_path = Path(receipt["outputRoot"]) / "accepted-intent.diff"
    preview = "".join(difflib.unified_diff(
        purpose_before.decode("utf-8").splitlines(keepends=True),
        purpose_after.decode("utf-8").splitlines(keepends=True),
        fromfile=USE_CASE_PATH + " (before)", tofile=USE_CASE_PATH + " (selected)",
    )) + "".join(difflib.unified_diff(
        program_before.decode("utf-8").splitlines(keepends=True),
        program_after.decode("utf-8").splitlines(keepends=True),
        fromfile=PROGRAM_PATH + " (before)", tofile=PROGRAM_PATH + " (selected)",
    ))
    diff_path.write_text(preview, encoding="utf-8", newline="")
    if not accept:
        raise ChangeSmokeFailure("review the exact two-file diff, then select it with --accept-minimum-two in a new output directory")
    purpose_path.write_bytes(purpose_after)
    program_path.write_bytes(program_after)
    git_diff = git(repo, "diff", "--", USE_CASE_PATH, PROGRAM_PATH)
    if not git_diff or set(
        line.decode("utf-8", errors="replace")
        for line in git(repo, "diff", "--name-only").splitlines()
    ) != {USE_CASE_PATH, PROGRAM_PATH}:
        raise ChangeSmokeFailure("intent diff is empty or includes paths outside the two reviewed inputs")
    diff_path.write_bytes(git_diff)
    receipt["intentChange"] = {
        "paths": [USE_CASE_PATH, PROGRAM_PATH],
        "oldPurposeSha256": sha256(purpose_before),
        "newPurposeSha256": sha256(purpose_after),
        "oldProbeSha256": sha256(program_before),
        "newProbeSha256": sha256(program_after),
        "diffPath": str(diff_path),
        "diffSha256": sha256(git_diff),
        "ownerSelectionFlag": "--accept-minimum-two",
        "selected": accept,
    }
    git(repo, "add", "--", USE_CASE_PATH, PROGRAM_PATH)
    staged = set(line.decode("utf-8", errors="strict") for line in git(repo, "diff", "--cached", "--name-only").splitlines())
    if staged != {USE_CASE_PATH, PROGRAM_PATH}:
        raise ChangeSmokeFailure("staged intent change differs from the exact two-path allowlist")
    committed = smoke.run_raw(
        ["git", "-c", f"safe.directory={repo.as_posix()}", "-C", str(repo), "commit", "-m", "accept minimum order quantity of two"],
        repo,
    )
    if committed.returncode != 0:
        raise ChangeSmokeFailure(f"could not commit exact accepted intent change: {committed.stderr.decode('utf-8', errors='replace')}")
    head = git(repo, "rev-parse", "HEAD").decode("ascii").strip()
    if git(repo, "diff", "--quiet", "HEAD", "--").strip() or git(repo, "diff", "--cached", "--quiet").strip():
        raise ChangeSmokeFailure("accepted intent commit left tracked or staged source changes")
    receipt["intentChange"]["newSourceRevision"] = head
    return head


def baseline_pass(args: argparse.Namespace, binary: Path, source_root: Path,
                  output: Path, receipt: dict[str, Any]) -> tuple[Path, str, dict[str, Any], dict[str, bytes], Path]:
    trial = smoke.run_trial(args, binary, source_root, "positive", output, receipt)
    trial["name"] = "baseline-positive"
    if trial.get("status") != "passed":
        raise ChangeSmokeFailure(f"baseline positive cycle stopped incomplete: {trial.get('failure', 'see preserved stage receipt')}")
    repo = Path(trial["repositoryPath"])
    revision = trial["sourceRevision"]
    trial_root = output / "positive"
    plan_path = trial_root / "controller-execute.json"
    apply_path = trial_root / "controller-apply.json"
    run = json.loads(plan_path.read_bytes())
    reviewed = smoke.verify_candidate_review("positive", run)
    applied = json.loads(apply_path.read_bytes())
    active_records = applied.get("records")
    if not isinstance(active_records, list) or len(active_records) != 2:
        raise ChangeSmokeFailure("baseline Apply did not return both active Commerce Projection records")
    active_record_ids = {item.get("projectionId") for item in active_records if isinstance(item, dict)}
    if active_record_ids != {smoke.DOTNET_PROJECTION, smoke.MARKDOWN_PROJECTION}:
        raise ChangeSmokeFailure("baseline Apply record set did not cover both Commerce projections")
    active_records_path = trial_root / "active-projection-records.json"
    active_records_path.write_text(json.dumps(active_records, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    trial["activeRecordsPath"] = str(active_records_path)
    trial["activeRecordsSha256"] = sha256(active_records_path.read_bytes())
    trial["recordStorePath"] = json.loads(Path(trial["runtimePath"]).read_bytes())["recordStore"]
    trial["planPath"] = str(plan_path)
    trial["applyPath"] = str(apply_path)
    trial["executorRoleInvocations"] = sum(1 for item in run.get("work", []) if item.get("executor"))
    verification = smoke.parse_json_output(next(step for step in trial["steps"] if step["name"] == "controller-verify-from-saved-apply"))
    trial["verifierRoleInvocations"] = len(verification.get("verifierRuns", []))
    if trial["executorRoleInvocations"] != 1 or trial["verifierRoleInvocations"] != 2:
        raise ChangeSmokeFailure("baseline role invocation count differed from the conservative reservation")
    return repo, revision, trial, reviewed, active_records_path


def changed_pass(args: argparse.Namespace, binary: Path, repo: Path, old_revision: str,
                 baseline: dict[str, Any], baseline_outputs: dict[str, bytes], active_records_path: Path,
                 trial_root: Path, receipt: dict[str, Any]) -> dict[str, Any]:
    trial_root.mkdir()
    logs = trial_root / "cli-logs"
    logs.mkdir()
    evidence = trial_root / "external" / "check-evidence"
    evidence.mkdir(parents=True)
    trial: dict[str, Any] = {
        "name": "accepted-minimum-two",
        "status": "incomplete",
        "steps": [],
        "logsPath": str(logs),
        "checkEvidencePath": str(evidence),
        "limitations": ["The protocol Verifier is synthetic; behavior is established only by the finite independent local check."],
    }
    receipt.setdefault("trials", []).append(trial)
    try:
        original_runtime_path = Path(baseline["runtimePath"])
        runtime = json.loads(original_runtime_path.read_bytes())
        trial["recordStorePath"] = runtime["recordStore"]
        if runtime.get("recordStore") != baseline.get("recordStorePath"):
            raise ChangeSmokeFailure("changed runtime does not retain baseline active-record ledger path")
        runtime_root = trial_root / "external"
        runtime_root.mkdir(exist_ok=True)
        runtime_path = runtime_root / "runtime.json"
        runner_path = str(repo / "examples/classic-commerce/tools/protocol_runner.py")
        runtime["executor"]["args"] = [runner_path, "--mode", "minimum-two"]
        runtime["verifier"]["args"] = [runner_path, "--mode", "minimum-two"]
        runtime_path.write_text(json.dumps(runtime, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        trial["runtimePath"] = str(runtime_path)
        trial["runtimeSha256"] = sha256(runtime_path.read_bytes())
        env = os.environ.copy()
        env["MARKITECT_CLASSIC_COMMERCE_EVIDENCE_DIR"] = str(evidence)
        source = ["--repo", str(repo), "--config", "examples/classic-commerce/canonical.yaml"]
        common = [*source, "--runtime", str(runtime_path)]
        baseline_common = [*source, "--runtime", str(original_runtime_path)]
        revision = git(repo, "rev-parse", "HEAD").decode("ascii").strip()
        trial["oldSourceRevision"] = old_revision
        trial["sourceRevision"] = revision
        record_store_root = Path(runtime["recordStore"])
        before_reconcile_ledger = file_tree_hashes(record_store_root)
        before_reconcile_targets = {path: sha256(repo.joinpath(*path.split("/")).read_bytes()) for path in baseline_outputs}

        impact = run_step(trial, "changed-impact", [str(binary), "canonical", *source, "--action", "impact", "--base", old_revision, "--revision", revision], repo, env)
        trial["impactOutputSha256"] = sha256(Path(impact["stdoutPath"]).read_bytes())
        reconcile = run_step(trial, "changed-reconcile-plan", [str(binary), "canonical", *source, "--action", "reconcile-plan", "--base", old_revision, "--revision", revision, "--evidence", str(active_records_path)], repo, env)
        reconcile_bytes = Path(reconcile["stdoutPath"]).read_bytes()
        if b"status: planned" not in reconcile_bytes.splitlines():
            raise ChangeSmokeFailure("read-only changed-source reconcile plan was not planned")
        trial["reconcilePlanOutputSha256"] = sha256(reconcile_bytes)
        trial["reconcilePlanFormat"] = "captured YAML; no general YAML parser used"

        audit = run_step(trial, "stale-audit", [str(binary), "canonical", *baseline_common, "--action", "controller-audit", "--base", old_revision, "--revision", revision], repo, env, expected_exit=2)
        stale_audit = smoke.parse_json_output(audit)
        if stale_audit.get("status") != "incomplete":
            raise ChangeSmokeFailure("audit of old materialization under changed intent did not report incomplete")
        trial["staleAuditStatus"] = stale_audit.get("status")
        trial["staleAuditFindings"] = stale_audit.get("findings", [])
        if file_tree_hashes(record_store_root) != before_reconcile_ledger:
            raise ChangeSmokeFailure("read-only impact/reconcile/audit phase changed the retained record-store bytes")
        if {path: sha256(repo.joinpath(*path.split("/")).read_bytes()) for path in baseline_outputs} != before_reconcile_targets:
            raise ChangeSmokeFailure("read-only impact/reconcile/audit phase changed baseline target bytes")

        old_plan_path = Path(receipt["trials"][0]["planPath"])
        pre_refusal_head = git(repo, "rev-parse", "HEAD").decode("ascii").strip()
        pre_refusal_tree = git(repo, "write-tree").decode("ascii").strip()
        pre_refusal_status = git(repo, "status", "--porcelain=v1", "--untracked-files=all").decode("utf-8", errors="replace")
        pre_refusal_ledger = file_tree_hashes(record_store_root)
        pre_refusal_targets = {path: sha256(repo.joinpath(*path.split("/")).read_bytes()) for path in baseline_outputs}
        old_plan = json.loads(old_plan_path.read_bytes())
        refuse = run_step(trial, "refuse-stale-apply", [str(binary), "canonical", *baseline_common, "--action", "controller-apply", "--base", old_revision, "--revision", revision, "--plan", str(old_plan_path), "--expect", str(old_plan.get("digest", "")), "--write"], repo, env, expected_exit=2)
        refusal_text = Path(refuse["stderrPath"]).read_bytes().decode("utf-8", errors="replace")
        if "--base and --revision must exactly match the saved reviewed run" not in refusal_text:
            raise ChangeSmokeFailure("old reviewed Apply was not refused for its source binding")
        if git(repo, "rev-parse", "HEAD").decode("ascii").strip() != pre_refusal_head or git(repo, "write-tree").decode("ascii").strip() != pre_refusal_tree:
            raise ChangeSmokeFailure("stale Apply refusal changed HEAD or index")
        if git(repo, "status", "--porcelain=v1", "--untracked-files=all").decode("utf-8", errors="replace") != pre_refusal_status:
            raise ChangeSmokeFailure("stale Apply refusal changed source or materialized target paths")
        if file_tree_hashes(record_store_root) != pre_refusal_ledger:
            raise ChangeSmokeFailure("stale Apply refusal changed the record-store bytes")
        if {path: sha256(repo.joinpath(*path.split("/")).read_bytes()) for path in baseline_outputs} != pre_refusal_targets:
            raise ChangeSmokeFailure("stale Apply refusal changed materialized target bytes")
        trial["staleApplyRefusal"] = "exact saved-run source-binding error; HEAD/index/worktree unchanged"
        trial["staleRefusalLedgerSha256"] = pre_refusal_ledger
        trial["staleRefusalTargetSha256"] = pre_refusal_targets

        proposal_step = run_step(trial, "changed-propose", [str(binary), "canonical", *common, "--action", "controller-propose", "--base", old_revision, "--revision", revision], repo, env)
        proposal = smoke.parse_json_output(proposal_step)
        if proposal.get("status") != "planned":
            raise ChangeSmokeFailure("fresh changed-intent proposal is not planned")
        trial["proposalDigest"] = proposal.get("digest")
        execute = run_step(trial, "changed-execute", [str(binary), "canonical", *common, "--action", "controller-execute", "--base", old_revision, "--revision", revision], repo, env)
        execute_bytes = Path(execute["stdoutPath"]).read_bytes()
        plan_path = trial_root / "controller-execute.json"
        plan_path.write_bytes(execute_bytes)
        run = json.loads(execute_bytes)
        reviewed = review_minimum_two(run)
        expected_changed = {path for path, data in reviewed.items() if baseline_outputs[path] != data}
        if expected_changed != {"src/Commerce/CreateOrderHandler.cs", smoke.MARKDOWN_OUTPUT}:
            raise ChangeSmokeFailure(f"min-two intent must change only the handler and Markdown outputs, got {sorted(expected_changed)}")
        initial_tree = git(repo, "write-tree").decode("ascii").strip()
        trial["executeDigest"] = run.get("digest")
        trial["reviewedOutputs"] = {path: sha256(data) for path, data in sorted(reviewed.items())}
        trial["reviewApproval"] = "explicit --approve-known-fixture; exact candidate paths and bytes checked by this driver"
        if not args.approve_known_fixture:
            raise ChangeSmokeFailure("fresh Apply requires --approve-known-fixture")
        apply = run_step(trial, "changed-apply", [str(binary), "canonical", *common, "--action", "controller-apply", "--base", old_revision, "--revision", revision, "--plan", str(plan_path), "--expect", str(run.get("digest", "")), "--write"], repo, env)
        apply_bytes = Path(apply["stdoutPath"]).read_bytes()
        apply_path = trial_root / "controller-apply.json"
        apply_path.write_bytes(apply_bytes)
        applied = json.loads(apply_bytes)
        if applied.get("status") != "materialized-unverified":
            raise ChangeSmokeFailure("changed Apply did not return materialized-unverified")
        smoke.assert_git_after_apply(repo, revision, reviewed, applied, initial_tree, expected_written=expected_changed)
        verify = run_step(trial, "changed-verify", [str(binary), "canonical", *common, "--action", "controller-verify", "--apply-result", str(apply_path), "--write"], repo, env)
        verification = smoke.parse_json_output(verify)
        if verification.get("status") != "passed" or not verification.get("results") or any(item.get("outcome") != "passed" for item in verification["results"]):
            raise ChangeSmokeFailure("fresh min-two verification did not pass every assurance scope")
        trial["verificationStatus"] = verification.get("status")
        trial["executorRoleInvocations"] = sum(1 for item in run.get("work", []) if item.get("executor"))
        trial["verifierRoleInvocations"] = len(verification.get("verifierRuns", []))
        if trial["executorRoleInvocations"] != 1 or trial["verifierRoleInvocations"] != 2:
            raise ChangeSmokeFailure("changed role invocation count differed from the conservative reservation")
        trial["checkReceipts"] = smoke.check_receipts(evidence, "positive")
        trial["checkEvidenceFiles"] = {
            path.relative_to(evidence).as_posix(): sha256(path.read_bytes())
            for path in sorted(evidence.rglob("*")) if path.is_file()
        }
        audit_step = run_step(trial, "changed-audit", [str(binary), "canonical", *common, "--action", "controller-audit", "--base", old_revision, "--revision", revision], repo, env)
        audit_report = smoke.parse_json_output(audit_step)
        if audit_report.get("status") != "complete":
            raise ChangeSmokeFailure("changed-intent audit is not complete")
        trial["auditStatus"] = audit_report.get("status")
        trial["auditDigest"] = audit_report.get("digest")
        smoke.assert_git_after_apply(repo, revision, reviewed, applied, initial_tree, expected_written=expected_changed)
        trial["status"] = "passed"
        return trial
    except Exception as exc:
        trial["failure"] = f"{type(exc).__name__}: {exc}"
        raise


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, help="published native Markitect v0.14.1 executable")
    parser.add_argument("--source-root", required=True, help="clean Git checkout containing the Classic Commerce example")
    parser.add_argument("--source-revision", required=True, help="full source commit ID used to prepare both disposable passes")
    parser.add_argument("--output", required=True, help="absent external directory for all receipts and disposable repos")
    parser.add_argument("--approve-known-fixture", action="store_true", help="permit exact checked candidates in disposable repositories")
    parser.add_argument("--accept-minimum-two", action="store_true", help="select the one displayed, exact two-file synthetic intent change")
    args = parser.parse_args()
    if not args.approve_known_fixture or not args.accept_minimum_two:
        parser.error("this fixed two-pass example requires --approve-known-fixture and --accept-minimum-two")
    source_root = Path(args.source_root).expanduser().resolve(strict=True)
    binary = Path(args.binary).expanduser().resolve(strict=True)
    output = Path(args.output).expanduser().resolve(strict=False)
    if len(args.source_revision) != 40 or any(char not in "0123456789abcdef" for char in args.source_revision.lower()):
        parser.error("--source-revision must be a full 40-character commit ID")
    if output.exists() or not output.parent.is_dir():
        parser.error("--output must be an absent directory under an existing parent")
    if smoke.within(output, source_root) or smoke.within(output, binary.parent):
        parser.error("--output must be outside source and binary directories")
    expected_hash = smoke.PUBLISHED_BINARY_SHA256.get(platform.system())
    if not expected_hash or sha256(binary.read_bytes()) != expected_hash:
        parser.error("--binary must match the pinned published v0.14.1 asset for this platform")

    receipt: dict[str, Any] | None = None
    receipt_path: Path | None = None
    scratch = Path(tempfile.mkdtemp(prefix="markitect-classic-change-preflight-"))
    budget = ProcessBudget(scratch, time.monotonic() + MAX_WALL_SECONDS)
    preflight_stage = "source identity"
    original_run_raw = smoke.run_raw
    smoke.run_raw = budget
    try:
        source_head = smoke.git(source_root, "rev-parse", "HEAD").decode("ascii").strip().lower()
        if source_head != args.source_revision.lower():
            raise ChangeSmokeFailure("source HEAD does not equal --source-revision")
        preflight_stage = "source cleanliness"
        if smoke.git(source_root, "status", "--porcelain=v1", "--untracked-files=all").strip():
            raise ChangeSmokeFailure("--source-root must have a clean tracked and untracked worktree")
        preflight_stage = "fixture source markers"
        source_entries = dict((path, data) for path, data, _ in smoke.source_files_at_revision(source_root, args.source_revision))
        if source_entries.get(USE_CASE_PATH, b"").count(BASELINE_PURPOSE.encode()) != 1 or source_entries.get(PROGRAM_PATH, b"").count(BASELINE_PROBE.encode()) != 1:
            raise ChangeSmokeFailure("source revision lacks the required known intent and check markers; marker checks only bound the two-file edit")
        preflight_stage = "output location"
        listing = smoke.git(source_root, "worktree", "list", "--porcelain").decode("utf-8", errors="strict")
        roots = [Path(line.removeprefix("worktree ")).resolve() for line in listing.splitlines() if line.startswith("worktree ")]
        if any(smoke.within(output, root) for root in roots):
            raise ChangeSmokeFailure("--output must be outside every worktree of the source repository")
        preflight_stage = "published binary identity"
        version = budget([str(binary), "version"], binary.parent)
        if version.returncode != 0 or b"0.14.1" not in version.stdout + version.stderr:
            raise ChangeSmokeFailure("pinned binary did not report version 0.14.1")
        output.mkdir()
        old_logs = budget.logs
        budget.logs = output / "process-logs"
        shutil.move(str(old_logs), str(budget.logs))
        budget.output = output
        for command in budget.commands:
            for name in ("stdoutPath", "stderrPath"):
                if name in command:
                    command[name] = str(budget.logs / Path(command[name]).name)
        receipt = {
            "schema": "markitect.classic-commerce-intent-change-smoke/v1",
            "status": "incomplete",
            "sourceRevision": source_head,
            "sourceSelection": "operator-selected clean source revision; only required known markers are validated",
            "outputRoot": str(output),
            "preflightScratch": str(scratch),
            "binary": {"path": str(binary), "sha256": sha256(binary.read_bytes())},
            "binaryVersionSha256": sha256(version.stdout + version.stderr),
            "platform": platform.platform(),
            "python": platform.python_version(),
            "workflowCycleLimit": 2,
            "workflowCyclesCompleted": 0,
            "workflowAttempts": 1,
            "automaticRetries": 0,
            "nativeMutatingStartLimit": MAX_NATIVE_MUTATING_STARTS,
            "deterministicRoleReservationLimit": MAX_DETERMINISTIC_ROLE_RESERVATIONS,
            "parallelismLimit": 2,
            "limitations": [
                "The operator selects and trusts a clean source revision; fixed marker checks only bound the named two-file intent edit and do not validate arbitrary fixture modules or checks.",
                "Executor and Verifier are deterministic protocol test doubles; verifier observations are not semantic review or human acceptance.",
                "Fixed behavior evidence covers only the declared finite cases; this is not autonomous-operation or provider-quality evidence.",
            ],
            "trials": [],
        }
        receipt_path = output / "receipt.json"
    except Exception as exc:
        failure_receipt = {
            "schema": "markitect.classic-commerce-intent-change-smoke/preflight-v1",
            "status": "incomplete", "stage": preflight_stage,
            "failure": f"{type(exc).__name__}: {exc}", "commands": budget.commands,
        }
        preflight_receipt = scratch / "preflight-receipt.json"
        preflight_receipt.write_text(json.dumps(failure_receipt, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        smoke.run_raw = original_run_raw
        parser.error(f"bounded preflight failed at {preflight_stage}; command logs and receipt retained at {scratch}")
    finally:
        smoke.run_raw = original_run_raw

    # Continue with the same 1800-second deadline and role/start reservations.
    smoke.run_raw = budget
    try:
        assert receipt is not None and receipt_path is not None
        repo, baseline_revision, baseline, baseline_outputs, active_records_path = baseline_pass(args, binary, source_root, output, receipt)
        receipt["workflowCyclesCompleted"] = 1
        receipt["baselineSourceRevision"] = baseline_revision
        receipt["baselineAuditStatus"] = "complete"

        new_revision = accept_exact_intent_change(repo, args.accept_minimum_two, receipt)
        for path, original_bytes in baseline_outputs.items():
            if repo.joinpath(*path.split("/")).read_bytes() != original_bytes:
                raise ChangeSmokeFailure(f"baseline materialization changed during intent-only commit: {path}")
        receipt["retainedBaselineArtifacts"] = {path: sha256(data) for path, data in sorted(baseline_outputs.items())}
        trial = changed_pass(args, binary, repo, baseline_revision, baseline, baseline_outputs, active_records_path, output / "minimum-two", receipt)
        receipt["workflowCyclesCompleted"] = 2
        receipt["changedSourceRevision"] = new_revision
        receipt["changedAuditStatus"] = trial.get("auditStatus")
        receipt["status"] = "passed" if baseline.get("status") == "passed" and trial.get("status") == "passed" else "incomplete"
    except Exception as exc:
        assert receipt is not None
        receipt["failure"] = f"{type(exc).__name__}: {exc}"
        receipt["status"] = "incomplete"
    finally:
        smoke.run_raw = original_run_raw
        if receipt is not None and receipt_path is not None:
            receipt["nativeMutatingStarts"] = budget.native_mutating_starts
            receipt["deterministicRoleReservations"] = budget.role_reservations
            receipt["maximumConcurrentProcesses"] = budget.max_active
            receipt["commands"] = budget.commands
            receipt["receiptSha256Note"] = "receipt.json is a mutable final receipt; verify its bytes after completion, not via an embedded self-hash"
            receipt_path.write_text(json.dumps(receipt, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    if receipt is None or receipt_path is None:
        return 2
    print(json.dumps({"status": receipt["status"], "receiptPath": str(receipt_path)}, separators=(",", ":")))
    return 0 if receipt["status"] == "passed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
