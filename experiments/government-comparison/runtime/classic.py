"""Native Markitect Classic v0.14.1 adapter facts and bounded argv construction.

No provider is started here. The common Scientist harness owns limits, request/result
binding, trial ledger, deadlines, and raw receipt storage.
"""
from __future__ import annotations

import hashlib
import json
import os
import subprocess
import time
import tempfile
from pathlib import Path
from typing import Any

EXPECTED_SOURCE = "7dbd599c81540c8203a1b7f83afbc335174f4f1f"
EXPECTED_HELD_SOURCE = "c91363b7ac4decbe87212ff0f588b5451581a152"
EXPECTED_BINARY_SHA256 = "2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4"
MAX_CALL_SECONDS = 180


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def inspect_packet(packet_path: str | os.PathLike[str]) -> dict[str, Any]:
    """Verify the frozen readiness packet and return explicit version/capability facts."""
    packet = Path(packet_path).resolve(strict=True)
    manifest = packet / "checksums.sha256"
    if not manifest.is_file():
        raise ValueError(f"missing packet checksum manifest: {manifest}")
    pin = json.loads(Path(__file__).with_name("classic-pin.json").read_text(encoding="utf-8"))
    if sha256_file(manifest) != pin["packetManifestSha256"]:
        raise ValueError("packet checksum manifest differs from external Scientist pin")
    checked = 0
    for line_no, line in enumerate(manifest.read_text(encoding="utf-8").splitlines(), 1):
        if not line.strip():
            continue
        try:
            expected, relative = line.split("  ", 1)
        except ValueError as exc:
            raise ValueError(f"malformed checksum line {line_no}") from exc
        target = (packet / relative).resolve(strict=True)
        if packet not in target.parents:
            raise ValueError(f"packet path escapes root: {relative}")
        actual = sha256_file(target)
        if actual != expected:
            raise ValueError(f"packet hash mismatch for {relative}: {actual}")
        checked += 1
    data = json.loads((packet / "classic-readiness.json").read_text(encoding="utf-8"))
    binary = packet / "artifacts" / "markitect-v0.14.1-windows-amd64.exe"
    binary_sha = sha256_file(binary)
    identities = data["identities"]
    source = identities["runtimeSource"]
    held = identities["heldSource"]
    if source != EXPECTED_SOURCE or held != EXPECTED_HELD_SOURCE:
        raise ValueError("packet runtime/held source identities differ from the frozen study pin")
    if binary_sha != EXPECTED_BINARY_SHA256:
        raise ValueError(f"published Windows binary digest mismatch: {binary_sha}")
    modules = [{"name": item["name"], "version": item["version"], "sha256": item["digest"].removeprefix("sha256:")}
               for item in data["modulePins"]]
    if modules != pin["modules"]:
        raise ValueError("module identities differ from external Scientist pin")
    if {"path": data["canonicalConfig"], "sha256": data["canonicalConfigSha256"]} != pin["canonicalConfig"]:
        raise ValueError("canonical config identity differs from external Scientist pin")
    if identities["sourceBundleSha256"] != pin["sourceBundleSha256"]:
        raise ValueError("source bundle identity differs from external Scientist pin")
    return {
        "packetPath": str(packet),
        "packetFilesVerified": checked,
        "product": "Markitect Classic",
        "version": "0.14.1",
        "runtimeSourceSha": source,
        "heldSourceSha": held,
        "sourceBundleSha256": identities["sourceBundleSha256"],
        "annotatedTagObject": "784d3c3c61443b291ac7db9727c7c5856e253d66",
        "releaseId": "405781964",
        "binary": {"path": str(binary), "platform": "windows/amd64", "sha256": binary_sha},
        "modulePins": data["modulePins"],
        "canonicalConfig": {"path": data["canonicalConfig"], "sha256": data["canonicalConfigSha256"]},
        "capabilities": [
            "version and read-only project/model/context/impact inspection",
            "canonical controller proposal and Execute through configured Executor/Verifier",
            "guarded Apply after external exact-digest review",
            "fresh Verify from saved Apply result",
            "read-only scoped Audit",
        ],
        "gaps": [
            "no native Scientist study role or --request/--result interface",
            "no general persistent resume or native study stop operation",
            "no provider/model provisioned by product; provider/model pins and auth are external",
            "Apply authorization is not authenticated by the CLI",
            "actual provider usage is optional; product does not enforce shared trial token/session budgets",
            "controller is experimental alpha; semantic quality and unattended operation unproven",
            "local caller filesystem/process authority; no OS sandbox or selected-scope privacy claim",
            "protocol/readiness packet synthetic smoke is not a live-agent result",
        ],
        "policyPreparation": "The published Commerce smoke explicitly prepares three Markdown policies using the release-test helper; bare canonical fixture escalates without this setup. This is synthetic setup, not model intent.",
    }


