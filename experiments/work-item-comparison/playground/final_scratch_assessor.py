"""Final-only Conventional assessment using a hash-bound disposable copy.

This module provides transport mechanics, not a scientific score. The native
assessor receives one writable execution copy; frozen candidate and public
requirements remain outside its working directory and are rechecked after the
turn. Only newly-created files below ``.scratch`` and ``.assessment-output``
inside the execution copy are accepted as runtime artifacts.
"""

from __future__ import annotations

from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import shutil
import threading
from typing import Any, Mapping


_ARTIFACT_ROOTS = {".scratch", ".assessment-output"}
_ASSESSMENT_DIR = "supplemental-assessment"
_NATIVE_ENVIRONMENT_POLICY = {
    "PYTHONDONTWRITEBYTECODE": "1",
    "scope": "temporarily set in assessor process and inherited by native child during the single run call",
    "priorValue": "not captured",
    "restoredAfterRun": True,
}


def _utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def _sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def _canonical_manifest(manifest: Mapping[str, str]) -> dict[str, str]:
    if not isinstance(manifest, Mapping) or not manifest:
        raise ValueError("expected nonempty frozen file manifest")
    normalized: dict[str, str] = {}
    for raw_name, digest in manifest.items():
        if not isinstance(raw_name, str) or not raw_name:
            raise ValueError("manifest paths must be nonempty relative strings")
        name = PurePosixPath(raw_name.replace("\\", "/"))
        if name.is_absolute() or any(part in {"", ".", ".."} for part in name.parts):
            raise ValueError("manifest contains an unsafe relative path")
        if ".git" in name.parts:
            raise ValueError("Git metadata is not part of the frozen semantic manifest")
        if not isinstance(digest, str) or len(digest) != 64:
            raise ValueError("manifest values must be SHA-256 hex digests")
        try:
            int(digest, 16)
        except ValueError as exc:
            raise ValueError("manifest values must be SHA-256 hex digests") from exc
        key = name.as_posix()
        if key in normalized:
            raise ValueError("manifest contains duplicate normalized paths")
        normalized[key] = digest.lower()
    return dict(sorted(normalized.items()))


def _manifest(root: Path) -> dict[str, str]:
    if not root.is_dir() or root.is_symlink():
        raise ValueError(f"bound root must be an existing non-symlink directory: {root}")
    result: dict[str, str] = {}
    for path in sorted(root.rglob("*")):
        relative = path.relative_to(root)
        if ".git" in relative.parts:
            continue
        if path.is_symlink():
            raise ValueError(f"symlink found in bound tree: {relative.as_posix()}")
        if path.is_file():
            result[relative.as_posix()] = _sha256(path.read_bytes())
    return dict(sorted(result.items()))


def _has_git_metadata(root: Path) -> bool:
    return any(path.name == ".git" for path in root.rglob(".git"))


def _helper_activity_started(event: dict[str, Any]) -> bool:
    """Conservatively notice explicit helper/collaboration start events."""
    parsed = event.get("parsed")
    if not isinstance(parsed, dict):
        return False
    method = parsed.get("method")
    if isinstance(method, str) and "collab" in method.lower() and "start" in method.lower():
        return True

    def visit(value: Any) -> bool:
        if isinstance(value, dict):
            kind = value.get("kind")
            activity_type = value.get("type")
            status = value.get("status", value.get("state"))
            if (isinstance(activity_type, str) and
                    activity_type.lower() in {"subagentactivity", "collabtoolcall", "collabagenttoolcall"} and
                    isinstance(kind, str) and kind.lower() in {"started", "start"}):
                return True
            if (isinstance(activity_type, str) and "collab" in activity_type.lower() and
                    isinstance(status, str) and status.lower() in {"started", "start", "running"}):
                return True
            return any(visit(child) for child in value.values())
        if isinstance(value, list):
            return any(visit(child) for child in value)
        return False

    return visit(parsed)


def _manifest_digest(manifest: Mapping[str, str]) -> str:
    encoded = (json.dumps(dict(sorted(manifest.items())), sort_keys=True,
                          separators=(",", ":")) + "\n").encode("utf-8")
    return _sha256(encoded)


