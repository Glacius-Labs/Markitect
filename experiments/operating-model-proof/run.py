#!/usr/bin/env python3
"""Source-bound driver for real canonical-controller proof runs.

This script calls the Markitect CLI. It does not emulate agents, produce candidates,
or interpret an agent result as acceptance. Full CLI JSON is kept in an external
staging directory; only digests and observed fields are written under runs/.
"""
from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
import re
import subprocess
import sys
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

PROTOCOL = "operating-model-proof/v3"
FULL_SHA = re.compile(r"^(?:[0-9a-f]{40}|[0-9a-f]{64})$")
ROOT = Path(__file__).resolve().parents[2]


class ProofError(Exception):
    pass


def sha(data: bytes) -> str:
    return "sha256:" + hashlib.sha256(data).hexdigest()


def file_sha(path: Path) -> tuple[str, int]:
    if not path.is_file():
        raise ProofError(f"required file is missing: {path}")
    data = path.read_bytes()
    return sha(data), len(data)


def inside(child: Path, parent: Path) -> bool:
    try:
        child.resolve().relative_to(parent.resolve())
        return True
    except ValueError:
        return False


def ensure_external(path: Path, fixture: Path, label: str) -> None:
    if not path.is_absolute():
        raise ProofError(f"{label} must be an absolute path")
    resolved = path.resolve()
    for forbidden in (fixture.resolve(), ROOT.resolve()):
        if inside(resolved, forbidden) or inside(forbidden, resolved):
            raise ProofError(f"{label} overlaps fixture or Markitect checkout")


def git(fixture: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", "-C", str(fixture), *args],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
        shell=False,
    )
    if result.returncode:
        raise ProofError(f"git {args[0]} failed: {result.stderr.decode('utf-8', 'replace').strip()}")
    return result.stdout.decode("utf-8", "strict").strip()


def validate_revision(fixture: Path, revision: str) -> None:
    if not FULL_SHA.fullmatch(revision):
        raise ProofError("base/revision must be full lowercase Git commit IDs")
    git(fixture, "cat-file", "-e", revision + "^{commit}")


