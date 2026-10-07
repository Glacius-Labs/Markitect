"""Prepare a disposable positive Government G5 native-integration fixture.

This module constructs exact public inputs and reviewable argv. It never starts
Markitect, Python role processes, an Actor, or a provider. The deterministic
delegate is a protocol/mechanics fixture; its receipts cannot establish model
quality, human acceptance, or product-study readiness.
"""
from __future__ import annotations

import hashlib
import json
import math
import os
from pathlib import Path
import re
import stat
import subprocess

import government
import government_roles

FIXTURE_ID = "government-native-positive-v1"
FIXTURE_ROOT = Path(__file__).with_name("fixtures") / "government_positive"
MANAGED_REF = "refs/markitect/government/active/scientist-positive"
PENDING_CONSTITUTION = "REPLACE_FROM_GOVERNMENT_INSPECT"
TASK_ID = "positive-overflow-release"
MAX_ROLE_CALLS = 12
QUEUE_ROLE_STARTS = 3
MAX_REPAIRS = 0
MAX_WALL_SECONDS = 40
MAX_PARALLELISM = 2
ROLE_TIMEOUT_SECONDS = 30
CHECK_TIMEOUT_SECONDS = 30
FIXTURE_AUTHORIZATION = {
    "sourceGrantKey": "native-s1-integration-fixtures-20261008",
    "classification": "nativeFixture",
    "providerUse": "noProvider",
}
_SHA256 = re.compile(r"^[0-9a-f]{64}$")
_CONSTITUTION = re.compile(r"^sha256:[0-9a-f]{64}$")

SLOTS = (
    ("positive-root-executor", "execute", "executor", "executor"),
    ("positive-root-verifier", "review", "verifier", "verifier"),
    ("positive-ressort-correctness", "vote", "verifier", "ressort"),
)

# Native role wrappers import these modules through government_roles.py and the
# bound Scientist Authority. Keeping this list explicit makes the runtime pin
# reviewable; Python stdlib files are supplied by the pinned interpreter.
BRIDGE_FILES = (
    "government_roles.py", "native_controller.py", "government.py",
    "government-pin.json", "dispatch.py", "identity_probe.py", "ledger.py",
    "process.py", "runner.py", "measurement_profile.py",
    "context_allocation.py", "context_tools_allocation.py", "native_fixture_budget.py",
)