def _execution_matches_frozen(expected: dict[str, str], current: dict[str, str]) -> bool:
    """Require exact semantic files and preservation of any pre-existing artifacts."""
    for name, digest in expected.items():
        if current.get(name) != digest:
            return False
    expected_semantic = {name: digest for name, digest in expected.items()
                         if PurePosixPath(name).parts[0] not in _ARTIFACT_ROOTS}
    current_semantic = {name: digest for name, digest in current.items()
                        if PurePosixPath(name).parts[0] not in _ARTIFACT_ROOTS}
    return expected_semantic == current_semantic


def _resolve_source_tree(path: Path, label: str) -> Path:
    path = Path(path)
    if path.is_symlink():
        raise ValueError(f"{label} root must not be a symlink")
    resolved = path.resolve(strict=True)
    if not resolved.is_dir():
        raise ValueError(f"{label} root must be an existing directory")
    return resolved


def _copy_manifest(source: Path, destination: Path, expected: dict[str, str]) -> None:
    """Copy only the already-verified regular files named by a frozen manifest."""
    actual = _manifest(source)
    if actual != expected:
        raise ValueError(f"frozen input differs from its expected manifest: {source}")
    destination.mkdir(parents=True, exist_ok=False)
    for relative_name, digest in expected.items():
        relative = PurePosixPath(relative_name)
        source_file = source.joinpath(*relative.parts)
        target_file = destination.joinpath(*relative.parts)
        if source_file.is_symlink() or not source_file.is_file():
            raise ValueError(f"frozen input is not a regular file: {relative_name}")
        if _sha256(source_file.read_bytes()) != digest:
            raise ValueError(f"frozen input changed during copy: {relative_name}")
        target_file.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source_file, target_file)
    if _manifest(destination) != expected:
        raise ValueError(f"execution copy differs from frozen input: {destination}")


def _safe_audit(audit: Path) -> Path:
    audit = Path(audit)
    if audit.is_symlink():
        raise ValueError("audit path must be an existing non-symlink directory")
    audit = audit.resolve(strict=True)
    if not audit.is_dir():
        raise ValueError("audit path must be an existing non-symlink directory")
    return audit


def prepare_scratch(candidate: Path, audit: Path, initial_public: Path,
                    expected_candidate_manifest: Mapping[str, str],
                    expected_public_manifest: Mapping[str, str]) -> dict[str, Any]:
    """Create a fresh execution copy and frozen public requirements sibling.

    The returned execution copy is the intended native AppServer CWD. The
    requirements sibling is readable from that CWD but is not under its
    writable root.
    """
    candidate = _resolve_source_tree(Path(candidate), "candidate")
    initial_public = _resolve_source_tree(Path(initial_public), "public requirements")
    audit = _safe_audit(Path(audit))
    expected_candidate = _canonical_manifest(expected_candidate_manifest)
    expected_public = _canonical_manifest(expected_public_manifest)
    if _manifest(candidate) != expected_candidate:
        raise ValueError("frozen candidate differs from its expected manifest")
    if _manifest(initial_public) != expected_public:
        raise ValueError("frozen public requirements differ from their expected manifest")

    scratch_root = audit / "scratch"
    binding_path = audit / "scratch-preparation-binding.json"
    if scratch_root.exists() or scratch_root.is_symlink() or binding_path.exists() or binding_path.is_symlink():
        raise ValueError("scratch assessment preparation already exists; reuse is forbidden")
    scratch_root.mkdir()
    execution_candidate = scratch_root / "candidate"
    requirements = scratch_root / "requirements"
    _copy_manifest(candidate, execution_candidate, expected_candidate)
    _copy_manifest(initial_public, requirements, expected_public)
    # Existing candidate tests use this scratch DB location. The assessor may
    # add runtime data here, while semantic files remain hash-pinned.
    (execution_candidate / ".scratch").mkdir(exist_ok=True)
    (execution_candidate / ".assessment-output").mkdir(exist_ok=True)
    if _has_git_metadata(execution_candidate) or _manifest(execution_candidate) != expected_candidate:
        raise ValueError("assessment artifact directories unexpectedly contain semantic files")
    binding = {
        "schema": 1,
        "createdAt": _utc_now(),
        "kind": "conventional_final_scratch_preparation",
        "frozenCandidatePath": str(candidate),
        "frozenCandidateManifest": expected_candidate,
        "frozenCandidateManifestSha256": _manifest_digest(expected_candidate),
        "executionCandidatePath": str(execution_candidate.resolve(strict=True)),
        "executionCandidateManifest": _manifest(execution_candidate),
        "frozenPublicRequirementsPath": str(initial_public),
        "frozenPublicRequirementsManifest": expected_public,
        "frozenPublicRequirementsManifestSha256": _manifest_digest(expected_public),
        "executionRequirementsPath": str(requirements.resolve(strict=True)),
        "executionRequirementsManifest": _manifest(requirements),
        "allowedArtifactRoots": sorted(_ARTIFACT_ROOTS),
        "writableRoot": str(execution_candidate.resolve(strict=True)),
        "writableRoots": [],
        "nativeStarts": 0,
    }
    binding_path.write_text(json.dumps(binding, indent=2) + "\n", encoding="utf-8")
    return {
        "scratchRoot": scratch_root.resolve(strict=True),
        "executionCandidate": execution_candidate.resolve(strict=True),
        "requirements": requirements.resolve(strict=True),
        "bindingPath": binding_path.resolve(strict=True),
        "binding": binding,
    }


