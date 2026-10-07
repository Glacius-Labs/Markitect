"""Prepare and normalize a bounded Classic v0.14.1 controller fixture.

This module prepares a disposable source fixture and exact native command plan.
It does not start Markitect, agentexec, Codex, or a provider. The shared
Scientist wrapper is the native RunnerSpec; the pinned packet protocol
test-double is only an explicitly authorized delegate and is not semantic
evidence or human acceptance.
"""
from __future__ import annotations

import hashlib
import json
import math
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

import classic

PACKET = Path(json.loads(Path(__file__).with_name("classic-pin.json").read_text(encoding="utf-8"))["packet"])
CLASSIC_SOURCE = "7dbd599c81540c8203a1b7f83afbc335174f4f1f"
DOTNET_PROJECTION = '["markitect.foundation/v1","Projection","commerce","application-dotnet"]'
MARKDOWN_PROJECTION = '["markitect.foundation/v1","Projection","commerce","application-markdown"]'
SCOPES = {
    "commerce-markdown": {"projectionId": MARKDOWN_PROJECTION, "children": ["commerce-dotnet"]},
    "commerce-dotnet": {"projectionId": DOTNET_PROJECTION, "children": []},
}
ROLE_SLOTS = {
    ("executor", DOTNET_PROJECTION): ("execute", "classic/execute/commerce-dotnet", "commerce-dotnet"),
    ("verifier", DOTNET_PROJECTION): ("review", "classic/review/commerce-dotnet", "commerce-dotnet"),
    ("verifier", MARKDOWN_PROJECTION): ("review", "classic/review/commerce-markdown", "commerce-markdown"),
}
FLOW_ACTIONS = ("execute", "apply", "verify", "audit", "apply-replay")
EXPECTED_STATUSES = {"execute": "planned", "apply": "materialized-unverified",
                    "verify": "passed", "audit": "complete"}
TASK_ID = "native-classic-positive"
CONTROLLER_ACTIONS = ["execute", "apply", "verify", "audit", "apply-replay"]
BRIDGE_FILES = ("government_roles.py", "native_controller.py", "government.py", "classic.py",
                "classic_integration.py", "classic-pin.json", "dispatch.py", "identity_probe.py",
                "ledger.py", "process.py", "runner.py", "measurement_profile.py",
                "context_allocation.py", "context_tools_allocation.py", "diagnostic_history.py",
                "native_fixture_budget.py")


