"""One separately ordered assessment of an existing immutable candidate."""
from __future__ import annotations

import argparse
from datetime import datetime, timedelta, timezone
import hashlib
import json
from pathlib import Path
import subprocess
import uuid

ROOT = Path(__file__).resolve().parent
ORDER_ID = "conventional-frozen-candidate-scratch-assessment-supplement-20261009"
ORDER_HASH = "22d7d1297fa59b1e48391b5464bb0163d98a403ff195e5e867c05c7f67017485"
EXE_HASH = "3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68"
DESTINATION = Path("C:/Users/Consiliari/Documents/Conventional-Frozen-Candidate-Scratch-Assessment-Supplement-20261009")
LEDGER = Path(str(DESTINATION) + ".reservation.json")


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def save(path, value):
    Path(path).write_bytes((json.dumps(value, ensure_ascii=False, indent=2) + "\n").encode("utf-8"))


def manifest(root):
    result = {}
    for path in sorted(Path(root).rglob("*")):
        if path.is_symlink():
            raise ValueError("links are not supported in frozen assessment inputs")
        if path.is_file():
            result[path.relative_to(root).as_posix()] = digest(path)
    return result


def verify(plan, executable):
    """File-only admission; consumes no reservation or native start."""
    if plan.get("id") != ORDER_ID or plan.get("schema") != 1:
        raise ValueError("unknown supplemental assessment order")
    authority = plan["authority"]
    raw = Path(authority["orderPath"]).read_bytes()
    if hashlib.sha256(raw).hexdigest() != ORDER_HASH or authority["orderSha256"] != ORDER_HASH:
        raise ValueError("supplement authority bytes changed")
    order = json.loads(raw)
    if order["id"] != ORDER_ID:
        raise ValueError("supplement order id changed")
    if (Path(plan["destination"]).resolve() != DESTINATION.resolve() or
            Path(plan["ledger"]).resolve() != LEDGER.resolve()):
        raise ValueError("owned supplemental destination changed")
    if LEDGER.exists() or DESTINATION.exists():
        raise ValueError("supplement is reserved or its destination already exists; no retry")
    if digest(executable) != EXE_HASH or plan["executableSha256"] != EXE_HASH:
        raise ValueError("existing pinned executable changed")
    cfg = plan["choice"]["config"]
    if (plan["choice"]["case"] != "readinglog" or plan["choice"]["method"] != "Conventional" or
            cfg["backend"] != "codex-app-server" or cfg["model"] != "gpt-6-luna" or cfg["effort"] != "high" or
            cfg["runtimeOptions"] != {"sandbox": "workspace-write", "approvalPolicy": "never",
                "memoryEnabled": False, "nativeHelperModel": "gpt-6-luna", "nativeHelperEffort": "high",
                "windowsSandbox": "mxc", "nativeMaxConcurrentAgents": 1, "allowLoginShell": False}):
        raise ValueError("supplemental native runtime changed")
    limits = order["limits"]
    if plan["limits"] != limits:
        raise ValueError("supplemental finite limits changed")
    current = datetime.now(timezone.utc)
    if current >= datetime.fromisoformat(limits["overallAbsoluteEndUtc"].replace("Z", "+00:00")):
        raise ValueError("supplement absolute deadline exhausted")
    issued = datetime.fromisoformat(order["issuedUtc"].replace("Z", "+00:00"))
    if current >= issued + timedelta(seconds=limits["sourcePreparationSeconds"]):
        raise ValueError("supplement source preparation allowance exhausted")
    candidate_order = order["candidate"]
    snapshot_path = Path(plan["candidateSnapshotPath"])
    if (str(snapshot_path).replace("\\", "/") != candidate_order["snapshotPath"] or
            digest(snapshot_path) != candidate_order["snapshotSha256"] or
            plan["candidateSnapshotSha256"] != candidate_order["snapshotSha256"]):
        raise ValueError("original frozen candidate snapshot binding changed")
    snapshot = json.loads(snapshot_path.read_bytes())
    info = snapshot["assessmentCandidate"]
    candidate = snapshot_path.parent / info["path"]
    if (snapshot["completionTarget"] != "workspace_snapshot" or info["kind"] != "workspace_snapshot" or
            manifest(candidate) != info["manifest"] or plan["candidateManifest"] != info["manifest"]):
        raise ValueError("original frozen candidate bytes changed")
    original_result = snapshot_path.parent.parent / "independent-final-assessment/result.json"
    if digest(original_result) != candidate_order["originalAssessmentResultSha256"]:
        raise ValueError("original assessment result changed")
    initial_binding_path = snapshot_path.parent.parent / "initial-public-binding.json"
    initial_public = initial_binding_path.parent / "initial-public"
    initial_binding = json.loads(initial_binding_path.read_bytes())
    initial_manifest = manifest(initial_public)
    logical = hashlib.sha256((json.dumps(initial_manifest, sort_keys=True, separators=(",", ":")) + "\n").encode()).hexdigest()
    if (initial_binding["kind"] != "prepared_public_workspace" or
            initial_binding["fileManifest"] != initial_manifest or initial_binding["fileManifestSha256"] != logical or
            plan["initialPublicBindingSha256"] != digest(initial_binding_path) or
            plan["initialPublicManifest"] != initial_manifest):
        raise ValueError("original pre-actor public requirements changed")
    required_sources = {"run_scratch_assessment.py", "final_scratch_assessor.py",
                        "conventional/backends.py", "conventional/approvals.py"}
    if set(plan["sourcePins"]) != required_sources:
        raise ValueError("supplement source pin set changed")
    if plan["originalSourceCommit"] != candidate_order["actualSource"]:
        raise ValueError("original implementation source attribution changed")
    for name, pin in plan["sourcePins"].items():
        path = ROOT / name
        if Path(name).is_absolute() or ".." in Path(name).parts or digest(path) != pin:
            raise ValueError("supplement source pin changed")
    return raw, candidate, initial_public, initial_binding_path