def _verify_prepared(candidate: Path, audit: Path, initial_public: Path,
                     expected_candidate: dict[str, str], expected_public: dict[str, str]) -> dict[str, Any]:
    candidate = _resolve_source_tree(candidate, "candidate")
    initial_public = _resolve_source_tree(initial_public, "public requirements")
    audit = _safe_audit(audit)
    binding_path = audit / "scratch-preparation-binding.json"
    if not binding_path.is_file() or binding_path.is_symlink():
        raise ValueError("prepared scratch binding is missing or unsafe")
    binding = json.loads(binding_path.read_text(encoding="utf-8"))
    scratch_root = audit / "scratch"
    execution_candidate = scratch_root / "candidate"
    requirements = scratch_root / "requirements"
    if (binding.get("kind") != "conventional_final_scratch_preparation" or
            Path(binding.get("frozenCandidatePath", "")).resolve(strict=True) != Path(candidate).resolve(strict=True) or
            Path(binding.get("frozenPublicRequirementsPath", "")).resolve(strict=True) != Path(initial_public).resolve(strict=True) or
            Path(binding.get("executionCandidatePath", "")).resolve(strict=True) != execution_candidate.resolve(strict=True) or
            Path(binding.get("executionRequirementsPath", "")).resolve(strict=True) != requirements.resolve(strict=True)):
        raise ValueError("prepared scratch binding paths do not match this assessment")
    if (binding.get("frozenCandidateManifest") != expected_candidate or
            binding.get("frozenCandidateManifestSha256") != _manifest_digest(expected_candidate) or
            binding.get("frozenPublicRequirementsManifest") != expected_public or
            binding.get("frozenPublicRequirementsManifestSha256") != _manifest_digest(expected_public)):
        raise ValueError("prepared scratch binding does not match expected frozen inputs")
    if _manifest(Path(candidate)) != expected_candidate or _manifest(Path(initial_public)) != expected_public:
        raise ValueError("frozen source input changed after scratch preparation")
    execution_manifest = _manifest(execution_candidate)
    if (_has_git_metadata(execution_candidate) or
            not _execution_matches_frozen(expected_candidate, execution_manifest) or
            _manifest(requirements) != expected_public):
        raise ValueError("prepared execution copy or requirements differ from frozen inputs")
    return {"audit": audit, "scratchRoot": scratch_root, "executionCandidate": execution_candidate,
            "requirements": requirements, "bindingPath": binding_path, "binding": binding,
            "executionManifest": execution_manifest}