def sha256(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


def _json_bytes(value) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")


def bind_request(request: dict, request_path: str | os.PathLike[str] | None = None, *,
                 allow_materialized_workspace: bool = False) -> dict:
    """Validate frozen Classic bindings before roles; optionally permit post-Apply files.

    The caller may set ``allow_materialized_workspace`` only for the same
    already-claimed controller Request after native Apply has materialized the
    reviewed plan. The requested base HEAD and all static product pins remain
    mandatory in either mode.
    """
    if type(allow_materialized_workspace) is not bool:
        raise ValueError("explicit materialized-workspace boolean required")
    if not isinstance(request, dict) or request.get("arm") != "classic" or request.get("operation") != "run_task":
        raise ValueError("Classic run_task Request required")
    if request.get("task", {}).get("id") != TASK_ID:
        raise ValueError("Classic Request must select only the native fixture positive task")
    repo = Path(request.get("actorRepository", ""))
    if not repo.is_absolute():
        raise ValueError("absolute Classic actorRepository required")
    repo = repo.resolve(strict=True)
    evidence = Path(request.get("evidenceDirectory", ""))
    if not evidence.is_absolute():
        raise ValueError("absolute Classic evidenceDirectory required")
    evidence = evidence.resolve()
    if evidence == repo or repo in evidence.parents or evidence in repo.parents:
        raise ValueError("Classic Actor and evidence roots must be separate")

    product = request.get("product")
    classic_product = product.get("classic") if isinstance(product, dict) else None
    if not isinstance(classic_product, dict):
        raise ValueError("Request.product.classic bindings required")
    if classic_product.get("controllerActions") != CONTROLLER_ACTIONS:
        raise ValueError("Classic Request must bind the exact finite controller action sequence")
    packet_binding = classic_product.get("packet")
    if not isinstance(packet_binding, dict) or not isinstance(packet_binding.get("path"), str):
        raise ValueError("frozen Classic packet path and manifest binding required")
    inspection = classic.inspect_packet(packet_binding["path"])
    pin = json.loads(Path(__file__).with_name("classic-pin.json").read_text(encoding="utf-8"))
    packet_path = Path(inspection["packetPath"]).resolve(strict=True)
    manifest = packet_path / "checksums.sha256"
    if (packet_path != Path(pin["packet"]).resolve() or
            sha256(manifest.read_bytes()) != packet_binding.get("manifestSha256")):
        raise ValueError("Classic packet is not the exact externally pinned readiness packet")

    executable_binding = classic_product.get("executable")
    if not isinstance(executable_binding, dict) or not isinstance(executable_binding.get("path"), str):
        raise ValueError("Classic product executable binding required")
    executable = Path(executable_binding["path"]).resolve(strict=True)
    if (executable != Path(inspection["binary"]["path"]).resolve(strict=True) or
            sha256(executable.read_bytes()) != executable_binding.get("sha256") or
            executable_binding.get("sha256") != classic.EXPECTED_BINARY_SHA256 or
            executable_binding.get("sourceCommit") != classic.EXPECTED_SOURCE or
            executable_binding.get("heldSourceCommit") != classic.EXPECTED_HELD_SOURCE):
        raise ValueError("Classic source/binary binding differs from the pinned release packet")

    config_binding = classic_product.get("projectConfig")
    if not isinstance(config_binding, dict) or config_binding.get("path") != "examples/canonical-projection/canonical.yaml":
        raise ValueError("pinned Classic canonical project configuration required")
    config = (repo / config_binding["path"]).resolve(strict=True)
    if repo not in config.parents or sha256(config.read_bytes()) != config_binding.get("sha256"):
        raise ValueError("Classic project configuration is outside Actor or differs from the bound fixture config")

    runtime_binding = classic_product.get("runtime")
    if not isinstance(runtime_binding, dict) or not isinstance(runtime_binding.get("path"), str):
        raise ValueError("external Classic runtime binding required")
    runtime = Path(runtime_binding["path"]).resolve(strict=True)
    if sha256(runtime.read_bytes()) != runtime_binding.get("sha256"):
        raise ValueError("Classic runtime digest mismatch")
    runtime_value = json.loads(runtime.read_bytes())
    if runtime_value.get("apiVersion") != "markitect.canonical/controller/v1alpha1":
        raise ValueError("Classic controller runtime API mismatch")
    if ([item.get("id") for item in runtime_value.get("assuranceScopes", [])] !=
            ["commerce-markdown", "commerce-dotnet"] or
            runtime_value.get("assuranceRoots") != ["commerce-markdown"]):
        raise ValueError("Classic runtime scope graph differs from the pinned public fixture")
    roles = configured_roles(runtime_value)

    authorization_binding = classic_product.get("roleAuthorization")
    if not isinstance(authorization_binding, dict) or not isinstance(authorization_binding.get("path"), str):
        raise ValueError("external Classic role authorization binding required")
    authorization = Path(authorization_binding["path"]).resolve(strict=True)
    authorization_raw = authorization.read_bytes()
    if sha256(authorization_raw) != authorization_binding.get("sha256"):
        raise ValueError("Classic role authorization digest mismatch")
    released = {str(Path(item["path"]).resolve()): item.get("sha256")
                for item in request.get("releasedInputs", []) if isinstance(item, dict) and item.get("path")}
    for label, path, expected in (("runtime", runtime, runtime_binding["sha256"]),
                                  ("role authorization", authorization, authorization_binding["sha256"])):
        if path in (repo, evidence) or repo in path.parents or evidence in path.parents or released.get(str(path)) != expected:
            raise ValueError(f"Classic {label} must be a separately released input outside Actor/evidence")
    auth_value = json.loads(authorization_raw)
    if (auth_value.get("apiVersion") != "markitect.scientist-role-authorization/v1alpha1" or
            auth_value.get("status") != "approved" or auth_value.get("trialId") != request.get("trialId") or
            auth_value.get("dispatchId") != request.get("dispatchId") or
            auth_value.get("taskId") != request.get("task", {}).get("id") or
            Path(auth_value.get("runtimePath", "")).resolve() != runtime):
        raise ValueError("Classic role authorization does not bind this Request and runtime")
    if request_path is not None and Path(auth_value.get("requestPath", "")).resolve() != Path(request_path).resolve():
        raise ValueError("Classic role authorization Request path binding mismatch")
    expected_slots = {item["slotId"] for item in roles}
    auth_slots = auth_value.get("slots")
    if not isinstance(auth_slots, list) or {item.get("slotId") for item in auth_slots if isinstance(item, dict)} != expected_slots:
        raise ValueError("Classic role authorization slots differ from the configured native role set")

    try:
        head = subprocess.run(["git", "-C", str(repo), "rev-parse", "--verify", "HEAD^{commit}"],
                              capture_output=True, timeout=10, check=True).stdout.decode().strip()
        status = subprocess.run(["git", "-C", str(repo), "status", "--porcelain", "--untracked-files=all"],
                                capture_output=True, timeout=10, check=True).stdout
    except (OSError, subprocess.SubprocessError) as exc:
        raise ValueError("Classic Actor must be a readable Git fixture") from exc
    if head != request.get("baseCommit"):
        raise ValueError("Classic Actor HEAD must remain the exact Request baseCommit")
    if status and not allow_materialized_workspace:
        raise ValueError("Classic Actor must be clean at the exact Request baseCommit")
    return {"executable": executable, "sourceCommit": classic.EXPECTED_SOURCE,
            "heldSourceCommit": classic.EXPECTED_HELD_SOURCE, "packet": packet_path,
            "projectConfig": config, "runtime": runtime, "runtimeValue": runtime_value,
            "roleAuthorization": authorization, "roleAuthorizationValue": auth_value,
            "repository": repo, "baseCommit": head, "roles": roles}


def resolve_role(invocation: dict, auth: dict | None = None, runtime: dict | None = None) -> dict:
    """Project a native Classic Invocation to a stable middleware slot without rewriting it."""
    if not isinstance(invocation, dict) or invocation.get("apiVersion") != "markitect.example.org/agent-execution/v1alpha1":
        raise ValueError("pinned Classic agentexec Invocation required")
    request = invocation.get("request")
    if not isinstance(request, dict):
        raise ValueError("Classic agentexec Request required")
    key = (request.get("role"), request.get("projectionId"))
    resolved = ROLE_SLOTS.get(key)
    if resolved is None:
        raise ValueError("Classic role/projection is not an authorized fixture slot")
    phase, slot_id, scope_id = resolved
    scopes = request.get("scopeIds")
    if not isinstance(scopes, list) or scope_id not in scopes:
        raise ValueError("Classic Invocation is missing the exact configured scope identity")
    if runtime is not None:
        runner = runtime.get("executor" if request["role"] == "executor" else "verifier")
        if not isinstance(runner, dict):
            raise ValueError("Classic runtime does not configure the invoked response role")
    if auth is not None:
        slots = auth.get("slots")
        approved = [item for item in slots if isinstance(item, dict) and item.get("slotId") == slot_id] if isinstance(slots, list) else []
        if len(approved) != 1 or (approved[0].get("phase"), approved[0].get("responseRole")) != (phase, request["role"]):
            raise ValueError("Classic Invocation role/projection is not present in the approved role slots")
    # The controller's opaque native context/artifacts remain inputs as supplied;
    # this projection adds only the Scientist's authorization slot identity.
    return {"slotId": slot_id, "phase": phase, "responseRole": request["role"]}


def configured_roles(runtime: dict) -> list[dict]:
    """Expand shared-wrapper Executor/Verifier config into canonical slots."""
    if not isinstance(runtime, dict) or runtime.get("apiVersion") != "markitect.canonical/controller/v1alpha1":
        raise ValueError("pinned Classic controller runtime required")
    output = []
    for role, projections in (("executor", (DOTNET_PROJECTION,)),
                              ("verifier", (DOTNET_PROJECTION, MARKDOWN_PROJECTION))):
        runner = runtime.get(role)
        if (not isinstance(runner, dict) or not isinstance(runner.get("command"), str) or
                not isinstance(runner.get("args"), list) or not all(isinstance(arg, str) for arg in runner["args"])):
            raise ValueError("Classic runtime must pin shared-wrapper command/args for both native roles")
        for projection in projections:
            phase, slot_id, scope_id = ROLE_SLOTS[(role, projection)]
            output.append({"slotId": slot_id, "phase": phase, "responseRole": role,
                           "projectionId": projection, "scopeId": scope_id,
                           "command": runner["command"], "args": list(runner["args"]),
                           "runtimeFiles": json.loads(json.dumps(runner.get("runtimeFiles", []))),
                           "model": runner.get("model"), "modelOptions": runner.get("modelOptions"),
                           "providerVersion": runner.get("providerVersion"),
                           "delegate": json.loads(json.dumps(runner.get("delegate"))),
                           "runnerKey": role})
    return output


def invocation_call_id(dispatch_id: str, invocation: dict) -> str:
    """Return the S1 bridge's stable replay identity for a native invocation."""
    if not isinstance(dispatch_id, str) or not dispatch_id:
        raise ValueError("Classic dispatch identity required")
    if not all(isinstance(invocation.get(key), str) and invocation[key]
               for key in ("runId", "nonce", "inputDigest")):
        raise ValueError("Classic invocation replay identity is incomplete")
    return sha256(_json_bytes({"dispatchId": dispatch_id, "runId": invocation["runId"],
                               "nonce": invocation["nonce"], "inputDigest": invocation["inputDigest"]}))


def _mode(path: Path) -> str:
    import stat
    mode = stat.S_IMODE(path.stat().st_mode)
    if os.name == "nt":
        return "0644" if mode & stat.S_IWUSR else "0444"
    return f"{mode:04o}"


def _runtime_file(path: str | os.PathLike[str]) -> dict:
    target = Path(path)
    if not target.is_absolute() or target.is_symlink():
        raise ValueError("Classic runtime file must be absolute and non-symlink")
    target = target.resolve(strict=True)
    if not target.is_file():
        raise ValueError("Classic runtime file must be a regular file")
    return {"path": str(target), "mode": _mode(target), "digest": "sha256:" + sha256(target.read_bytes())}


def build_role_authorization(*, trial_id: str, dispatch_id: str, task_id: str,
                             request_path: str | os.PathLike[str], ledger_path: str | os.PathLike[str],
                             runtime_path: str | os.PathLike[str], role_evidence_directory: str | os.PathLike[str],
                             expires_at: int | float, packet_path: str | os.PathLike[str],
                             python_executable: str | os.PathLike[str] = sys.executable,
                             status: str = "pending") -> dict:
    """Build the exact three-slot Classic role-auth payload; never grants authority.

    The packet test-double is a pinned delegate under each slot. A caller must
    supply externally authorized identity, expiry, and status before launch.
    """
    if (not trial_id or not dispatch_id or not task_id or type(expires_at) not in (int, float) or
            not math.isfinite(expires_at) or expires_at <= 0):
        raise ValueError("finite trial, dispatch, task and authorization expiry are required")
    if status not in {"pending", "approved"}:
        raise ValueError("unsupported Classic role-authorization status")
    paths = [Path(value) for value in (request_path, ledger_path, runtime_path, role_evidence_directory)]
    if any(not value.is_absolute() for value in paths):
        raise ValueError("Classic role authorization paths must be absolute")
    python = Path(python_executable).resolve(strict=True)
    packet = Path(classic.inspect_packet(packet_path)["packetPath"])
    delegate = (packet / "smoke" / "protocol_test_double.py").resolve(strict=True)
    python_file, delegate_file = _runtime_file(python), _runtime_file(delegate)
    # Match role wrapper digest conventions and keep the exact artifact closure
    # visible in both delegate and wrapper RuntimeFile pins.
    command_digest = python_file["digest"]
    slots = []
    for (role, projection), (phase, slot_id, _scope) in ROLE_SLOTS.items():
        if role == "executor" and phase != "execute":
            raise AssertionError("Classic Executor slot phase drift")
        slots.append({"slotId": slot_id, "phase": phase, "responseRole": role,
                      "wrapper": {"command": str(python)},
                      "delegate": {"argv": [str(python), str(delegate)],
                                   "commandDigest": command_digest,
                                   "runtimeFiles": [python_file, delegate_file],
                                   "model": "protocol-test-double",
                                   "modelOptions": {"label": "packet protocol test double; no semantic judgment"},
                                   "providerVersion": "protocol-test-double/1",
                                   "timeoutSeconds": 20,
                                   "maxStdoutBytes": 1_048_576,
                                   "maxStderrBytes": 1_048_576}})
    import government_roles
    return {"apiVersion": "markitect.scientist-role-authorization/v1alpha1", "status": status,
            "trialId": trial_id, "dispatchId": dispatch_id, "taskId": task_id,
            "requestPath": str(paths[0].resolve()), "ledgerPath": str(paths[1].resolve()),
            "runtimePath": str(paths[2].resolve()), "roleEvidenceDirectory": str(paths[3].resolve()),
            "expiresAt": expires_at, "maxCalls": 12,
            "fixtureAuthorization": dict(government_roles.NATIVE_FIXTURE_AUTH), "slots": slots}


def prepare_fixture(packet_path: str | os.PathLike[str], destination: str | os.PathLike[str], *,
                    actor_repository: str | os.PathLike[str] | None = None,
                    reports_directory: str | os.PathLike[str] | None = None,
                    runtime_path: str | os.PathLike[str] | None = None,
                    authorization_path: str | os.PathLike[str] | None = None,
                    controller_state_directory: str | os.PathLike[str] | None = None,
                    request_path: str | os.PathLike[str] | None = None,
                    ledger_path: str | os.PathLike[str] | None = None,
                    role_evidence_directory: str | os.PathLike[str] | None = None,
                    trial_id: str = "native-fixture-classic-20261008",
                    dispatch_id: str = "classic-native-positive",
                    task_id: str = "native-classic-positive",
                    expires_at: int | float | None = None,
                    authorization_status: str = "approved") -> dict:
    """Clone and explicitly prepare the pinned public fixture outside this checkout.

    Only Git, Python fixture setup, and file writes are used. No product binary or
    role runner is launched. Destination must be absent and outside packet/source.
    """
    inspection = classic.inspect_packet(packet_path)
    if expires_at is None:
        expires_at = time.time() + 90 * 60
    packet = Path(inspection["packetPath"]).resolve(strict=True)
    pin = json.loads(Path(__file__).with_name("classic-pin.json").read_text(encoding="utf-8"))
    if packet != Path(pin["packet"]).resolve():
        raise ValueError("fixture packet differs from the held Classic readiness pin")
    destination = Path(destination)
    if not destination.is_absolute():
        raise ValueError("fixture destination must be absolute")
    destination = destination.resolve()
    checkout = Path(__file__).resolve().parents[3]
    source_bundle = (packet / "source" / "classic-source.bundle").resolve(strict=True)
    if (destination.exists() or destination == checkout or checkout in destination.parents or
            destination == packet or packet in destination.parents or
            destination == source_bundle or source_bundle in destination.parents):
        raise ValueError("fixture destination must be new and outside study checkout and packet")
    destination.mkdir(parents=True, exist_ok=False)
    repo = Path(actor_repository) if actor_repository is not None else destination / "actor"
    runtime_path = Path(runtime_path) if runtime_path is not None else destination / "released" / "runtime.json"
    reports = Path(reports_directory) if reports_directory is not None else destination / "results"
    released = destination / "released"
    released.mkdir(parents=True, exist_ok=True)
    if any(not path.is_absolute() for path in (repo, runtime_path, reports)):
        raise ValueError("Classic Actor, runtime and reports paths must be absolute")
    reports.mkdir(parents=True, exist_ok=False)
    clone = subprocess.run(["git", "clone", "--no-checkout", str(source_bundle), str(repo)],
                           capture_output=True, timeout=120, check=False)
    if clone.returncode:
        raise RuntimeError("pinned Classic fixture bundle clone failed: " + clone.stderr.decode(errors="replace")[-2000:])
    commands = []

    def git(name: str, *args: str, env: dict | None = None) -> str:
        result = subprocess.run(["git", "-C", str(repo), *args], capture_output=True,
                                timeout=30, check=False, env=env)
        commands.append({"name": name, "argv": ["git", "-C", str(repo), *args], "returnCode": result.returncode,
                         "stdoutSha256": sha256(result.stdout), "stderrSha256": sha256(result.stderr)})
        if result.returncode:
            raise RuntimeError(f"fixture git {name} failed: {result.stderr.decode(errors='replace')[-2000:]}")
        return result.stdout.decode("utf-8", errors="strict").strip()

    git("autocrlf", "config", "core.autocrlf", "false")
    git("checkout", "switch", "-c", "codex/classic-study-fixture", CLASSIC_SOURCE)
    setup_script = packet / "smoke" / "prepare_commerce.py"
    setup = subprocess.run([sys.executable, str(setup_script), "--repo", str(repo)],
                          capture_output=True, timeout=30, check=False)
    commands.append({"name": "fixture-setup", "argv": [sys.executable, str(setup_script), "--repo", str(repo)],
                     "returnCode": setup.returncode, "stdoutSha256": sha256(setup.stdout),
                     "stderrSha256": sha256(setup.stderr), "classification": "existing explicit release-test fixture preparation"})
    if setup.returncode:
        raise RuntimeError("pinned fixture preparation failed: " + setup.stderr.decode(errors="replace")[-2000:])
    preparation = json.loads(setup.stdout)
    commit_env = dict(os.environ, GIT_AUTHOR_NAME="Markitect Classic Fixture",
                      GIT_AUTHOR_EMAIL="classic-fixture@example.invalid",
                      GIT_COMMITTER_NAME="Markitect Classic Fixture",
                      GIT_COMMITTER_EMAIL="classic-fixture@example.invalid")
    git("stage-fixture", "add", "--", "examples/canonical-projection", env=commit_env)
    git("commit-fixture", "-c", "user.name=Markitect Classic Fixture", "-c",
        "user.email=classic-fixture@example.invalid", "-c", "commit.gpgsign=false", "commit",
        "-m", "Prepare established Markdown policy fixture", env=commit_env)
    source = git("source-revision", "rev-parse", "HEAD")
    if len(source) != 40 or git("clean-status", "status", "--porcelain", "--untracked-files=all"):
        raise RuntimeError("prepared Classic fixture is not a clean full-commit source")
    authorization_path = Path(authorization_path) if authorization_path is not None else released / "role-auth.json"
    request_path = Path(request_path) if request_path is not None else released / "request.json"
    ledger_path = Path(ledger_path) if ledger_path is not None else destination / "fixture-ledger.sqlite"
    role_evidence_directory = (Path(role_evidence_directory) if role_evidence_directory is not None
                               else destination / "role-evidence")
    if any(not path.is_absolute() for path in (authorization_path, request_path, ledger_path,
                                                 role_evidence_directory)):
        raise ValueError("Classic authorization, Request, ledger and role evidence paths must be absolute")
    authorization = build_role_authorization(
        trial_id=trial_id, dispatch_id=dispatch_id, task_id=task_id,
        request_path=request_path, ledger_path=ledger_path, runtime_path=runtime_path,
        role_evidence_directory=role_evidence_directory, expires_at=expires_at,
        packet_path=packet, status=authorization_status)
    auth_raw = _json_bytes(authorization) + b"\n"
    authorization_path.parent.mkdir(parents=True, exist_ok=True)
    with authorization_path.open("xb") as stream:
        stream.write(auth_raw)
        stream.flush()
        os.fsync(stream.fileno())
    runtime = prepare_protocol_runtime(packet, destination, repo,
                                      authorization_path=authorization_path,
                                      authorization_sha256=sha256(auth_raw),
                                      role_evidence_directory=role_evidence_directory,
                                      state_directory=controller_state_directory,
                                      runtime_path=runtime_path)
    plan = plan_initial_flow(packet, repo, runtime_path, reports, source)
    plan_path = destination / "native-plan.json"
    plan_path.write_text(json.dumps(plan, indent=2) + "\n", encoding="utf-8")
    return {"kind": "classic-fixture-preparation", "productSource": CLASSIC_SOURCE,
            "heldSource": pin["heldSourceSha"], "runtimeBinarySha256": inspection["binary"]["sha256"],
            "fixtureRepo": str(repo), "fixtureSourceCommit": source, "runtimePath": str(runtime_path),
            "roleAuthorizationPath": str(authorization_path), "roleAuthorizationSha256": sha256(auth_raw),
            "reportsDirectory": str(reports), "preparation": preparation, "commands": commands,
            "planPath": str(plan_path), "planSha256": sha256(plan_path.read_bytes()),
            "nativeOrActorProcessesStarted": False,
            "classification": "prepared disposable fixture and prospective native plan only"}


def prepare_protocol_runtime(packet_path: str | os.PathLike[str], destination: str | os.PathLike[str],
                             repo: str | os.PathLike[str], *, authorization_path: str | os.PathLike[str],
                             authorization_sha256: str,
                             role_evidence_directory: str | os.PathLike[str],
                             state_directory: str | os.PathLike[str] | None = None,
                             runtime_path: str | os.PathLike[str] | None = None) -> Path:
    """Write a wrapper-pinned native runtime with test-double delegates behind it."""
    inspection = classic.inspect_packet(packet_path)
    packet = Path(inspection["packetPath"])
    destination = Path(destination).resolve()
    repo = Path(repo).resolve(strict=True)
    state = Path(state_directory) if state_directory is not None else destination / "controller-evidence"
    if not state.is_absolute():
        raise ValueError("Classic controller state directory must be absolute")
    (state / "ledger").mkdir(parents=True, exist_ok=False)
    (state / "private-logs").mkdir(parents=True, exist_ok=False)
    double = packet / "smoke" / "protocol_test_double.py"
    raw = double.read_bytes()
    digest = "sha256:" + sha256(raw)
    auth_path = Path(authorization_path).resolve(strict=True)
    auth_raw = auth_path.read_bytes()
    if sha256(auth_raw) != authorization_sha256:
        raise ValueError("Classic role authorization digest mismatch before runtime creation")
    evidence = Path(role_evidence_directory)
    if not evidence.is_absolute():
        raise ValueError("Classic role evidence directory must be absolute")
    import government_roles
    wrapper = Path(government_roles.__file__).resolve(strict=True)
    native_controller = Path(__file__).with_name("native_controller.py").resolve(strict=True)
    helper_dir = Path(__file__).resolve().parent
    bridge_files = [_runtime_file(helper_dir / name) for name in BRIDGE_FILES]
    bridge_files += [_runtime_file(wrapper), _runtime_file(native_controller), _runtime_file(auth_path),
                     _runtime_file(Path(sys.executable).resolve(strict=True)), _runtime_file(double.resolve(strict=True))]
    bridge_files = list({item["path"]: item for item in bridge_files}.values())
    run_config = {"command": str(Path(sys.executable).resolve(strict=True)),
                  "args": government_roles.wrapper_arguments(str(wrapper), str(auth_path),
                                                             authorization_sha256, str(evidence)),
                  "model": "protocol-test-double",
                  "modelOptions": {"label": "packet protocol test double; no semantic judgment"},
                  "providerVersion": "protocol-test-double/1", "timeoutSeconds": 20,
                  "maxStdoutBytes": 1048576, "maxStderrBytes": 1048576,
                  "runtimeFiles": bridge_files}
    check = {"name": "canonical-projection-fixture", "run": ["go", "run", "examples/canonical-projection/evidence/check.go"]}
    runtime = {"apiVersion": "markitect.canonical/controller/v1alpha1",
               "recordStore": str((state / "ledger").resolve()), "privateLogs": str((state / "private-logs").resolve()),
               "referenceDepth": 1, "auditAll": True,
               "checkInputs": ["examples/canonical-projection/evidence/check.go"],
               "executor": json.loads(json.dumps(run_config)), "verifier": json.loads(json.dumps(run_config)),
               "assuranceRoots": ["commerce-markdown"],
               "assuranceScopes": [
                   {"id": "commerce-markdown", "projectionId": MARKDOWN_PROJECTION,
                    "children": ["commerce-dotnet"], "checks": [check],
                    "checkInputs": ["examples/canonical-projection/evidence/check.go"]},
                   {"id": "commerce-dotnet", "projectionId": DOTNET_PROJECTION,
                    "children": [], "checks": [check],
                    "checkInputs": ["examples/canonical-projection/evidence/check.go"]}]}
    path = Path(runtime_path) if runtime_path is not None else destination / "released" / "runtime.json"
    if not path.is_absolute():
        raise ValueError("Classic runtime output path must be absolute")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("x", encoding="utf-8", newline="\n") as stream:
        stream.write(json.dumps(runtime, indent=2) + "\n")
        stream.flush()
        os.fsync(stream.fileno())
    return path


def plan_initial_flow(packet_path: str | os.PathLike[str], repo_path: str | os.PathLike[str],
                      runtime_path: str | os.PathLike[str], reports_directory: str | os.PathLike[str],
                      revision: str) -> dict:
    """Plan only the necessary positive Execute call; never start the pinned binary."""
    inspection = classic.inspect_packet(packet_path)
    binary = inspection["binary"]["path"]
    repo, runtime, reports = Path(repo_path).resolve(strict=True), Path(runtime_path).resolve(strict=True), Path(reports_directory).resolve()
    if len(revision) != 40 or any(c not in "0123456789abcdef" for c in revision):
        raise ValueError("full prepared fixture commit required")
    if reports == repo or repo in reports.parents or runtime == repo or repo in runtime.parents:
        raise ValueError("runtime and reports must stay outside fixture repository")
    config = "examples/canonical-projection/canonical.yaml"
    def step(action, spec, report_name):
        return {"action": action, "argv": spec["argv"], "cwd": spec["cwd"],
                "expectedReport": spec["expectedReport"], "timeoutSeconds": spec["timeoutSeconds"],
                "automaticRetries": spec["automaticRetries"],
                "stdoutPath": str((reports / (report_name + (".yaml" if spec["expectedReport"] == "yaml" else ".json"))).resolve()),
                "stderrPath": str((reports / (report_name + ".stderr.txt")).resolve())}
    spec = classic.native_command("execute", binary_path=binary, repo_path=repo, config_path=config,
                                  revision=revision, runtime_path=runtime)
    calls = [step("execute", spec, "execute")]
    return {"schemaVersion": 1, "kind": "prospective-classic-native-plan", "product": "Markitect Classic 0.14.1",
            "productSourceSha": classic.EXPECTED_SOURCE, "binarySha256": classic.EXPECTED_BINARY_SHA256,
            "fixtureSourceCommit": revision, "condition": "disposable-prepared-release-fixture",
            "delegate": "shared Scientist role wrapper with packet protocol_test_double.py as the pinned deterministic delegate; not a model/verifier/human",
            "calls": calls, "afterExecute": {"executeReportPath": str((reports / "execute.json").resolve()),
                            "review": "external reviewer must bind both exact Execute stdout SHA-256 and report.digest before guarded Apply"},
            "noNativeCallPerformed": True}


def review_execute(execute_report_path: str | os.PathLike[str], review: dict) -> str:
    """Check an external review record is bound to exact Execute report bytes/digest."""
    path = Path(execute_report_path).resolve(strict=True)
    raw = path.read_bytes()
    report = json.loads(raw)
    execute_digest = report.get("digest")
    if report.get("status") != "planned" or not isinstance(execute_digest, str) or not execute_digest.startswith("sha256:"):
        raise ValueError("Execute report must be planned and carry its native digest")
    if (not isinstance(review, dict) or review.get("status") != "approved" or
            review.get("executeReportSha256") != sha256(raw) or
            review.get("executeDigest") != execute_digest or
            not isinstance(review.get("reviewerReference"), str) or not review["reviewerReference"].strip()):
        raise ValueError("external review must bind the exact Execute report and native digest")
    return execute_digest


def plan_native_action(action: str, *, packet_path: str | os.PathLike[str],
                       repo_path: str | os.PathLike[str], runtime_path: str | os.PathLike[str],
                       reports_directory: str | os.PathLike[str], revision: str,
                       execute_report_path: str | os.PathLike[str] | None = None,
                       review: dict | None = None,
                       apply_result_path: str | os.PathLike[str] | None = None) -> dict:
    """Plan one required native action, including its exact report capture paths."""
    if action not in {"execute", "apply", "verify", "audit", "apply-replay"}:
        raise ValueError("only the bounded Classic positive-flow actions are supported")
    inspection = classic.inspect_packet(packet_path)
    binary = inspection["binary"]["path"]
    repo = Path(repo_path).resolve(strict=True)
    runtime = Path(runtime_path).resolve(strict=True)
    reports = Path(reports_directory).resolve()
    if (runtime == repo or repo in runtime.parents or reports == repo or repo in reports.parents or
            reports == runtime or runtime in reports.parents or reports in runtime.parents):
        raise ValueError("Classic runtime and report outputs must remain outside the Actor repository and each other")
    if len(revision) != 40 or any(c not in "0123456789abcdef" for c in revision):
        raise ValueError("full prepared fixture commit required")
    config = "examples/canonical-projection/canonical.yaml"
    if action == "execute":
        spec = classic.native_command("execute", binary_path=binary, repo_path=repo,
                                      config_path=config, revision=revision, runtime_path=runtime)
    elif action in {"apply", "apply-replay"}:
        if execute_report_path is None or review is None:
            raise ValueError("guarded Apply requires an exact Execute report and external review")
        reviewed = review_execute(execute_report_path, review)
        spec = classic.native_command("apply", binary_path=binary, repo_path=repo, config_path=config,
                                      revision=revision, runtime_path=runtime,
                                      execute_report=execute_report_path, reviewed_digest=reviewed)
    elif action == "verify":
        if apply_result_path is None:
            raise ValueError("fresh Verify requires the exact saved Apply result")
        apply_result = Path(apply_result_path).resolve(strict=True)
        spec = {"argv": [binary, "canonical", "--repo", str(repo), "--config", config,
                          "--runtime", str(runtime), "--action", "controller-verify",
                          "--apply-result", str(apply_result), "--write"],
                "cwd": str(repo), "expectedReport": "json", "timeoutSeconds": classic.MAX_CALL_SECONDS,
                "automaticRetries": 0}
    else:
        spec = classic.native_command("audit", binary_path=binary, repo_path=repo, config_path=config,
                                      revision=revision, runtime_path=runtime)
    suffix = ".json"
    return {"action": action, "argv": spec["argv"], "cwd": spec["cwd"],
            "expectedReport": spec["expectedReport"], "timeoutSeconds": spec["timeoutSeconds"],
            "automaticRetries": 0,
            "stdoutPath": str((reports / (action + suffix)).resolve()),
            "stderrPath": str((reports / (action + ".stderr.txt")).resolve())}


def plan_guarded_flow(packet_path: str | os.PathLike[str], repo_path: str | os.PathLike[str],
                      runtime_path: str | os.PathLike[str], reports_directory: str | os.PathLike[str],
                      revision: str, execute_report_path: str | os.PathLike[str], review: dict) -> dict:
    """Plan reviewed Apply, fresh Verify, Audit, and an explicit replay attempt."""
    execute_digest = review_execute(execute_report_path, review)
    inspection = classic.inspect_packet(packet_path)
    binary = inspection["binary"]["path"]
    repo, runtime, reports = Path(repo_path).resolve(strict=True), Path(runtime_path).resolve(strict=True), Path(reports_directory).resolve()
    config = "examples/canonical-projection/canonical.yaml"
    apply_spec = classic.native_command("apply", binary_path=binary, repo_path=repo, config_path=config,
                                        revision=revision, runtime_path=runtime,
                                        execute_report=execute_report_path, reviewed_digest=execute_digest)
    # The CLI requires the exact saved Apply result to exist before Verify is
    # planned. Keep the already documented argv target explicit without
    # manufacturing a placeholder Apply result during preparation.
    verify_spec = {"argv": [binary, "canonical", "--repo", str(repo), "--config", config,
                             "--runtime", str(runtime), "--action", "controller-verify",
                             "--apply-result", str((reports / "apply.json").resolve()), "--write"],
                   "cwd": str(repo), "expectedReport": "json", "timeoutSeconds": classic.MAX_CALL_SECONDS,
                   "automaticRetries": 0}
    audit_spec = classic.native_command("audit", binary_path=binary, repo_path=repo, config_path=config,
                                       revision=revision, runtime_path=runtime)
    replay_spec = classic.native_command("apply", binary_path=binary, repo_path=repo, config_path=config,
                                         revision=revision, runtime_path=runtime,
                                         execute_report=execute_report_path, reviewed_digest=execute_digest)
    def item(action, spec):
        return {"action": action, "argv": spec["argv"], "cwd": spec["cwd"],
                "expectedReport": spec["expectedReport"], "timeoutSeconds": spec["timeoutSeconds"],
                "automaticRetries": spec["automaticRetries"],
                "stdoutPath": str((reports / (action + ".json")).resolve()),
                "stderrPath": str((reports / (action + ".stderr.txt")).resolve())}
    return {"schemaVersion": 1, "kind": "review-gated-classic-native-plan",
            "executeReportPath": str(Path(execute_report_path).resolve()),
            "executeReportSha256": review["executeReportSha256"], "reviewerReference": review["reviewerReference"],
            "reviewedDigest": execute_digest,
            "calls": [item("apply", apply_spec), item("verify", verify_spec), item("audit", audit_spec)],
            "targetedReplay": {**item("apply-replay", replay_spec),
                               "expectedBehavior": "original Execute report is stale after successful Apply advances the ledger head; native Apply must refuse it and must not materialize a second write",
                               "notStarted": True},
            "noNativeCallPerformed": True}


def persist_native_capture(action: str, planned_step: dict, capture: dict) -> dict:
    """Persist a future `classic.run_native` result at the plan's exact paths.

    The caller, not this function, performs any authorized product launch. Files
    are created exclusively so a retry cannot overwrite prior command evidence.
    """
    if action not in FLOW_ACTIONS or not isinstance(capture, dict):
        raise ValueError("known Classic action and native capture required")
    if (capture.get("argv") != planned_step.get("argv") or capture.get("cwd") != planned_step.get("cwd") or
            capture.get("productSourceSha") != classic.EXPECTED_SOURCE or
            capture.get("runtimeSha256") != classic.EXPECTED_BINARY_SHA256 or
            not isinstance(capture.get("stdout"), bytes) or not isinstance(capture.get("stderr"), bytes) or
            type(capture.get("returnCode")) is not int):
        raise ValueError("native capture does not match the pinned planned invocation")
    stdout_path, stderr_path = Path(planned_step["stdoutPath"]), Path(planned_step["stderrPath"])
    if not stdout_path.is_absolute() or not stderr_path.is_absolute() or stdout_path.parent != stderr_path.parent:
        raise ValueError("planned stdout/stderr must be absolute paths in one external reports directory")
    if stdout_path.exists() or stderr_path.exists():
        raise FileExistsError("refusing to overwrite prior Classic command capture")
    stdout_path.parent.mkdir(parents=True, exist_ok=True)
    for path, raw in ((stdout_path, capture["stdout"]), (stderr_path, capture["stderr"])):
        with path.open("xb") as stream:
            stream.write(raw)
            stream.flush()
            os.fsync(stream.fileno())
    process_value = {key: capture[key] for key in ("argv", "cwd", "returnCode", "wallSeconds", "timedOut",
                                                   "processTreeControl", "stopReason", "productSourceSha", "runtimeSha256")
                     if key in capture}
    process_path = stdout_path.with_name(action + ".process.json")
    process_raw = (json.dumps(process_value, indent=2) + "\n").encode("utf-8")
    with process_path.open("xb") as stream:
        stream.write(process_raw)
        stream.flush()
        os.fsync(stream.fileno())
    return {"action": action, "expectedReport": planned_step["expectedReport"],
            "argv": planned_step["argv"], "cwd": planned_step["cwd"],
            "returnCode": capture["returnCode"], "stdoutPath": str(stdout_path.resolve()),
            "stderrPath": str(stderr_path.resolve()), "processPath": str(process_path.resolve()),
            "processSha256": sha256(process_raw)}


def normalize_result(request_raw: bytes, captures: list[dict], *, inference_performed: bool | None = None) -> dict:
    """Normalize persisted native stdout/stderr captures to the study Result shape.

    This function reports technical controller status only. It never sets a
    candidate commit and never describes protocol-test-double output as accepted.
    """
    request = json.loads(request_raw)
    if request.get("arm") != "classic" or request.get("schemaVersion") != 1:
        raise ValueError("Classic study Request bytes required")
    by_action = {row.get("action"): row for row in captures if isinstance(row, dict)}
    receipts, reports = [], {}
    errors = []
    for action in FLOW_ACTIONS:
        row = by_action.get(action)
        if row is None:
            errors.append("missing persisted native capture: " + action)
            continue
        for label in ("stdoutPath", "stderrPath"):
            path = Path(row.get(label, ""))
            if not path.is_absolute() or not path.is_file():
                errors.append("missing native capture file: " + action + "/" + label)
                continue
            raw = path.read_bytes()
            receipts.append({"path": str(path.resolve()), "sha256": sha256(raw),
                             "kind": "classic-native-" + action + ("-stdout" if label == "stdoutPath" else "-stderr")})
            if label == "stdoutPath" and row.get("expectedReport") in {"json", "text"}:
                if row["expectedReport"] == "json":
                    try:
                        reports[action] = json.loads(raw)
                    except (UnicodeDecodeError, json.JSONDecodeError):
                        errors.append("invalid native JSON report: " + action)
                else:
                    reports[action] = raw.decode("utf-8", errors="replace").strip()
        process_path = row.get("processPath")
        if process_path is not None:
            process_file = Path(process_path)
            if not process_file.is_absolute() or not process_file.is_file():
                errors.append("missing native process receipt: " + action)
            else:
                process_raw = process_file.read_bytes()
                if row.get("processSha256") != sha256(process_raw):
                    errors.append("native process receipt digest mismatch: " + action)
                try:
                    process_value = json.loads(process_raw)
                    if process_value.get("argv") != row.get("argv") or process_value.get("cwd") != row.get("cwd"):
                        errors.append("native process receipt invocation mismatch: " + action)
                    if (process_value.get("productSourceSha") != classic.EXPECTED_SOURCE or
                            process_value.get("runtimeSha256") != classic.EXPECTED_BINARY_SHA256):
                        errors.append("native process receipt source/binary pin mismatch: " + action)
                except (UnicodeDecodeError, json.JSONDecodeError):
                    errors.append("invalid native process receipt: " + action)
                receipts.append({"path": str(process_file.resolve()), "sha256": sha256(process_raw),
                                 "kind": "classic-native-" + action + "-process"})
        else:
            errors.append("missing native process receipt: " + action)
        if type(row.get("returnCode")) is not int or (action != "apply-replay" and row.get("returnCode") != 0):
            errors.append("native command returned nonzero: " + action)
    for action, expected in EXPECTED_STATUSES.items():
        value = reports.get(action)
        if not isinstance(value, dict) or value.get("status") != expected:
            errors.append("native controller status mismatch: " + action)
    audit = reports.get("audit")
    if isinstance(audit, dict) and (audit.get("findings") or audit.get("nextSteps")):
        errors.append("native audit reports findings or next steps")
    replay = reports.get("apply-replay")
    if not isinstance(replay, dict) or replay.get("status") not in {"blocked", "refused", "stale", "rejected"}:
        errors.append("targeted stale Apply replay refusal was not explicitly reported")
    elif replay.get("materialized") is True or replay.get("writeApplied") is True:
        errors.append("targeted stale Apply replay reports a second materialization")
    result = {"schemaVersion": 1, "trialId": request.get("trialId"), "requestSha256": sha256(request_raw),
              "operation": request.get("operation"), "mode": request.get("mode"),
              "status": "completed" if not errors else "incomplete", "candidateCommit": None,
              "capabilities": ["pinned Classic controller flow executed against disposable fixture"],
              "gaps": ["technical fixture flow is not semantic acceptance or human approval"],
              "receipts": receipts, "usage": None, "inferencePerformed": inference_performed,
              "classic": {"productSourceSha": classic.EXPECTED_SOURCE,
                          "binarySha256": classic.EXPECTED_BINARY_SHA256,
                          "reports": {action: value for action, value in reports.items() if isinstance(value, dict)},
                          "errors": errors,
                          "semanticAcceptance": False, "humanAcceptance": False}}
    if errors:
        result["gaps"].extend(errors)
    return result