def probe_command(binary_path: str | os.PathLike[str]) -> list[str]:
    binary = str(Path(binary_path).resolve(strict=True))
    return [binary, "version"]


def native_command(
    action: str,
    *,
    binary_path: str | os.PathLike[str],
    repo_path: str | os.PathLike[str],
    config_path: str,
    revision: str,
    runtime_path: str | os.PathLike[str] | None = None,
    execute_report: str | os.PathLike[str] | None = None,
    reviewed_digest: str | None = None,
    apply_result: str | os.PathLike[str] | None = None,
) -> dict[str, Any]:
    """Build one documented native CLI invocation; no shell interpolation is used."""
    binary = str(Path(binary_path).resolve(strict=True))
    repo = str(Path(repo_path).resolve(strict=True))
    base = revision
    argv = [binary]
    if action == "model":
        argv += ["canonical", "--repo", repo, "--config", config_path, "--action", "model", "--revision", revision]
        output = "yaml"
    elif action == "reconcile":
        argv += ["canonical", "--repo", repo, "--config", config_path, "--action", "reconcile-plan", "--base", base, "--revision", revision]
        output = "yaml"
    elif action in {"propose", "execute", "apply", "verify", "audit"}:
        if runtime_path is None:
            raise ValueError(f"{action} requires an explicit runtime path")
        argv += ["canonical", "--repo", repo, "--config", config_path, "--runtime", str(Path(runtime_path).resolve(strict=True))]
        if action == "propose":
            argv += ["--action", "controller-propose", "--base", base, "--revision", revision]
        elif action == "execute":
            argv += ["--action", "controller-execute", "--base", base, "--revision", revision]
        elif action == "apply":
            if execute_report is None or not reviewed_digest:
                raise ValueError("apply requires exact execute_report and externally reviewed_digest")
            argv += ["--action", "controller-apply", "--base", base, "--revision", revision,
                     "--plan", str(Path(execute_report).resolve(strict=True)), "--expect", reviewed_digest, "--write"]
        elif action == "verify":
            if apply_result is None:
                raise ValueError("verify requires saved apply_result")
            argv += ["--action", "controller-verify", "--apply-result", str(Path(apply_result).resolve(strict=True)), "--write"]
        else:
            argv += ["--action", "controller-audit", "--base", base, "--revision", revision]
        output = "json"
    else:
        raise ValueError(f"unsupported Classic action: {action}")
    return {"argv": argv, "cwd": repo, "expectedReport": output, "timeoutSeconds": MAX_CALL_SECONDS,
            "automaticRetries": 0, "productSourceSha": EXPECTED_SOURCE,
            "note": "Each CLI call is a fresh process; classify status from report and exit code per product contract."}


def run_native(spec: dict[str, Any], *, timeout_seconds: int = MAX_CALL_SECONDS) -> dict[str, Any]:
    """Execute a single product process and retain exact command provenance in memory.

    The common wrapper should persist this structure and captured bytes as its raw receipt.
    It enforces the hard per-process cap independently from provider/session budgets.
    """
    timeout = min(max(int(timeout_seconds), 1), MAX_CALL_SECONDS)
    started = time.monotonic()
    from process import bounded
    with tempfile.TemporaryDirectory(prefix="scientist-native-call-") as temporary:
        directory = Path(temporary) / "receipt"
        receipt = bounded(spec["argv"], spec["cwd"], directory, timeout)
        return {"argv": spec["argv"], "cwd": spec["cwd"], "returnCode": receipt["returnCode"],
                "stdout": (directory / "stdout.log").read_bytes(), "stderr": (directory / "stderr.log").read_bytes(),
                "wallSeconds": time.monotonic() - started, "timedOut": receipt["stopReason"] == "wall_deadline",
                "processTreeControl": receipt["processTreeControl"], "stopReason": receipt["stopReason"],
                "productSourceSha": EXPECTED_SOURCE, "runtimeSha256": sha256_file(Path(spec["argv"][0]))}