def _post_assessment_check(candidate: Path, initial_public: Path, prepared: dict[str, Any],
                           candidate_before: dict[str, str], public_before: dict[str, str],
                           execution_before: dict[str, str], preparation_binding_sha256: str) -> dict[str, Any]:
    execution = _manifest(prepared["executionCandidate"])
    frozen_candidate_unchanged = _manifest(candidate) == candidate_before
    frozen_public_unchanged = _manifest(initial_public) == public_before
    requirements_unchanged = (_manifest(prepared["requirements"]) == public_before and
                               not _has_git_metadata(prepared["requirements"]))
    preparation_receipt_unchanged = (
        _sha256(prepared["bindingPath"].read_bytes()) == preparation_binding_sha256)
    outside_artifacts = {name: digest for name, digest in execution.items()
                         if PurePosixPath(name).parts[0] not in _ARTIFACT_ROOTS}
    before_outside = {name: digest for name, digest in execution_before.items()
                      if PurePosixPath(name).parts[0] not in _ARTIFACT_ROOTS}
    artifact_baseline_preserved = all(execution.get(name) == digest
                                      for name, digest in execution_before.items()
                                      if PurePosixPath(name).parts[0] in _ARTIFACT_ROOTS)
    return {
        "frozenCandidateUnchanged": frozen_candidate_unchanged,
        "frozenPublicRequirementsUnchanged": frozen_public_unchanged,
        "executionRequirementsUnchanged": requirements_unchanged,
        "preparationReceiptUnchanged": preparation_receipt_unchanged,
        "executionSemanticFilesUnchanged": outside_artifacts == before_outside and not _has_git_metadata(prepared["executionCandidate"]),
        "executionGitMetadataAbsent": not _has_git_metadata(prepared["executionCandidate"]),
        "executionArtifactBaselinePreserved": artifact_baseline_preserved,
        "executionCandidateManifestAfter": execution,
        "newAllowedArtifacts": sorted(name for name in execution
                                       if PurePosixPath(name).parts[0] in _ARTIFACT_ROOTS and
                                       name not in execution_before),
    }