def digest(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


def native_digest(raw: bytes) -> str:
    """G5 RuntimeFile and agentexec digests include the literal sha256: prefix."""
    return "sha256:" + digest(raw)


def canonical_json(value) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")


def _run_git(repo: Path, *args: str) -> str:
    return subprocess.check_output(["git", "-C", str(repo), *args], text=True,
                                   stderr=subprocess.PIPE, timeout=10).strip()


def _fixture_marker(repo: Path) -> tuple[Path, dict]:
    repo = repo.resolve(strict=True)
    marker_path = repo / ".git" / "scientist-government-fixture.json"
    if not marker_path.is_file() or marker_path.is_symlink():
        raise ValueError("repository is not a marked Scientist Government disposable fixture")
    marker = json.loads(marker_path.read_bytes())
    if marker.get("fixtureId") != FIXTURE_ID or Path(marker.get("repository", "")).resolve() != repo:
        raise ValueError("disposable Government fixture marker binding mismatch")
    return marker_path, marker


def _outside_study_checkout(path: Path) -> None:
    checkout = Path(__file__).resolve().parents[3]
    resolved = path.resolve()
    if resolved == checkout or checkout in resolved.parents or resolved in checkout.parents:
        raise ValueError("disposable Actor repository must stay outside the study checkout hierarchy")


def create_disposable_repository(destination: str | Path) -> dict:
    """Create and commit the public fixture; Order remains unbound until inspect.

    The created active ref starts at the exact initial commit. A later explicit
    `bind_order_to_inspection` commit replaces the placeholder using the digest
    printed by the accepted binary's read-only inspect action.
    """
    repo = Path(destination)
    if not repo.is_absolute():
        raise ValueError("absolute disposable repository path required")
    _outside_study_checkout(repo)
    if repo.exists() or repo.is_symlink():
        raise FileExistsError("refusing to reuse an existing disposable fixture path")
    repo.parent.mkdir(parents=True, exist_ok=True)
    repo.mkdir()
    for relative in ("README.md", "government.yaml", "order.yaml", "go.mod",
                     "inventory/reservation.go", "inventory/reservation_test.go"):
        source = FIXTURE_ROOT / relative
        target = repo / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(source.read_bytes())
    _run_git(repo, "init", "-b", "main")
    _run_git(repo, "config", "user.name", "Scientist Government disposable fixture")
    _run_git(repo, "config", "user.email", "government-fixture@example.invalid")
    _run_git(repo, "config", "core.autocrlf", "false")
    _run_git(repo, "add", "--all")
    _run_git(repo, "-c", "commit.gpgsign=false", "commit", "-m", "Initial Government positive fixture")
    base = _run_git(repo, "rev-parse", "--verify", "HEAD^{commit}")
    _run_git(repo, "update-ref", MANAGED_REF, base)
    marker = {"fixtureId": FIXTURE_ID, "repository": str(repo.resolve()), "managedRef": MANAGED_REF,
              "baseCommit": base, "constitutionDigest": None, "orderSha256": digest((repo / "order.yaml").read_bytes())}
    marker_path = repo / ".git" / "scientist-government-fixture.json"
    marker_path.write_bytes(canonical_json(marker) + b"\n")
    if _run_git(repo, "status", "--porcelain"):
        raise AssertionError("fixture preparation left a dirty repository")
    return marker


def bind_order_to_inspection(repo: str | Path, constitution_digest: str) -> dict:
    """Bind the Order to one real inspect result and advance only the fixture ref."""
    repo = Path(repo).resolve(strict=True)
    _outside_study_checkout(repo)
    marker_path, marker = _fixture_marker(repo)
    if not _CONSTITUTION.fullmatch(constitution_digest):
        raise ValueError("exact sha256:<64 lowercase hex> inspect result required")
    if marker.get("constitutionDigest") is not None:
        raise ValueError("fixture Order is already bound; refusing to create another baseline")
    if _run_git(repo, "rev-parse", "--verify", "HEAD^{commit}") != marker.get("baseCommit"):
        raise ValueError("disposable fixture baseline changed before Order binding")
    if _run_git(repo, "rev-parse", "--verify", f"{MANAGED_REF}^{{commit}}") != marker.get("baseCommit"):
        raise ValueError("managed active ref moved before Order binding")
    if _run_git(repo, "status", "--porcelain"):
        raise ValueError("disposable fixture must be clean before Order binding")
    order_path = repo / "order.yaml"
    order = order_path.read_text(encoding="utf-8")
    if order.count(PENDING_CONSTITUTION) != 1:
        raise ValueError("Order must contain exactly one initial Constitution placeholder")
    order_path.write_text(order.replace(PENDING_CONSTITUTION, constitution_digest), encoding="utf-8", newline="\n")
    _run_git(repo, "add", "--", "order.yaml")
    _run_git(repo, "-c", "commit.gpgsign=false", "commit", "-m", "Bind fixture Order to inspected Constitution")
    new_base = _run_git(repo, "rev-parse", "--verify", "HEAD^{commit}")
    _run_git(repo, "update-ref", MANAGED_REF, new_base, marker["baseCommit"])
    marker["baseCommit"] = new_base
    marker["constitutionDigest"] = constitution_digest
    marker["orderSha256"] = digest(order_path.read_bytes())
    marker_path.write_bytes(canonical_json(marker) + b"\n")
    if _run_git(repo, "status", "--porcelain"):
        raise AssertionError("Order binding left a dirty fixture repository")
    return marker


def _mode(path: Path) -> str:
    permissions = stat.S_IMODE(path.stat().st_mode)
    if os.name == "nt":
        return "0644" if permissions & stat.S_IWUSR else "0444"
    return f"{permissions:04o}"


def runtime_file(path: str | Path) -> dict:
    target = Path(path)
    if not target.is_absolute() or target.is_symlink():
        raise ValueError("native RuntimeFile must be an absolute non-symlink file")
    target = target.resolve(strict=True)
    if not target.is_file():
        raise ValueError("native RuntimeFile must be a regular file")
    return {"path": str(target), "mode": _mode(target), "digest": native_digest(target.read_bytes())}


def _delegate_slot(python_executable: Path, delegate_path: Path, slot_id: str,
                   phase: str, response_role: str) -> dict:
    argv = [str(python_executable), str(delegate_path), "--phase", phase]
    command_digest = runtime_file(python_executable)["digest"]
    delegate_files = [runtime_file(python_executable), runtime_file(delegate_path)]
    model = "deterministic-government-fixture"
    model_options = {}
    provider_version = "python-fixed-role-response-v1"
    return {
        "slotId": slot_id,
        "phase": phase,
        "responseRole": response_role,
        "wrapper": {"command": str(python_executable)},
        "delegate": {
            "argv": argv,
            "commandDigest": command_digest,
            "runtimeFiles": delegate_files,
            "model": model,
            "modelOptions": model_options,
            "providerVersion": provider_version,
            "timeoutSeconds": ROLE_TIMEOUT_SECONDS,
            "maxStdoutBytes": 1_048_576,
            "maxStderrBytes": 1_048_576,
        },
    }


def build_role_authorization(*, trial_id: str, dispatch_id: str, request_path: str | Path,
                             ledger_path: str | Path, runtime_path: str | Path,
                             role_evidence_directory: str | Path, expires_at: int | float,
                             python_executable: str | Path, delegate_path: str | Path,
                             status: str = "pending",
                             fixture_authorization: dict | None = None) -> dict:
    """Build the role-authorization payload; pending is the safe default.

    A caller must supply live operator-authorized identity/expiry values. This
    helper only describes the proposed slots and never grants Authority.
    """
    paths = [Path(value) for value in (request_path, ledger_path, runtime_path, role_evidence_directory)]
    if any(not path.is_absolute() for path in paths):
        raise ValueError("request, ledger, runtime and role evidence paths must be absolute")
    if (not trial_id or not dispatch_id or type(expires_at) not in (int, float) or
            not math.isfinite(expires_at) or expires_at <= 0):
        raise ValueError("bound trial/dispatch identity and finite expiry required")
    if status not in {"pending", "approved"}:
        raise ValueError("unsupported role authorization status")
    if status == "approved" and fixture_authorization != FIXTURE_AUTHORIZATION:
        raise ValueError("approved fixture role authorization requires the exact parent source grant")
    if status == "pending" and fixture_authorization is not None:
        raise ValueError("pending role authorization cannot carry fixture approval provenance")
    python = Path(python_executable).resolve(strict=True)
    delegate = Path(delegate_path).resolve(strict=True)
    slots = [_delegate_slot(python, delegate, slot, phase, role)
             for slot, phase, role, _kind in SLOTS]
    authorization = {
        "apiVersion": "markitect.scientist-role-authorization/v1alpha1",
        "status": status,
        "trialId": trial_id,
        "dispatchId": dispatch_id,
        "taskId": TASK_ID,
        "requestPath": str(paths[0].resolve()),
        "ledgerPath": str(paths[1].resolve()),
        "runtimePath": str(paths[2].resolve()),
        "roleEvidenceDirectory": str(paths[3].resolve()),
        "expiresAt": expires_at,
        "maxCalls": MAX_ROLE_CALLS,
        "slots": slots,
    }
    if status == "approved":
        # The parent-supplied grant is incorporated before serialization and
        # runtime hashing; callers must not mutate these bytes after freezing.
        authorization["fixtureAuthorization"] = dict(fixture_authorization)
    return authorization


def _native_role_runtime(*, slot_id: str, phase: str, response_role: str,
                         python_executable: Path, wrapper_path: Path,
                         authorization_path: Path, authorization_sha256: str,
                         role_evidence_directory: Path, bridge_files: list[dict],
                         authorization_files: list[dict], delegate_path: Path) -> dict:
    args = government_roles.wrapper_arguments(
        str(wrapper_path), str(authorization_path), authorization_sha256,
        str(role_evidence_directory), slot_id)
    runtime_files = {item["path"]: item for item in [*bridge_files, *authorization_files,
                                                       runtime_file(python_executable), runtime_file(delegate_path)]}
    return {
        "slotId": slot_id,
        "command": str(python_executable),
        "args": args,
        "model": "deterministic-government-fixture",
        "modelOptions": {},
        "providerVersion": "python-fixed-role-response-v1",
        "timeoutSeconds": ROLE_TIMEOUT_SECONDS,
        "maxStdoutBytes": 1_048_576,
        "maxStderrBytes": 1_048_576,
        "runtimeFiles": [runtime_files[path] for path in sorted(runtime_files)],
    }


def build_runtime(*, repository: str | Path, base_commit: str,
                  queue_state_directory: str | Path, run_state_directory: str | Path,
                  temporary_directory: str | Path, runtime_path: str | Path,
                  authorization_path: str | Path, authorization_raw: bytes,
                  python_executable: str | Path, delegate_path: str | Path,
                  expected_constitution_digest: str) -> dict:
    """Create the G5 RuntimeSpec after the parent has authored role authority."""
    repo = Path(repository).resolve(strict=True)
    queue_state = Path(queue_state_directory).resolve()
    run_state = Path(run_state_directory).resolve()
    temporary = Path(temporary_directory).resolve()
    auth_path = Path(authorization_path).resolve(strict=True)
    runtime_target = Path(runtime_path).resolve()
    auth = json.loads(authorization_raw)
    if auth.get("status") != "approved":
        raise ValueError("native Government Runtime requires actual approved operator role authorization")
    if auth.get("fixtureAuthorization") != FIXTURE_AUTHORIZATION:
        raise ValueError("native Government Runtime requires the exact parent fixture authorization grant")
    if auth_path.read_bytes() != authorization_raw:
        raise ValueError("role authorization path bytes differ from supplied bytes")
    if auth.get("runtimePath") != str(runtime_target):
        raise ValueError("role authorization runtime path differs from output path")
    if auth.get("roleEvidenceDirectory") is None or auth.get("maxCalls") != MAX_ROLE_CALLS:
        raise ValueError("operator role authorization does not match the finite fixture bounds")
    marker_path, marker = _fixture_marker(repo)
    if marker.get("constitutionDigest") != expected_constitution_digest:
        raise ValueError("fixture Order and expected inspected Constitution digest differ")
    if marker.get("baseCommit") != base_commit:
        raise ValueError("Government Runtime base must equal the bound disposable fixture commit")
    for directory in (queue_state, run_state, temporary):
        if directory == repo or directory in repo.parents or repo in directory.parents:
            raise ValueError("Government queue/run/temp state must be outside the Actor repository")
    if len({queue_state, run_state, temporary}) != 3:
        raise ValueError("Government queue/run/temp state directories must be separate")

    python = Path(python_executable).resolve(strict=True)
    delegate = Path(delegate_path).resolve(strict=True)
    wrapper = Path(government_roles.__file__).resolve(strict=True)
    helper_dir = Path(__file__).resolve().parent
    bridge_files = [runtime_file(helper_dir / name) for name in BRIDGE_FILES]
    bridge_files.append(runtime_file(helper_dir / ".." / "public" / "resource-proposal.json"))
    authorization_files = [runtime_file(wrapper), runtime_file(helper_dir / "native_controller.py"),
                           runtime_file(auth_path)]
    auth_sha = digest(authorization_raw)
    slots = auth.get("slots")
    expected_slots = [slot for slot, _phase, _role, _kind in SLOTS]
    if not isinstance(slots, list) or [item.get("slotId") for item in slots] != expected_slots:
        raise ValueError("approved role authorization must cover the exact three fixture slots in order")
    expected_authorized_slots = [
        _delegate_slot(python, delegate, slot_id, phase, role)
        for slot_id, phase, role, _kind in SLOTS
    ]
    if slots != expected_authorized_slots:
        raise ValueError("approved role authorization delegate bindings differ from the fixed fixture slots")
    if not all(path.is_dir() and not path.is_symlink() for path in (queue_state, run_state, temporary)):
        raise ValueError("queue, run-state, and temporary directories must already exist as separate directories")
    role_evidence = Path(auth["roleEvidenceDirectory"]).resolve()
    if not role_evidence.is_absolute() or role_evidence == repo or role_evidence in repo.parents or repo in role_evidence.parents:
        raise ValueError("role evidence directory must be absolute and separate from the Actor repository")
    configured = []
    for slot_id, phase, response_role, _kind in SLOTS:
        configured.append(_native_role_runtime(
            slot_id=slot_id, phase=phase, response_role=response_role,
            python_executable=python, wrapper_path=wrapper,
            authorization_path=auth_path, authorization_sha256=auth_sha,
            role_evidence_directory=role_evidence,
            bridge_files=bridge_files, authorization_files=authorization_files,
            delegate_path=delegate))
    if (len(configured[0]["runtimeFiles"]) > 32 or
            any(item["digest"] != runtime_file(Path(item["path"]))["digest"] for item in configured[0]["runtimeFiles"])):
        raise ValueError("G5 RuntimeFile set exceeds its count bound or changed during preparation")
    return {
        "apiVersion": government.RUN_API,
        "activeRef": MANAGED_REF,
        "expectedBase": base_commit,
        "timeoutSeconds": MAX_WALL_SECONDS,
        "stateDirectory": str(run_state),
        "temporaryDirectory": str(temporary),
        "executor": configured[0],
        "verifier": configured[1],
        "ressorts": [{
            "ressort": {"apiVersion": "markitect.government/v1alpha1", "kind": "Ressort",
                        "namespace": "inventory", "name": "correctness"},
            "runner": configured[2],
        }],
        "checks": [{"name": "inventory-overflow", "run": ["go", "test", "./inventory", "-count=1"],
                    "timeoutSeconds": CHECK_TIMEOUT_SECONDS}],
    }


def build_backlog(*, runtime_path: str | Path, queue_state_directory: str | Path,
                  task_id: str = TASK_ID) -> dict:
    runtime_path = Path(runtime_path).resolve()
    queue_state = Path(queue_state_directory).resolve()
    if not runtime_path.is_absolute() or not queue_state.is_absolute():
        raise ValueError("absolute Government runtime and queue state paths required")
    return {
        "apiVersion": government.QUEUE_API,
        "stateDirectory": str(queue_state),
        "limits": {"actorStarts": QUEUE_ROLE_STARTS, "maxRepairs": MAX_REPAIRS,
                   "maxWallTimeSeconds": MAX_WALL_SECONDS, "maxParallelism": MAX_PARALLELISM},
        "jobs": [{"id": task_id, "dependsOn": [], "configPath": "government.yaml",
                  "orderPath": "order.yaml", "runtimePath": str(runtime_path)}],
    }


def product_binding(*, repository: str | Path, runtime_path: str | Path, runtime_raw: bytes,
                    backlog_path: str | Path, backlog_raw: bytes,
                    authorization_path: str | Path, authorization_raw: bytes,
                    queue_state_directory: str | Path) -> tuple[dict, list[dict]]:
    """Return Request.product.government and its exact external releasedInputs."""
    pin = government.PIN["accepted"]
    handoff = Path(pin["handoff"]["path"]).resolve(strict=True)
    executable = Path(pin["binary"]["path"]).resolve(strict=True)
    if digest(handoff.read_bytes()) != pin["handoff"]["sha256"]:
        raise ValueError("accepted G5 handoff bytes differ from the pinned SHA-256")
    if digest(executable.read_bytes()) != pin["binary"]["sha256"]:
        raise ValueError("accepted G5 executable bytes differ from the pinned SHA-256")
    for path, raw, label in ((runtime_path, runtime_raw, "Government runtime"),
                             (backlog_path, backlog_raw, "Government backlog"),
                             (authorization_path, authorization_raw, "role authorization")):
        if Path(path).resolve(strict=True).read_bytes() != raw:
            raise ValueError(f"{label} path bytes differ from the supplied exact bytes")
    repo = Path(repository).resolve(strict=True)
    bindings = {
        "handoff": {"path": str(handoff), "sha256": pin["handoff"]["sha256"]},
        "executable": {"path": str(executable), "sha256": pin["binary"]["sha256"],
                       "sourceCommit": pin["sourceCommit"]},
        "projectConfig": {"path": "government.yaml", "sha256": digest((repo / "government.yaml").read_bytes())},
        "order": {"path": "order.yaml", "sha256": digest((repo / "order.yaml").read_bytes())},
        "runtime": {"path": str(Path(runtime_path).resolve()), "sha256": digest(runtime_raw)},
        "backlog": {"path": str(Path(backlog_path).resolve()), "sha256": digest(backlog_raw)},
        "roleAuthorization": {"path": str(Path(authorization_path).resolve()),
                              "sha256": digest(authorization_raw)},
        "queueStateDirectory": str(Path(queue_state_directory).resolve()),
    }
    released = [bindings[name] for name in ("handoff", "runtime", "backlog", "roleAuthorization")]
    if any(item["path"] in {str(repo), str(repo.resolve())} for item in released):
        raise ValueError("Government external inputs cannot be inside the disposable Actor repository")
    return bindings, released


def inspect_argv(markitect_executable: str | Path, repository: str | Path) -> list[str]:
    """Review-only argv for the one required Constitution-digest read."""
    executable = Path(markitect_executable).resolve(strict=True)
    return [str(executable), "government", "--repo", str(Path(repository).resolve()),
            "--config", "government.yaml", "--action", "inspect"]


def queue_argv(markitect_executable: str | Path, repository: str | Path,
               backlog_path: str | Path) -> list[str]:
    return government.queue_argv(markitect_executable, repository, backlog_path)


def resume_argv(markitect_executable: str | Path, repository: str | Path,
                backlog_path: str | Path, queue_directory: str | Path) -> list[str]:
    return government.resume_argv(markitect_executable, repository, backlog_path, queue_directory)


def expected_queue_role_calls() -> dict:
    return {"executor": 1, "independentVerifier": 1, "selectedRessortVotes": 1,
            "queueTotal": QUEUE_ROLE_STARTS, "resumeExpectedAdditional": 0,
            "interpretation": "Deterministic provider-free role processes exercise native G5 receipts only."}
