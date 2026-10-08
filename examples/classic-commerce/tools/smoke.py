#!/usr/bin/env python3
"""Run bounded Classic Commerce checks through an installed Markitect CLI.

This is an example-specific local smoke, not a provider adapter or semantic
acceptance system. `--approve-known-fixture` authorizes only this driver's
review of the fixed candidate paths and bytes below.
"""
from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
import platform
import subprocess
import sys
import time
from pathlib import Path
from typing import Any

API_VERSION = "markitect.canonical/controller/v1alpha1"
DOTNET_PROJECTION = '["markitect.foundation/v1","Projection","commerce","application-dotnet"]'
MARKDOWN_PROJECTION = '["markitect.foundation/v1","Projection","commerce","application-markdown"]'
MARKDOWN_OUTPUT = "docs/represented/index.md"
CHECK_ID = "classic-commerce-build-and-behavior"
CHECK_INPUTS = [
    "examples/classic-commerce/global.json",
    "examples/classic-commerce/NuGet.Config",
    "examples/classic-commerce/checks/ClassicCommerce.Check.csproj",
    "examples/classic-commerce/checks/Program.cs",
    "examples/classic-commerce/checks/check.py",
]
PUBLISHED_BINARY_SHA256 = {
    "Windows": "2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4",
    "Linux": "f515f8fb37335ff5bd36f77cbbd0af227c3558d0d5aa8ba80503bae40314fbdc",
}


class SmokeFailure(RuntimeError):
    pass


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def within(path: Path, root: Path) -> bool:
    try:
        path.relative_to(root)
        return True
    except ValueError:
        return False


def run_raw(argv: list[str], cwd: Path, env: dict[str, str] | None = None, input_bytes: bytes | None = None) -> subprocess.CompletedProcess[bytes]:
    return subprocess.run(
        argv,
        cwd=cwd,
        env=env,
        input=input_bytes,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
        shell=False,
    )


def git(repo: Path, *args: str, input_bytes: bytes | None = None) -> bytes:
    argv = ["git", "-c", f"safe.directory={repo.as_posix()}", "-C", str(repo), *args]
    result = run_raw(argv, repo, input_bytes=input_bytes)
    if result.returncode != 0:
        raise SmokeFailure(f"git {' '.join(args)} failed ({result.returncode}): {result.stderr.decode('utf-8', errors='replace')}")
    return result.stdout


def source_files_at_revision(source_root: Path, revision: str) -> list[tuple[str, bytes, str]]:
    listing = git(source_root, "ls-tree", "-rz", "--full-tree", revision, "--", "examples/classic-commerce")
    entries: list[tuple[str, bytes, str]] = []
    for item in listing.split(b"\0"):
        if not item:
            continue
        metadata, raw_path = item.split(b"\t", 1)
        mode, object_type, object_id = metadata.decode("ascii").split(" ", 2)
        path = raw_path.decode("utf-8")
        if object_type != "blob" or mode != "100644" or not path.startswith("examples/classic-commerce/"):
            raise SmokeFailure(f"unsupported tracked fixture entry: {path}")
        data = git(source_root, "show", f"{revision}:{path}")
        actual_object = git(source_root, "hash-object", "--stdin", input_bytes=data).decode("ascii").strip()
        if actual_object != object_id:
            raise SmokeFailure(f"Git blob bytes changed while copying {path}")
        entries.append((path, data, object_id))
    required = {"examples/classic-commerce/canonical.yaml", *(CHECK_INPUTS)}
    present = {path for path, _, _ in entries}
    if not required.issubset(present):
        raise SmokeFailure(f"source revision lacks required fixture inputs: {sorted(required - present)}")
    return sorted(entries, key=lambda entry: entry[0])