def assess_scratch(plan: dict[str, Any], choice: dict[str, Any], executable: Path,
                   candidate: Path, audit: Path, expires: datetime, initial_public: Path,
                   expected_candidate_manifest: Mapping[str, str],
                   expected_public_manifest: Mapping[str, str]) -> dict[str, Any]:
    """Run one fresh AppServer assessment and preserve evidence separately.

    This does not translate a native terminal state into a scientific score.
    The assessor's report is preserved as its own artifact for later evaluation.
    """
    expected_candidate = _canonical_manifest(expected_candidate_manifest)
    expected_public = _canonical_manifest(expected_public_manifest)
    prepared = _verify_prepared(Path(candidate), Path(audit), Path(initial_public),
                                expected_candidate, expected_public)
    if expires.tzinfo is None:
        raise ValueError("assessment deadline must be timezone-aware")
    remaining = (expires - datetime.now(timezone.utc)).total_seconds()
    if remaining <= 0:
        return {"state": "NOT RUN", "reason": "final assessment deadline exhausted", "nativeStarts": 0}

    config = choice.get("config") if isinstance(choice, dict) else None
    if (not isinstance(config, dict) or config.get("backend") != "codex-app-server" or
            config.get("model") != "gpt-6-luna" or config.get("effort") != "high"):
        raise ValueError("supplement requires the bound codex-app-server gpt-6-luna/high runtime")
    if not Path(executable).is_absolute() or not Path(executable).is_file():
        raise ValueError("pinned native executable must be an existing absolute file")

    output = prepared["executionCandidate"] / ".assessment-output"
    expected_report = output / "final-assessment.md"
    if expected_report.exists() or expected_report.is_symlink():
        raise ValueError("assessment report path must be fresh")
    report_dir = prepared["audit"] / _ASSESSMENT_DIR
    if report_dir.exists() or report_dir.is_symlink():
        raise ValueError("supplemental assessment evidence already exists; replay is forbidden")
    report_dir.mkdir()

    options = {
        "sandbox": "workspace-write",
        "approvalPolicy": "never",
        "memoryEnabled": False,
        "nativeHelperModel": "gpt-6-luna",
        "nativeHelperEffort": "high",
        "nativeMaxConcurrentAgents": 1,
        "scopedGitApproval": False,
        "allowLoginShell": False,
        "windowsSandbox": "mxc",
    }
    command = [str(Path(executable).resolve(strict=True)), "-c", "agents.enabled=false",
               "-c", "features.multi_agent=false"]
    timeout = min(float(plan.get("freshFinalSeconds", 5400)), remaining)
    spec = {
        "backend": "codex-app-server",
        "command": command,
        "cwd": str(prepared["executionCandidate"].resolve(strict=True)),
        "model": "gpt-6-luna",
        "effort": "high",
        "runtimeOptions": options,
        "timeoutSeconds": timeout,
        "requestTimeoutSeconds": min(600.0, remaining),
    }
    prompt = (
        "You are the fresh, independent final assessor for the frozen Conventional candidate in this working directory. "
        f"The original public requirements and evaluation rules are at the read-only sibling path: {prepared['requirements'].resolve(strict=True)}. "
        "Treat those exact frozen bytes as the only requirements and rubric; also use the candidate's own current public "
        "project instructions. Run the complete existing test suite (the expected suite has 24 tests) and the additional "
        "original public cases that are explicitly described in those requirements. For Python tests use `python -B` "
        "or set PYTHONDONTWRITEBYTECODE=1; do not run py_compile or explicit cache-rewrite commands. Do not change, repair, generate, "
        "delete, or reformat candidate code, tests, expectations, docs, caches, or requirements. Do not use agents, "
        "helpers, or model fallbacks. Any database, temporary, or test output must stay below .scratch/ or "
        ".assessment-output/. Write a complete evidence-based report to .assessment-output/final-assessment.md, including "
        "commands, actual test counts/results, each public acceptance case, failures, and limitations. Distinguish PASS, "
        "FAIL, NOT RUN, and EVALUATION_ERROR. A native turn completing is transport evidence only, not product success "
        "or human acceptance. Do not disclose or rely on implementation transcripts or private feedback."
    )

    baseline_execution = prepared["executionManifest"]
    if not _execution_matches_frozen(expected_candidate, baseline_execution):
        raise ValueError("execution candidate changed semantically after preparation")
    started_at = _utc_now()
    binding = {
        "schema": 1,
        "kind": "conventional_final_scratch_assessment",
        "authority": plan.get("id"),
        "startedAt": started_at,
        "sourcePreparationBinding": str(prepared["bindingPath"]),
        "sourcePreparationBindingSha256": _sha256(prepared["bindingPath"].read_bytes()),
        "frozenCandidatePath": str(Path(candidate).resolve(strict=True)),
        "frozenCandidateManifest": expected_candidate,
        "executionCandidatePath": spec["cwd"],
        "executionCandidateBaselineManifest": baseline_execution,
        "publicRequirementsPath": str(Path(initial_public).resolve(strict=True)),
        "publicRequirementsManifest": expected_public,
        "executionRequirementsPath": str(prepared["requirements"].resolve(strict=True)),
        "executionRequirementsManifest": _manifest(prepared["requirements"]),
        "writableRoot": spec["cwd"],
        "writableRoots": [],
        "runtimeOptions": options,
        "nativeEnvironmentPolicy": _NATIVE_ENVIRONMENT_POLICY,
        "sourceScope": "fresh independent final assessor; frozen candidate and public requirements only",
        "startRequests": 1,
        "nativeCanaries": 0,
        "implementationStarts": 0,
        "helpers": 0,
        "humanAcceptance": "not established",
    }
    preparation_binding_sha256 = _sha256(prepared["bindingPath"].read_bytes())
    (report_dir / "binding.json").write_text(json.dumps(binding, indent=2) + "\n", encoding="utf-8")
    (report_dir / "prompt.txt").write_text(prompt, encoding="utf-8")

    no_helpers_observed = True

    def emit(event: dict[str, Any]) -> None:
        nonlocal no_helpers_observed
        if not isinstance(event, dict):
            event = {"event": str(event)}
        if _helper_activity_started(event):
            no_helpers_observed = False
        record = {"observedAt": _utc_now(), **event}
        with (report_dir / "events.jsonl").open("ab") as stream:
            stream.write((json.dumps(record, ensure_ascii=False, sort_keys=True) + "\n").encode("utf-8"))
            stream.flush()

    from conventional.backends import run
    bytecode_env_was_present = "PYTHONDONTWRITEBYTECODE" in os.environ
    previous_bytecode_env = os.environ.get("PYTHONDONTWRITEBYTECODE")
    os.environ["PYTHONDONTWRITEBYTECODE"] = "1"
    try:
        try:
            native_result = run(spec, prompt, None, emit, threading.Event())
            if not isinstance(native_result, dict):
                native_result = {"state": "uncertain", "detail": "backend returned a non-object result"}
        except Exception as exc:
            native_result = {"state": "uncertain", "detail": f"{type(exc).__name__}: {exc}",
                             "replay": "forbidden"}
    finally:
        if bytecode_env_was_present:
            assert previous_bytecode_env is not None
            os.environ["PYTHONDONTWRITEBYTECODE"] = previous_bytecode_env
        else:
            os.environ.pop("PYTHONDONTWRITEBYTECODE", None)
    native_state = native_result.get("state")
    try:
        boundary = _post_assessment_check(Path(candidate), Path(initial_public), prepared,
                                          expected_candidate, expected_public, baseline_execution,
                                          preparation_binding_sha256)
    except Exception as exc:
        boundary = {
            "frozenCandidateUnchanged": False,
            "frozenPublicRequirementsUnchanged": False,
            "executionRequirementsUnchanged": False,
            "preparationReceiptUnchanged": False,
            "executionSemanticFilesUnchanged": False,
            "executionArtifactBaselinePreserved": False,
            "newAllowedArtifacts": [],
            "boundaryError": f"{type(exc).__name__}: {exc}",
        }
    report_text = None
    if expected_report.is_file() and not expected_report.is_symlink():
        try:
            report_text = expected_report.read_text(encoding="utf-8")
            (report_dir / "scientific-assessment-report.md").write_text(report_text, encoding="utf-8")
        except (OSError, UnicodeError) as exc:
            boundary["scientificReportReadError"] = f"{type(exc).__name__}: {exc}"
            report_text = None
    else:
        boundary["scientificAssessmentReportPresent"] = False
    if report_text is not None:
        boundary["scientificAssessmentReportPresent"] = bool(report_text.strip())

    result = {
        "schema": 1,
        "state": native_state if native_state in {"completed", "failed", "cancelled", "uncertain", "needs_input"}
                else "uncertain",
        "nativeState": native_state,
        "nativeResult": native_result,
        "nativeEnvironmentPolicy": _NATIVE_ENVIRONMENT_POLICY,
        "scientificJudgment": "PENDING INDEPENDENT REVIEW OF PRESERVED REPORT",
        "scientificReportPath": str(report_dir / "scientific-assessment-report.md") if report_text is not None else None,
        "candidateFilesUnchanged": boundary["frozenCandidateUnchanged"] and boundary["executionSemanticFilesUnchanged"],
        "publicRequirementsUnchanged": boundary["frozenPublicRequirementsUnchanged"],
        "executionRequirementsUnchanged": boundary["executionRequirementsUnchanged"],
        "preparationReceiptUnchanged": boundary["preparationReceiptUnchanged"],
        "noHelpersObserved": no_helpers_observed,
        "executionArtifactBaselinePreserved": boundary["executionArtifactBaselinePreserved"],
        "allowedArtifacts": boundary["newAllowedArtifacts"],
        "boundaryChecks": boundary,
        "endedAt": _utc_now(),
        "humanAcceptance": "not established",
    }
    boundary_ok = all((boundary["frozenCandidateUnchanged"],
                       boundary["frozenPublicRequirementsUnchanged"],
                       boundary["executionRequirementsUnchanged"],
                       boundary["preparationReceiptUnchanged"],
                       boundary["executionSemanticFilesUnchanged"],
                       boundary["executionArtifactBaselinePreserved"]))
    if native_state != "completed":
        result["reason"] = "native assessor did not complete with a terminal report"
    elif not boundary_ok:
        result["nativeState"] = native_state
        result.update(state="failed", reason="frozen inputs or semantic execution files changed")
    elif not boundary.get("scientificAssessmentReportPresent"):
        result["nativeState"] = native_state
        result.update(state="failed", reason="native turn completed without a preserved scientific report")
    elif not no_helpers_observed:
        result["nativeState"] = native_state
        result.update(state="failed", reason="native helper or collaboration activity was observed")
    (report_dir / "result.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    return result