def strict_object(path: Path) -> dict[str, Any]:
    def unique(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
        result: dict[str, Any] = {}
        for key, value in pairs:
            if key in result:
                raise ProofError("duplicate JSON object key")
            result[key] = value
        return result

    try:
        value = json.loads(path.read_text(encoding="utf-8"), object_pairs_hook=unique)
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise ProofError(f"cannot read closed JSON: {path}") from exc
    if not isinstance(value, dict):
        raise ProofError("expected one JSON object")
    return value


def strict_bytes(data: bytes) -> dict[str, Any]:
    def unique(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
        result: dict[str, Any] = {}
        for key, value in pairs:
            if key in result:
                raise ProofError("duplicate JSON object key")
            result[key] = value
        return result

    try:
        value = json.loads(data.decode("utf-8", "strict"), object_pairs_hook=unique)
    except (UnicodeError, json.JSONDecodeError) as exc:
        raise ProofError("CLI did not emit one UTF-8 JSON report") from exc
    if not isinstance(value, dict):
        raise ProofError("CLI report is not a JSON object")
    return value


def validate_build_receipt(receipt: dict[str, Any], source_sha: str, cli_digest: str) -> dict[str, Any]:
    if not FULL_SHA.fullmatch(source_sha) or receipt.get("sourceSha") != source_sha:
        raise ProofError("build receipt sourceSha must be the exact full source commit")
    command = receipt.get("buildCommand")
    if not ((isinstance(command, str) and command.strip()) or
            (isinstance(command, list) and command and all(isinstance(part, str) and part for part in command))):
        raise ProofError("build receipt must retain the actual build command")
    if type(receipt.get("exitCode")) is not int or receipt["exitCode"] != 0:
        raise ProofError("Markitect CLI build receipt must record exitCode 0")
    if receipt.get("binaryDigest") != cli_digest:
        raise ProofError("build receipt digest does not match the selected CLI executable")
    return {
        "sourceSha": source_sha,
        "buildCommandDigest": sha(json.dumps(command, ensure_ascii=False, separators=(",", ":")).encode("utf-8")),
        "exitCode": 0,
        "binaryDigest": cli_digest,
    }


def write_new_bytes(path: Path, data: bytes) -> None:
    with path.open("xb") as stream:
        stream.write(data)


def runtime_file_fact(path: Path, digest: str, size: int) -> dict[str, Any]:
    return {"pathDigest": sha(str(path.resolve()).encode()), "digest": digest, "bytes": size}


def new_attempt_paths(root: Path, action: str) -> tuple[str, Path, Path]:
    attempt = uuid.uuid4().hex
    return attempt, root / f"{action}-{attempt}.stdout.json", root / f"{action}-{attempt}.stderr"


def safe_text_facts(items: Any) -> list[dict[str, Any]]:
    result = []
    if not isinstance(items, list):
        return result
    for item in items:
        if not isinstance(item, dict):
            continue
        fact = {key: item[key] for key in ("code", "identity", "subject", "outcome") if key in item}
        detail = item.get("message", item.get("detail"))
        if isinstance(detail, str):
            fact["detailDigest"] = sha(detail.encode("utf-8"))
            fact["detailBytes"] = len(detail.encode("utf-8"))
        result.append(fact)
    return result


def inspect_runtime(path: Path, fixture: Path) -> dict[str, Any]:
    raw = path.read_bytes()
    cfg = strict_object(path)
    if cfg.get("apiVersion") != "markitect.canonical/controller/v1alpha1":
        raise ProofError("unsupported controller runtime API")
    record_store = Path(cfg.get("recordStore", ""))
    private_logs = Path(cfg.get("privateLogs", ""))
    ensure_external(record_store, fixture, "recordStore")
    ensure_external(private_logs, fixture, "privateLogs")
    if record_store.resolve() == private_logs.resolve():
        raise ProofError("recordStore and privateLogs must be disjoint")
    if inside(record_store, private_logs) or inside(private_logs, record_store):
        raise ProofError("recordStore and privateLogs overlap")

    files: dict[str, dict[str, Any]] = {}
    for role in ("executor", "verifier"):
        runner = cfg.get(role)
        if not isinstance(runner, dict):
            raise ProofError(f"{role} runner is missing")
        if runner.get("model") != "gpt-5.5" or runner.get("providerVersion") != "0.130.0":
            raise ProofError(f"{role} must use the frozen gpt-5.5 / Codex 0.130.0 binding")
        options = runner.get("modelOptions")
        if not isinstance(options, dict) or options.get("model_reasoning_effort") != "high":
            raise ProofError(f"{role} must explicitly bind high reasoning effort")
        args = runner.get("args")
        if not isinstance(args, list) or not args:
            raise ProofError(f"{role} command arguments are missing")
        command = Path(runner.get("command", ""))
        if not command.is_absolute() or not command.is_file():
            raise ProofError(f"{role} command must be an absolute executable file")
        py_version = subprocess.run([str(command), "--version"], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False, shell=False).stdout.decode("utf-8", "replace").strip()
        if "Python 3.13." not in py_version:
            raise ProofError(f"{role} must use Python 3.13, got: {py_version}")
        if "--codex-executable" not in args or "--codex-version" not in args or "--model" not in args:
            raise ProofError(f"{role} command must bind the native Codex executable, version and model")
        expected_runner = ROOT / "internal" / "tooling" / "codexrunner" / "runner.py"
        if not any(isinstance(a, str) and Path(a).resolve() == expected_runner.resolve() for a in args):
            raise ProofError(f"{role} must use the repository Codex adapter")
        codex_path = Path(args[args.index("--codex-executable") + 1])
        if not codex_path.is_absolute() or codex_path.suffix.lower() != ".exe" or not codex_path.is_file():
            raise ProofError(f"{role} must bind an absolute native Windows Codex executable")
        if args[args.index("--codex-version") + 1] != "0.130.0" or args[args.index("--model") + 1] != "gpt-5.5":
            raise ProofError(f"{role} Codex CLI/model binding differs from protocol")
        if command.is_absolute() and command.is_file():
            digest, size = file_sha(command)
            path_digest = sha(str(command.resolve()).encode())
            files[path_digest] = runtime_file_fact(command, digest, size)
        args = runner.get("args")
        if not isinstance(args, list) or not args:
            raise ProofError(f"{role} command arguments are missing")
        for arg in args:
            if isinstance(arg, str) and Path(arg).is_absolute() and Path(arg).is_file():
                candidate = Path(arg)
                digest, size = file_sha(candidate)
                path_digest = sha(str(candidate.resolve()).encode())
                files[path_digest] = runtime_file_fact(candidate, digest, size)
        for item in runner.get("runtimeFiles", []):
            if not isinstance(item, dict) or not isinstance(item.get("path"), str):
                raise ProofError(f"{role} runtimeFiles entry is invalid")
            runtime_file = Path(item["path"])
            digest, size = file_sha(runtime_file)
            if item.get("digest") != digest:
                raise ProofError(f"{role} runtime file digest mismatch: {runtime_file}")
            path_digest = sha(str(runtime_file.resolve()).encode())
            files[path_digest] = runtime_file_fact(runtime_file, digest, size)
    total = sum(item["bytes"] for item in files.values())
    if total > 256 * 1024 * 1024:
        raise ProofError(f"configured runtime files exceed 256 MiB ({total} bytes)")
    return {
        "runtimeConfigDigest": sha(raw),
        "recordStorePathDigest": sha(str(record_store.resolve()).encode()),
        "privateLogsPathDigest": sha(str(private_logs.resolve()).encode()),
        "runtimeFiles": files,
        "runtimeBytes": total,
    }


def summarize_receipt(value: Any) -> Any:
    if isinstance(value, list):
        return [summarize_receipt(item) for item in value]
    if not isinstance(value, dict):
        return value
    keep = {
        "apiVersion", "runId", "inputDigest", "contextDigest", "configDigest",
        "commandDigest", "executableDigest", "runtimeFilesDigest",
        "providerVersion", "providerVersionDigest", "stdoutDigest", "stderrDigest",
        "privateLogDigest", "outcome", "wallTimeMilliseconds", "retryCount", "usage",
    }
    output = {key: summarize_receipt(val) for key, val in value.items() if key in keep and key != "usage"}
    usage = value.get("usage")
    if isinstance(usage, dict):
        usage_fields = ("source", "inputTokens", "outputTokens", "cachedTokens", "toolCalls")
        output["usage"] = {key: usage[key] for key in usage_fields if key in usage and usage[key] is not None}
    return output


def proposal_metrics(report: dict[str, Any]) -> dict[str, Any]:
    plan = report.get("plan", {})
    proposals = plan.get("proposals", []) if isinstance(plan, dict) else []
    return {
        "status": report.get("status"),
        "proposalDigest": report.get("digest"),
        "configDigest": report.get("configDigest"),
        "inputDigest": report.get("inputDigest"),
        "ledgerHead": report.get("ledgerHead"),
        "ledgerSelectionDigest": report.get("ledgerSelectionDigest"),
        "baseRevision": plan.get("baseRevision") if isinstance(plan, dict) else None,
        "revision": plan.get("revision") if isinstance(plan, dict) else None,
        "sourceModelDigest": plan.get("modelDigest") if isinstance(plan, dict) else None,
        "observedPaths": plan.get("observedPaths", []) if isinstance(plan, dict) else [],
        "unknownArtifacts": plan.get("unknownArtifacts", []) if isinstance(plan, dict) else [],
        "evidenceRefreshRequired": plan.get("evidenceRefreshRequired", []) if isinstance(plan, dict) else [],
        "unobservedProjections": plan.get("unobservedProjections", []) if isinstance(plan, dict) else [],
        "proposals": [{
            "projectionId": p.get("projectionId"),
            "module": {k: p.get("module", {}).get(k) for k in ("name", "version", "digest")},
            "requestDigest": p.get("request", {}).get("requestDigest"),
            "targetDigests": p.get("request", {}).get("targetDigests", {}),
            "decision": p.get("decision"),
            "reasonCount": len(p.get("reasons", [])),
            "escalations": safe_text_facts(p.get("escalations", [])),
        } for p in proposals if isinstance(p, dict)],
    }


def run_metrics(action: str, report: dict[str, Any]) -> dict[str, Any]:
    if action == "controller-propose":
        return proposal_metrics(report)
    if action == "controller-execute":
        work = []
        for item in report.get("work", []):
            if not isinstance(item, dict):
                continue
            outputs = item.get("outputs", {})
            output_facts = []
            if isinstance(outputs, dict):
                for name, encoded in sorted(outputs.items()):
                    data = base64.b64decode(encoded, validate=True)
                    output_facts.append({"path": name, "digest": sha(data), "bytes": len(data)})
            work.append({
                "projectionId": item.get("projectionId"),
                "candidateDigest": item.get("candidateDigest"),
                "planDigest": item.get("planDigest"),
                "candidateBytes": len(base64.b64decode(item.get("candidate", ""), validate=True)),
                "outputs": output_facts,
                "executorReceipt": summarize_receipt(item.get("executor")),
                "escalations": safe_text_facts(item.get("escalations", [])),
            })
        return {
            "status": report.get("status"),
            "runDigest": report.get("digest"),
            "proposal": proposal_metrics(report.get("proposal", {})),
            "executorConfigDigest": report.get("executorDigest"),
            "verifierConfigDigest": report.get("verifierDigest"),
            "hostExecutableDigest": report.get("hostExecutableDigest"),
            "toolVersion": report.get("toolVersion"),
            "toolDigest": report.get("toolDigest"),
            "work": work,
        }
    if action == "controller-apply":
        return {
            "status": report.get("status"),
            "runDigest": report.get("runDigest"),
            "written": report.get("written", []),
            "records": [{"id": r.get("id"), "projectionId": r.get("projectionId")} for r in report.get("records", [])],
            "ledgerHead": report.get("ledgerHead"),
            "evidenceRevision": report.get("evidenceRevision"),
            "evidencePaths": report.get("evidencePaths", []),
            "evidenceRefreshRequired": report.get("evidenceRefreshRequired", []),
        }
    return {
        "status": report.get("status"),
        "outcome": report.get("outcome"),
        "sourceRevision": report.get("sourceRevision"),
        "evidenceRevision": report.get("evidenceRevision"),
        "ledgerHead": report.get("ledgerHead"),
        "ledgerSelectionDigest": report.get("ledgerSelectionDigest"),
        "results": [{"id": r.get("id"), "recordId": r.get("recordId"), "outcome": r.get("outcome")} for r in report.get("results", [])],
        "verifierRuns": [{
            "scopeId": v.get("scopeId"), "projectionId": v.get("projectionId"),
            "recordId": v.get("recordId"), "resultId": v.get("resultId"),
            "evidenceSnapshotDigest": v.get("evidenceSnapshotDigest"),
            "configFingerprint": v.get("configFingerprint"), "receiptDigest": v.get("receiptDigest"),
            "inputDigest": v.get("inputDigest"), "runId": v.get("runId"), "outcome": v.get("outcome"),
            "evidenceRefCount": len(v.get("evidenceRefs", [])),
            "evidenceRefsDigest": sha(json.dumps(v.get("evidenceRefs", []), sort_keys=True, separators=(",", ":")).encode("utf-8")),
            "observations": safe_text_facts(v.get("observations")),
            "receipt": summarize_receipt(v.get("receipt")),
        } for v in report.get("verifierRuns", [])],
        "assuranceNodeCount": len(report.get("assurance", {}).get("nodes", [])),
        "assuranceDigest": sha(json.dumps(report.get("assurance", {}), sort_keys=True, separators=(",", ":")).encode("utf-8")),
        "limits": report.get("limits", []),
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["propose", "execute", "apply", "verify"])
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--fixture", required=True, type=Path)
    parser.add_argument("--cli", required=True, type=Path)
    parser.add_argument("--config", required=True, help="fixture-relative canonical config path")
    parser.add_argument("--runtime", required=True, type=Path)
    parser.add_argument("--build-receipt", required=True, type=Path, help="external source-SHA/build-command/exit/binary-digest receipt")
    parser.add_argument("--external-root", required=True, type=Path, help="fresh absolute directory outside the fixture")
    parser.add_argument("--base", required=True, help="full source base SHA")
    parser.add_argument("--revision", required=True, help="full candidate SHA, or evidence SHA for verify")
    parser.add_argument("--reviewed-run", type=Path, help="external controller-execute JSON required by apply")
    parser.add_argument("--expect", help="exact reviewed-run digest required by apply")
    parser.add_argument("--write", action="store_true", help="explicitly append verification results")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,63}", args.run_id):
        raise ProofError("run-id must be a short filesystem-safe identifier")
    fixture = args.fixture.resolve(strict=True)
    cli = args.cli.resolve(strict=True)
    runtime_path = args.runtime.resolve(strict=True)
    ensure_external(runtime_path, fixture, "runtime configuration")
    build_receipt_path = args.build_receipt.resolve(strict=True)
    ensure_external(build_receipt_path, fixture, "CLI build receipt")
    external = args.external_root
    if not external.is_absolute():
        raise ProofError("external-root must be absolute")
    ensure_external(external, fixture, "external-root")
    if fixture == ROOT.resolve():
        raise ProofError("fixture must be a separate clone of the public operating-model fixture")
    if "konfyra" in str(fixture).lower():
        raise ProofError("private adopter paths are forbidden in this protocol")
    origin = git(fixture, "remote", "get-url", "origin") if git(fixture, "remote") else ""
    if "konfyra" in origin.lower():
        raise ProofError("private adopter remotes are forbidden")
    base = args.base
    revision = args.revision
    validate_revision(fixture, base)
    validate_revision(fixture, revision)
    if args.action in {"propose", "execute"}:
        if git(fixture, "rev-parse", "HEAD") != revision:
            raise ProofError("fixture HEAD must exactly equal --revision")
        if git(fixture, "status", "--porcelain"):
            raise ProofError("proposal/execute requires a clean fixture source tree")
    if args.action == "verify" and args.write is False:
        raise ProofError("verification evidence requires explicit --write")
    if args.action == "apply":
        if not args.write:
            raise ProofError("apply requires explicit --write")
        if not args.reviewed_run or not args.expect:
            raise ProofError("apply requires --reviewed-run and --expect")
        if args.reviewed_run.resolve().parent == fixture or inside(args.reviewed_run, fixture):
            raise ProofError("reviewed run must remain outside the fixture")
    elif args.reviewed_run or args.expect:
        raise ProofError("--reviewed-run and --expect apply only to apply")
    runtime_facts = inspect_runtime(runtime_path, fixture)
    if not cli.is_file():
        raise ProofError("CLI executable is missing")
    cli_digest, cli_bytes = file_sha(cli)
    build_receipt = strict_object(build_receipt_path)
    source_sha = build_receipt.get("sourceSha", "")
    git(ROOT, "cat-file", "-e", source_sha + "^{commit}")
    build_facts = validate_build_receipt(build_receipt, source_sha, cli_digest)
    build_facts["receiptDigest"] = file_sha(build_receipt_path)[0]
    cli_version_result = subprocess.run([str(cli), "version"], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False, shell=False)
    if cli_version_result.returncode:
        raise ProofError("Markitect CLI version command failed")
    cli_version = cli_version_result.stdout.decode("utf-8", "replace").strip()

    local_run = ROOT / "experiments" / "operating-model-proof" / "runs" / args.run_id
    local_run.mkdir(parents=True, exist_ok=False if args.action == "propose" else True)
    event_dir = local_run / "events"
    event_dir.mkdir(exist_ok=True)
    external_run = external / args.run_id
    external_run.mkdir(parents=True, exist_ok=False if args.action == "propose" else True)
    meta_path = local_run / "binding.json"
    if args.action == "propose":
        binding = {
            "protocol": PROTOCOL,
            "runId": args.run_id,
            "fixtureOrigin": origin,
            "fixtureHead": git(fixture, "rev-parse", "HEAD"),
            "markitectSourceSHA": source_sha,
            "fixtureBase": base,
            "fixtureRevision": revision,
            "config": args.config,
            "runtime": runtime_facts,
            "buildReceipt": build_facts,
            "cli": {"pathDigest": sha(str(cli).encode()), "digest": cli_digest, "bytes": cli_bytes, "version": cli_version},
            "createdAtUtc": datetime.now(timezone.utc).isoformat(),
        }
        write_json(meta_path, binding, exclusive=True)
    else:
        binding = strict_object(meta_path)
        if binding.get("protocol") != PROTOCOL or (args.action == "verify" and binding.get("fixtureRevision") != base) or (args.action != "verify" and (binding.get("fixtureBase") != base or binding.get("fixtureRevision") != revision)):
            raise ProofError("action does not match the run's frozen source/fixture bindings")
        if (binding.get("cli", {}).get("digest") != cli_digest or
                binding.get("runtime") != runtime_facts or binding.get("buildReceipt") != build_facts):
            raise ProofError("CLI, source-bound build receipt or runtime differs from the frozen run binding")

    command = [str(cli), "canonical", "--action", {
        "propose": "controller-propose", "execute": "controller-execute",
        "apply": "controller-apply", "verify": "controller-verify",
    }[args.action], "--repo", str(fixture), "--config", args.config,
        "--runtime", str(runtime_path), "--base", base, "--revision", revision]
    if args.action == "apply":
        reviewed = args.reviewed_run.resolve(strict=True)
        run_obj = strict_object(reviewed)
        if run_obj.get("digest") != args.expect:
            raise ProofError("--expect does not equal the reviewed run digest")
        if run_obj.get("proposal", {}).get("plan", {}).get("baseRevision") != base or run_obj.get("proposal", {}).get("plan", {}).get("revision") != revision:
            raise ProofError("reviewed run source revisions do not match this apply")
        command.extend(["--plan", str(reviewed), "--expect", args.expect, "--write"])
    if args.action == "verify" and args.write:
        command.append("--write")

    started = time.perf_counter()
    completed = subprocess.run(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False, shell=False)
    elapsed_ms = round((time.perf_counter() - started) * 1000)
    attempt_id, out_path, err_path = new_attempt_paths(external_run, args.action)
    write_new_bytes(out_path, completed.stdout)
    write_new_bytes(err_path, completed.stderr)
    try:
        report = strict_bytes(completed.stdout)
    except ProofError:
        report = {}
    command_template = ["markitect", "canonical", "--action", command[3], "--repo", "<fixture>", "--config", args.config, "--runtime", "<runtime>", "--base", base, "--revision", revision]
    if args.action == "apply":
        command_template.extend(["--plan", "<reviewed-run>", "--expect", args.expect, "--write"])
    elif args.action == "verify":
        command_template.append("--write")
    event = {
        "protocol": PROTOCOL,
        "runId": args.run_id,
        "attemptId": attempt_id,
        "action": args.action,
        "commandTemplate": command_template,
        "exitCode": completed.returncode,
        "elapsedWallMilliseconds": elapsed_ms,
        "stdoutDigest": sha(completed.stdout),
        "stdoutBytes": len(completed.stdout),
        "stderrDigest": sha(completed.stderr),
        "stderrBytes": len(completed.stderr),
        "externalOutput": f"<external-stage>/{args.run_id}/{out_path.name}",
        "externalOutputDigest": sha(completed.stdout),
        "metrics": run_metrics({"propose": "controller-propose", "execute": "controller-execute", "apply": "controller-apply", "verify": "controller-verify"}[args.action], report),
    }
    event_path = event_dir / f"{args.action}-{uuid.uuid4().hex}.json"
    write_json(event_path, event, exclusive=True)
    with (local_run / "events.jsonl").open("a", encoding="utf-8", newline="\n") as stream:
        stream.write(json.dumps(event, sort_keys=True, separators=(",", ":")) + "\n")
    if args.action == "execute":
        # Exact JSON is required for later guarded Apply; keep it outside the repository.
        if report:
            exact = completed.stdout.rstrip(b"\r\n") + b"\n"
            run_path = external_run / f"reviewed-run-{attempt_id}.json"
            write_new_bytes(run_path, exact)
            print(json.dumps({"status": report.get("status"), "runDigest": report.get("digest"), "reviewedRunExternalPath": str(run_path), "stdoutDigest": sha(exact)}, sort_keys=True))
    else:
        print(json.dumps(event, sort_keys=True))
    return 0 if completed.returncode == 0 else completed.returncode


def write_json(path: Path, value: Any, *, exclusive: bool = False) -> None:
    mode = "x" if exclusive else "w"
    with path.open(mode, encoding="utf-8", newline="\n") as stream:
        json.dump(value, stream, ensure_ascii=False, sort_keys=True, separators=(",", ":"))
        stream.write("\n")


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ProofError as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(2)