def prepare_repository(source_root: Path, revision: str, trial_root: Path) -> tuple[Path, str, dict[str, str]]:
    repo = trial_root / "repo"
    repo.mkdir()
    files = source_files_at_revision(source_root, revision)
    copied: dict[str, str] = {}
    for relative, data, blob in files:
        target = repo.joinpath(*relative.split("/"))
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)
        copied[relative] = sha256(data)
        if target.read_bytes() != data:
            raise SmokeFailure(f"copied fixture bytes differ for {relative}")
    init = run_raw(["git", "init", "--initial-branch=codex/classic-commerce-smoke", str(repo)], trial_root)
    if init.returncode != 0:
        raise SmokeFailure(f"git init failed: {init.stderr.decode('utf-8', errors='replace')}")
    for key, value in (("core.autocrlf", "false"), ("user.name", "Markitect Example Smoke"), ("user.email", "smoke@example.invalid")):
        git(repo, "config", key, value)
    git(repo, "add", "--all")
    env = os.environ.copy()
    env.update({
        "GIT_AUTHOR_NAME": "Markitect Example Smoke",
        "GIT_AUTHOR_EMAIL": "smoke@example.invalid",
        "GIT_COMMITTER_NAME": "Markitect Example Smoke",
        "GIT_COMMITTER_EMAIL": "smoke@example.invalid",
        "GIT_AUTHOR_DATE": "2000-01-01T00:00:00Z",
        "GIT_COMMITTER_DATE": "2000-01-01T00:00:00Z",
    })
    commit = run_raw(["git", "-c", f"safe.directory={repo.as_posix()}", "-C", str(repo), "commit", "-m", "fixed Classic Commerce smoke inputs"], repo, env=env)
    if commit.returncode != 0:
        raise SmokeFailure(f"fixture commit failed: {commit.stderr.decode('utf-8', errors='replace')}")
    head = git(repo, "rev-parse", "HEAD").decode("ascii").strip()
    if git(repo, "status", "--porcelain=v1", "--untracked-files=all").strip():
        raise SmokeFailure("prepared fixture repository is not clean")
    tracked = git(repo, "ls-files", "-z").split(b"\0")
    tracked_paths = {value.decode("utf-8") for value in tracked if value}
    if tracked_paths != set(copied):
        raise SmokeFailure("prepared repository contains an unexpected tracked file set")
    return repo, head, copied


def expected_candidate_files(mode: str) -> dict[str, bytes]:
    handler = (
        "using System;\n\n"
        "namespace Commerce.Application;\n\n"
        "public sealed class CreateOrderHandler\n"
        "{\n"
        "    public decimal Handle(int quantity, decimal unitPrice)\n"
        "    {\n"
        "        if (quantity <= 0) throw new ArgumentOutOfRangeException(nameof(quantity));\n"
        "        if (unitPrice < 0m) throw new ArgumentOutOfRangeException(nameof(unitPrice));\n"
        + ("        return quantity + unitPrice;\n" if mode == "bad-business" else "        return quantity * unitPrice;\n")
        + "    }\n"
        "}\n"
    )
    return {
        "src/Commerce/Commerce.csproj": (
            '<Project Sdk="Microsoft.NET.Sdk">\n'
            "  <PropertyGroup>\n"
            "    <TargetFramework>net8.0</TargetFramework>\n"
            "    <ImplicitUsings>enable</ImplicitUsings>\n"
            "    <Nullable>enable</Nullable>\n"
            "  </PropertyGroup>\n"
            "</Project>\n"
        ).encode(),
        "src/Commerce/CreateOrderHandler.cs": handler.encode(),
        "src/Commerce/EffectAxis.cs": b'namespace Commerce.Domain;\n\npublic sealed record EffectAxis(string Boundary = "application");\n',
    }


def parse_json_output(step: dict[str, Any]) -> dict[str, Any]:
    try:
        value = json.loads(Path(step["stdoutPath"]).read_bytes())
    except Exception as exc:
        raise SmokeFailure(f"{step['name']} did not emit one JSON report: {exc}") from exc
    if not isinstance(value, dict):
        raise SmokeFailure(f"{step['name']} JSON report is not an object")
    return value