def reserve(plan):
    with LEDGER.open("xb") as stream:
        stream.write((json.dumps({"order": ORDER_ID, "reservedAt": datetime.now(timezone.utc).isoformat(),
            "planSha256": hashlib.sha256(json.dumps(plan, sort_keys=True).encode()).hexdigest(),
            "scope": "one final assessor only; retained on every disposition"}, indent=2) + "\n").encode())


def sandbox_check(executable, cwd, audit, deadline):
    """One model-free native check of ordinary scratch and outside-root control."""
    marker = uuid.uuid4().hex
    control_dir = audit / "readonly-control"
    control_dir.mkdir()
    control = control_dir / "sentinel.txt"
    control.write_bytes(b"readonly-control")
    probe = cwd / ".scratch" / ("runtime-probe-" + marker + ".txt")
    def ps(path):
        return str(path).replace("'", "''")
    code = ("$ErrorActionPreference='Stop';"
        f"$p='{ps(probe)}';$control='{ps(control)}';"
        f"[IO.File]::WriteAllText($p,'{marker}');"
        f"if([IO.File]::ReadAllText($p) -ne '{marker}'){{exit 41}};"
        "Write-Output 'SCRATCH_WRITE_READ_OK';"
        "if([IO.File]::ReadAllText($control) -ne 'readonly-control'){exit 42};"
        "$denied=$false;try{[IO.File]::WriteAllText($control,'unexpected-write')}catch{$denied=$true};"
        "if(-not $denied){exit 43};Write-Output 'OUTSIDE_ROOT_WRITE_DENIED';"
        "Remove-Item -LiteralPath $p -Force;Write-Output 'SCRATCH_PROBE_OK';")
    args = [str(executable), "-c", 'windows.sandbox="mxc"', "sandbox", "--include-managed-config",
            "--permission-profile", ":workspace", "--", "powershell.exe", "-NoProfile", "-NonInteractive",
            "-Command", code]
    timeout = min(600, (deadline - datetime.now(timezone.utc)).total_seconds())
    if timeout <= 0:
        return {"state": "blocked", "reason": "supplement deadline exhausted before startup check"}
    started = datetime.now(timezone.utc).isoformat()
    try:
        output = subprocess.run(args, cwd=cwd, capture_output=True, timeout=timeout)
        stdout = output.stdout.decode("utf-8", "replace")
        ready = (output.returncode == 0 and "SCRATCH_WRITE_READ_OK" in stdout and "SCRATCH_PROBE_OK" in stdout and
                 "OUTSIDE_ROOT_WRITE_DENIED" in stdout and control.read_bytes() == b"readonly-control" and not probe.exists())
        result = {"state": "ready" if ready else "blocked", "exitCode": output.returncode,
                  "stdout": stdout, "stderr": output.stderr.decode("utf-8", "replace")}
    except subprocess.TimeoutExpired:
        result = {"state": "uncertain", "reason": "startup check timeout; no assessor start or retry",
                  "ownership": "direct child termination attempted by subprocess.run; descendant state unknown"}
    result.update(startedAt=started, endedAt=datetime.now(timezone.utc).isoformat(),
        argv=args, cwd=str(cwd), modelStarted=False, checkCount=1,
        scope="owned temporary probe and outside-root fixture only; original candidate never probed for writes")
    save(audit / "startup-check.json", result)
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--plan", type=Path, required=True)
    parser.add_argument("--executable", type=Path, required=True)
    args = parser.parse_args(argv)
    plan = json.loads(args.plan.read_bytes())
    git = subprocess.run(["git", "-C", str(ROOT), "status", "--porcelain"], check=True, capture_output=True)
    if git.stdout.strip():
        raise ValueError("source checkout must be clean before supplemental start")
    authority, candidate, initial_public, initial_binding_path = verify(plan, args.executable)
    source = subprocess.run(["git", "-C", str(ROOT), "rev-parse", "HEAD"], check=True, capture_output=True).stdout.decode().strip()
    reserve(plan)
    DESTINATION.mkdir()
    audit = DESTINATION / "audit"
    audit.mkdir()
    (DESTINATION / "delivery-order.json").write_bytes(authority)
    deadline = datetime.fromisoformat(plan["limits"]["overallAbsoluteEndUtc"].replace("Z", "+00:00"))
    binding = {"schema": 1, "sourceCommit": source, "planSha256": digest(args.plan), "plan": plan,
               "executable": str(args.executable), "executableSha256": digest(args.executable),
               "startedAt": datetime.now(timezone.utc).isoformat(), "originalCandidate": str(candidate),
               "originalRequirements": str(initial_public), "originalInitialBindingSha256": digest(initial_binding_path)}
    save(DESTINATION / "binding.json", binding)
    from final_scratch_assessor import prepare_scratch, assess_scratch
    result = {"state": "preparing", "authority": ORDER_ID, "originalOrder": "closed; unchanged"}
    try:
        prepared = prepare_scratch(candidate, audit, initial_public, plan["candidateManifest"], plan["initialPublicManifest"])
        readiness = sandbox_check(args.executable, Path(prepared["executionCandidate"]), audit, deadline)
        result["startupCheck"] = readiness
        if readiness["state"] != "ready":
            result.update(state="uncertain" if readiness["state"] == "uncertain" else "blocked",
                          reason="scratch runtime readiness failed; no assessor start")
        else:
            expires = min(deadline - timedelta(seconds=plan["limits"]["captureSeconds"]),
                          datetime.now(timezone.utc) + timedelta(seconds=plan["limits"]["assessorRoleSeconds"]))
            print(json.dumps({"event": "supplemental-fresh-assessor-start", "sourceCommit": source}), flush=True)
            assessed = assess_scratch(plan, plan["choice"], args.executable, candidate, audit, expires,
                                     initial_public, plan["candidateManifest"], plan["initialPublicManifest"])
            result.update(state=assessed.get("state", "uncertain"), assessment=assessed,
                          claimScope="native terminal and unchanged inputs only; scientific findings in separate report")
    except Exception as exc:
        result.update(state="blocked", reason=f"{type(exc).__name__}: {exc}", retry="forbidden")
    result.update(endedAt=datetime.now(timezone.utc).isoformat(),
        originalCandidateUnchanged=manifest(candidate) == plan["candidateManifest"],
        originalRequirementsUnchanged=manifest(initial_public) == plan["initialPublicManifest"],
        originalInitialBindingUnchanged=digest(initial_binding_path) == binding["originalInitialBindingSha256"])
    if not all(result[key] is True for key in ("originalCandidateUnchanged", "originalRequirementsUnchanged", "originalInitialBindingUnchanged")):
        result.update(state="failed", reason="original frozen input bytes changed")
    save(DESTINATION / "result.json", result)
    print(json.dumps({"event": "supplemental-assessment-ended", "state": result["state"], "destination": str(DESTINATION)}), flush=True)
    return 0 if result["state"] == "completed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