def execute_step(trial: dict[str, Any], name: str, argv: list[str], cwd: Path, env: dict[str, str], expected_exit: int = 0) -> dict[str, Any]:
    index = len(trial["steps"]) + 1
    logs = Path(trial["logsPath"])
    stdout_path = logs / f"{index:02d}-{name}.stdout.bin"
    stderr_path = logs / f"{index:02d}-{name}.stderr.bin"
    started = time.monotonic()
    failure = None
    try:
        completed = run_raw(argv, cwd, env=env)
        stdout, stderr = completed.stdout, completed.stderr
        exit_code: int | None = completed.returncode
    except OSError as exc:
        stdout, stderr, exit_code = b"", str(exc).encode("utf-8", errors="replace"), None
        failure = f"could not start command: {type(exc).__name__}: {exc}"
    stdout_path.write_bytes(stdout)
    stderr_path.write_bytes(stderr)
    step = {
        "name": name,
        "argv": argv,
        "cwd": str(cwd),
        "expectedExitCode": expected_exit,
        "exitCode": exit_code,
        "stdoutPath": str(stdout_path),
        "stderrPath": str(stderr_path),
        "stdoutSha256": sha256(stdout),
        "stderrSha256": sha256(stderr),
        "durationSeconds": round(time.monotonic() - started, 3),
        "status": "passed" if exit_code == expected_exit and failure is None else "failed",
    }
    if failure:
        step["failure"] = failure
    trial["steps"].append(step)
    if step["status"] != "passed":
        raise SmokeFailure(f"{name}: expected exit {expected_exit}, got {exit_code}; see captured logs")
    return step


def build_runtime(repo: Path, trial_root: Path, mode: str) -> tuple[Path, dict[str, Any]]:
    external = trial_root / "external"
    external.mkdir(exist_ok=True)
    runner = repo / "examples/classic-commerce/tools/protocol_runner.py"
    if not runner.is_file():
        raise SmokeFailure("protocol runner is absent from the prepared source revision")
    runner_digest = "sha256:" + sha256(runner.read_bytes())
    check = {"name": CHECK_ID, "run": ["python", "examples/classic-commerce/checks/check.py"]}
    scope_check_inputs = list(CHECK_INPUTS)
    runtime = {
        "apiVersion": API_VERSION,
        "recordStore": str(external / "record-store"),
        "privateLogs": str(external / "private-logs"),
        "referenceDepth": 1,
        "auditAll": True,
        "checkInputs": list(CHECK_INPUTS),
        "executor": {
            "command": sys.executable,
            "args": [str(runner), "--mode", mode],
            "model": "classic-commerce-protocol-test-double",
            "modelOptions": {"claim": "fixed protocol output only; no semantic judgment"},
            "providerVersion": "example-test-double/1",
            "timeoutSeconds": 20,
            "maxStdoutBytes": 4194304,
            "maxStderrBytes": 1048576,
            "runtimeFiles": [{"path": str(runner), "mode": "0644", "digest": runner_digest}],
        },
        "verifier": {
            "command": sys.executable,
            "args": [str(runner), "--mode", mode],
            "model": "classic-commerce-protocol-test-double",
            "modelOptions": {"claim": "synthetic response coverage only; no semantic verification"},
            "providerVersion": "example-test-double/1",
            "timeoutSeconds": 20,
            "maxStdoutBytes": 4194304,
            "maxStderrBytes": 1048576,
            "runtimeFiles": [{"path": str(runner), "mode": "0644", "digest": runner_digest}],
        },
        "assuranceRoots": ["commerce-markdown"],
        "assuranceScopes": [
            {
                "id": "commerce-markdown",
                "projectionId": MARKDOWN_PROJECTION,
                "children": ["commerce-dotnet"],
                "checks": [check],
                "checkInputs": scope_check_inputs,
            },
            {
                "id": "commerce-dotnet",
                "projectionId": DOTNET_PROJECTION,
                "children": [],
                "checks": [check],
                "checkInputs": scope_check_inputs,
            },
        ],
    }
    path = external / "runtime.json"
    path.write_text(json.dumps(runtime, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return path, runtime


def verify_candidate_review(mode: str, run: dict[str, Any]) -> dict[str, bytes]:
    if run.get("status") != "planned" or run.get("proposal", {}).get("status") != "planned":
        raise SmokeFailure(f"Execute did not return a planned reviewed candidate: {run.get('status')}")
    work = run.get("work")
    if not isinstance(work, list) or {item.get("projectionId") for item in work} != {DOTNET_PROJECTION, MARKDOWN_PROJECTION}:
        raise SmokeFailure("Execute work set does not contain exactly the two declared Commerce projections")
    proposal_work = {
        item.get("projectionId")
        for item in run.get("proposal", {}).get("plan", {}).get("proposals", [])
        if item.get("decision") == "work"
    }
    if proposal_work != {DOTNET_PROJECTION, MARKDOWN_PROJECTION}:
        raise SmokeFailure("proposal does not declare work for exactly the two expected projections")
    expected = expected_candidate_files(mode)
    outputs: dict[str, bytes] = {}
    markdown: list[str] = []
    for item in work:
        projection = item["projectionId"]
        encoded = item.get("outputs")
        if not isinstance(encoded, dict):
            raise SmokeFailure(f"work for {projection} contains no exact output map")
        decoded: dict[str, bytes] = {}
        for path, content in encoded.items():
            if not isinstance(path, str) or not isinstance(content, str):
                raise SmokeFailure("candidate output map has an invalid shape")
            decoded[path] = base64.b64decode(content, validate=True)
        if projection == DOTNET_PROJECTION:
            if decoded != expected:
                raise SmokeFailure(".NET candidate differs from the predeclared fixed example bytes")
        else:
            if set(decoded) != {MARKDOWN_OUTPUT}:
                raise SmokeFailure("Markdown projection did not produce only represented Markdown files")
            markdown.extend(decoded)
        for path, data in decoded.items():
            if path in outputs:
                raise SmokeFailure(f"multiple projections target {path}")
            outputs[path] = data
    if len(markdown) != 1 or len(outputs) != len(expected) + 1:
        raise SmokeFailure("candidate output set differs from the three .NET files plus one deterministic Markdown file")
    return outputs


def check_receipts(parent: Path, mode: str) -> list[dict[str, Any]]:
    receipts = sorted(parent.glob("classic-commerce-check-*/result.json"))
    decoded: list[dict[str, Any]] = []
    for path in receipts:
        try:
            value = json.loads(path.read_bytes())
        except Exception as exc:
            raise SmokeFailure(f"fixed-check receipt is unreadable: {path.name}: {exc}") from exc
        value["publicRelativePath"] = path.relative_to(parent).as_posix()
        decoded.append(value)
    if not decoded:
        raise SmokeFailure("fixed check produced no external check receipts")
    if mode == "positive":
        if any(item.get("status") != "passed" or any(step.get("status") != "passed" for step in item.get("steps", [])) for item in decoded):
            raise SmokeFailure("positive fixture did not pass every captured build and behavior check")
    else:
        functional_failure = False
        for item in decoded:
            steps = item.get("steps", [])
            names = {step.get("name"): step.get("status") for step in steps}
            if (
                item.get("status") == "failed"
                and item.get("failedStep") == "run-behavior-probe"
                and all(names.get(name) == "passed" for name in ("sdk-version", "restore-candidate", "build-candidate", "restore-probe", "build-probe"))
                and names.get("run-behavior-probe") == "failed"
            ):
                functional_failure = True
        if not functional_failure:
            raise SmokeFailure("bad-business fixture did not compile and fail specifically in the independent behavior probe")
    return decoded


def assert_git_after_apply(repo: Path, source_revision: str, reviewed: dict[str, bytes], apply: dict[str, Any], initial_index_tree: str, expected_written: set[str] | None = None) -> None:
    if git(repo, "rev-parse", "HEAD").decode("ascii").strip() != source_revision:
        raise SmokeFailure("controller changed fixture HEAD")
    if git(repo, "diff", "--quiet", "HEAD", "--").strip() or git(repo, "diff", "--cached", "--quiet").strip():
        raise SmokeFailure("controller changed tracked source or the Git index")
    current_index_tree = git(repo, "write-tree").decode("ascii").strip()
    if current_index_tree != initial_index_tree:
        raise SmokeFailure("controller changed the staged Git tree")
    untracked = {
        value.decode("utf-8").replace("\\", "/")
        for value in git(repo, "ls-files", "--others", "--exclude-standard", "-z").split(b"\0") if value
    }
    if untracked != set(reviewed):
        raise SmokeFailure(f"post-Apply untracked outputs differ from reviewed candidate: {sorted(untracked ^ set(reviewed))}")
    expected_paths = set(reviewed) if expected_written is None else expected_written
    if not expected_paths.issubset(reviewed):
        raise SmokeFailure("expected Apply write set contains paths outside the reviewed candidate")
    written = apply.get("written")
    if not isinstance(written, list) or set(written) != expected_paths or len(written) != len(expected_paths):
        raise SmokeFailure("Apply written-path list differs from the expected reviewed write set")
    for path, expected in reviewed.items():
        actual = repo.joinpath(*path.split("/")).read_bytes()
        if actual != expected:
            raise SmokeFailure(f"materialized output differs from reviewed bytes: {path}")


def run_trial(args: argparse.Namespace, binary: Path, source_root: Path, mode: str, output_root: Path, receipt: dict[str, Any]) -> dict[str, Any]:
    trial_root = output_root / mode
    trial_root.mkdir()
    logs = trial_root / "cli-logs"
    logs.mkdir()
    evidence = trial_root / "external" / "check-evidence"
    evidence.mkdir(parents=True)
    trial: dict[str, Any] = {
        "mode": mode,
        "status": "incomplete",
        "steps": [],
        "logsPath": str(logs),
        "checkEvidencePath": str(evidence),
        "limitations": ["The Executor and Verifier are deterministic protocol test doubles; Verifier observations do not establish semantic correctness or human acceptance."],
    }
    receipt["trials"].append(trial)
    try:
        repo, source_revision, copied = prepare_repository(source_root, args.source_revision, trial_root)
        trial["repositoryPath"] = str(repo)
        trial["sourceRevision"] = source_revision
        trial["preparedSourceSha256"] = copied
        trial["initialIndexTree"] = git(repo, "write-tree").decode("ascii").strip()
        runtime_path, runtime = build_runtime(repo, trial_root, mode)
        trial["runtimePath"] = str(runtime_path)
        trial["runtimeSha256"] = sha256(runtime_path.read_bytes())
        env = os.environ.copy()
        env["MARKITECT_CLASSIC_COMMERCE_EVIDENCE_DIR"] = str(evidence)
        source_common = ["--repo", str(repo), "--config", "examples/classic-commerce/canonical.yaml"]
        common = [*source_common, "--runtime", str(runtime_path)]
        revision = source_revision

        model_step = execute_step(trial, "canonical-model", [str(binary), "canonical", *source_common, "--action", "model", "--revision", revision], repo, env)
        model_bytes = Path(model_step["stdoutPath"]).read_bytes()
        if not model_bytes.strip():
            raise SmokeFailure("canonical model preflight emitted no report")
        trial["modelOutputSha256"] = sha256(model_bytes)
        impact_step = execute_step(trial, "canonical-impact", [str(binary), "canonical", *source_common, "--action", "impact", "--base", revision, "--revision", revision], repo, env)
        impact_bytes = Path(impact_step["stdoutPath"]).read_bytes()
        if not impact_bytes.strip():
            raise SmokeFailure("canonical impact preflight emitted no report")
        trial["impactOutputSha256"] = sha256(impact_bytes)

        proposal_step = execute_step(trial, "controller-propose", [str(binary), "canonical", *common, "--action", "controller-propose", "--base", revision, "--revision", revision], repo, env)
        proposal = parse_json_output(proposal_step)
        if proposal.get("status") != "planned":
            raise SmokeFailure(f"controller proposal status was {proposal.get('status')!r}, not planned")
        trial["proposalDigest"] = proposal.get("digest")

        execute_step_record = execute_step(trial, "controller-execute", [str(binary), "canonical", *common, "--action", "controller-execute", "--base", revision, "--revision", revision], repo, env)
        execute_bytes = Path(execute_step_record["stdoutPath"]).read_bytes()
        execute_path = trial_root / "controller-execute.json"
        execute_path.write_bytes(execute_bytes)
        run = json.loads(execute_bytes)
        reviewed = verify_candidate_review(mode, run)
        trial["executeDigest"] = run.get("digest")
        trial["reviewedOutputs"] = {path: sha256(data) for path, data in sorted(reviewed.items())}
        trial["reviewApproval"] = "explicit --approve-known-fixture; exact paths/bytes checked by this example-specific driver"
        if not args.approve_known_fixture:
            raise SmokeFailure("Apply was not authorized; pass --approve-known-fixture to review and apply only the fixed example candidate")

        apply_args = [
            str(binary), "canonical", *common, "--action", "controller-apply",
            "--base", revision, "--revision", revision, "--plan", str(execute_path),
            "--expect", str(run.get("digest", "")), "--write",
        ]
        apply_step = execute_step(trial, "controller-apply", apply_args, repo, env)
        apply_bytes = Path(apply_step["stdoutPath"]).read_bytes()
        apply_path = trial_root / "controller-apply.json"
        apply_path.write_bytes(apply_bytes)
        applied = json.loads(apply_bytes)
        if applied.get("status") != "materialized-unverified":
            raise SmokeFailure(f"Apply returned unexpected status {applied.get('status')!r}")
        assert_git_after_apply(repo, revision, reviewed, applied, trial["initialIndexTree"])
        trial["applyDigest"] = sha256(apply_bytes)
        trial["sourceRevision"] = applied.get("sourceRevision")
        trial["evidenceRevision"] = applied.get("evidenceRevision")

        verify_step = execute_step(
            trial,
            "controller-verify-from-saved-apply",
            [str(binary), "canonical", *common, "--action", "controller-verify", "--apply-result", str(apply_path), "--write"],
            repo,
            env,
            expected_exit=0 if mode == "positive" else 1,
        )
        verification = parse_json_output(verify_step)
        trial["verificationStatus"] = verification.get("status")
        results = verification.get("results", [])
        if not isinstance(results, list) or not results:
            raise SmokeFailure("fresh Verify returned no assurance results")
        if mode == "positive":
            if verification.get("status") != "passed" or any(item.get("outcome") != "passed" for item in results):
                raise SmokeFailure("positive candidate did not receive passing fresh Verify results")
        else:
            if verification.get("status") != "failed" or not any(item.get("outcome") == "failed" for item in results):
                raise SmokeFailure("bad-business candidate did not fail fresh Verify")
            if any(item.get("outcome") != "passed" for item in verification.get("verifierRuns", [])):
                raise SmokeFailure("protocol-only Verifier response coverage was not preserved separately from failed fixed checks")
        trial["checkReceipts"] = check_receipts(evidence, mode)
        trial["checkEvidenceFiles"] = {
            path.relative_to(evidence).as_posix(): sha256(path.read_bytes())
            for path in sorted(evidence.rglob("*")) if path.is_file()
        }

        audit_step = execute_step(trial, "controller-audit", [str(binary), "canonical", *common, "--action", "controller-audit", "--base", revision, "--revision", revision], repo, env, expected_exit=0 if mode == "positive" else 2)
        audit = parse_json_output(audit_step)
        trial["auditStatus"] = audit.get("status")
        if mode == "positive" and audit.get("status") != "complete":
            raise SmokeFailure("positive fresh audit is not complete")
        if mode == "bad-business" and audit.get("status") != "incomplete":
            raise SmokeFailure("failed behavior verification did not leave audit incomplete")
        assert_git_after_apply(repo, revision, reviewed, applied, trial["initialIndexTree"])
        trial["status"] = "passed"
    except Exception as exc:
        trial["failure"] = f"{type(exc).__name__}: {exc}"
    return trial


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, help="installed native Markitect v0.14.1 executable")
    parser.add_argument("--source-root", required=True, help="clean Git checkout containing the example at --source-revision")
    parser.add_argument("--source-revision", required=True, help="full 40-character Git commit containing the example and tools")
    parser.add_argument("--output", required=True, help="absent external directory for both smoke trials and receipts")
    parser.add_argument("--approve-known-fixture", action="store_true", help="allow this driver to Apply only its exact checked fixture bytes")
    args = parser.parse_args()

    source_root = Path(args.source_root).expanduser().resolve(strict=True)
    binary = Path(args.binary).expanduser().resolve(strict=True)
    if len(args.source_revision) != 40 or any(ch not in "0123456789abcdef" for ch in args.source_revision.lower()):
        parser.error("--source-revision must be a full 40-character Git commit ID")
    actual_head = git(source_root, "rev-parse", "HEAD").decode("ascii").strip().lower()
    if actual_head != args.source_revision.lower():
        parser.error(f"source HEAD {actual_head} does not equal --source-revision")
    if git(source_root, "status", "--porcelain=v1", "--untracked-files=all").strip():
        parser.error("--source-root must have a clean tracked and untracked worktree")
    expected_binary_hash = PUBLISHED_BINARY_SHA256.get(platform.system())
    if expected_binary_hash is None:
        parser.error(f"no pinned published v0.14.1 asset hash is declared for {platform.system()}")
    binary_hash = sha256(binary.read_bytes())
    if binary_hash != expected_binary_hash:
        parser.error(f"--binary SHA-256 {binary_hash} does not match the published {platform.system()} v0.14.1 asset")
    version = run_raw([str(binary), "version"], binary.parent)
    version_text = (version.stdout + version.stderr).decode("utf-8", errors="replace")
    if version.returncode != 0 or "0.14.1" not in version_text:
        parser.error(f"--binary did not identify itself as v0.14.1 (exit {version.returncode})")

    output_root = Path(args.output).expanduser().resolve(strict=False)
    if output_root.exists():
        parser.error("--output must name an absent directory; prior receipts are never overwritten")
    if within(output_root, source_root) or within(output_root, binary.parent):
        parser.error("--output must be outside source and binary directories")
    worktree_listing = git(source_root, "worktree", "list", "--porcelain").decode("utf-8", errors="strict")
    product_worktrees = [
        Path(line.removeprefix("worktree ")).resolve()
        for line in worktree_listing.splitlines() if line.startswith("worktree ")
    ]
    if any(within(output_root, worktree) for worktree in product_worktrees):
        parser.error("--output must be outside every worktree of the source repository")
    if not output_root.parent.is_dir():
        parser.error("--output parent must already exist")
    output_root.mkdir()
    preflight_dir = output_root / "preflight"
    preflight_dir.mkdir()
    version_out = preflight_dir / "version.stdout.bin"
    version_err = preflight_dir / "version.stderr.bin"
    version_out.write_bytes(version.stdout)
    version_err.write_bytes(version.stderr)
    receipt: dict[str, Any] = {
        "schema": "markitect.classic-commerce-cli-smoke/v1",
        "status": "incomplete",
        "createdUtc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "sourceRevision": actual_head,
        "sourceRoot": str(source_root),
        "binary": {"path": str(binary), "sha256": binary_hash, "versionCommand": version_text.strip()},
        "python": platform.python_version(),
        "platform": platform.platform(),
        "limitations": [
            "Executor and Verifier use a deterministic protocol test double, not a model or provider.",
            "Verifier observations report synthetic protocol coverage only; the declared .NET build/behavior check is the independent fixture oracle.",
            "A passing smoke does not establish semantic acceptance, provider quality, autonomous operation, or human approval.",
            "Commands run with the caller's local authority and are not OS-sandboxed.",
        ],
        "trials": [],
    }
    receipt["binaryPreflight"] = {
        "argv": [str(binary), "version"], "exitCode": version.returncode,
        "stdoutPath": str(version_out), "stderrPath": str(version_err),
        "stdoutSha256": sha256(version.stdout), "stderrSha256": sha256(version.stderr),
    }
    receipt_path = output_root / "receipt.json"
    try:
        trials = [run_trial(args, binary, source_root, mode, output_root, receipt) for mode in ("positive", "bad-business")]
        receipt["status"] = "passed" if all(item.get("status") == "passed" for item in trials) else "incomplete"
        receipt["summary"] = "both predeclared example protocol-smoke modes completed" if receipt["status"] == "passed" else "one or more predeclared modes stopped before their expected closure"
    except Exception as exc:
        receipt["failure"] = f"{type(exc).__name__}: {exc}"
    finally:
        receipt_path.write_text(json.dumps(receipt, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(json.dumps({"status": receipt["status"], "receiptPath": str(receipt_path)}, separators=(",", ":")))
    return 0 if receipt["status"] == "passed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
